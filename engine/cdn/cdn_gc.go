package cdn

import (
	"context"
	"math/rand"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

const (
	ItemLogGC = 24 * 3600
)

var (
	CleanUnitLockKey = cache.Key("cdn", "clean", "unit", "lock")
)

func (s *Service) itemPurge(ctx context.Context) {
	tickPurge := time.NewTicker(1 * time.Minute)
	tickPurgeStorageUnit := time.NewTicker(1 * time.Hour)
	defer tickPurge.Stop()
	defer tickPurgeStorageUnit.Stop()
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "cdn:ItemPurge: %v", ctx.Err())
			}
			return
		case <-tickPurge.C:
			if err := s.cleanItemToDelete(ctx); err != nil {
				ctx = sdk.ContextWithStacktrace(ctx, err)
				log.Error(ctx, "cdn:ItemPurge: error on cleanItemToDelete: %v", err)
			}
		case <-tickPurgeStorageUnit.C:
			if err := s.cleanStorageUnit(ctx); err != nil {
				ctx = sdk.ContextWithStacktrace(ctx, err)
				log.Error(ctx, "cdn:ItemPurge: error on cleanStorageUnit: %v", err)
			}

		}
	}
}

// ItemsGC clean long incoming item + delete item from buffer when synchronized everywhere
func (s *Service) itemsGC(ctx context.Context) {
	tickGC := time.NewTicker(1 * time.Minute)
	defer tickGC.Stop()
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "cdn:CompleteWaitingItems: %v", ctx.Err())
			}
			return
		case <-tickGC.C:
			if err := s.cleanBuffer(ctx); err != nil {
				ctx = sdk.ContextWithStacktrace(ctx, err)
				log.Error(ctx, "cdn:CompleteWaitingItems: cleanBuffer err: %v", err)
			}
			if err := s.cleanWaitingItem(ctx, ItemLogGC); err != nil {
				ctx = sdk.ContextWithStacktrace(ctx, err)
				log.Error(ctx, "cdn:CompleteWaitingItems: ContextWithStacktrace err: %v", err)
			}
		}
	}
}

func (s *Service) markUnitItemToDeleteByItemID(ctx context.Context, itemID string) (int, error) {
	db := s.mustDBWithCtx(ctx)
	itemUnitIDs, err := storage.LoadAllItemUnitsIDsByItemID(db, itemID)
	if err != nil {
		return 0, err
	}
	if len(itemUnitIDs) == 0 {
		return 0, nil
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, sdk.WithStack(err)
	}

	defer tx.Rollback() // nolint

	n, err := storage.MarkItemUnitToDelete(tx, itemUnitIDs)
	if err != nil {
		return 0, err
	}

	return n, sdk.WithStack(tx.Commit())
}

func (s *Service) cleanStorageUnit(ctx context.Context) error {
	units, err := storage.LoadAllUnits(ctx, s.Mapper, s.mustDBWithCtx(ctx))
	if err != nil {
		return err
	}
	for _, unit := range units {
		if !unit.ToDelete {
			continue
		}
		// Take a lock, 1 unit manage by 1 cdn instance
		b, err := s.Cache.Lock(cache.Key(CleanUnitLockKey, unit.ID), 10*time.Minute, 0, 1)
		if err != nil {
			return err
		}
		if !b {
			continue
		}
		log.Info(ctx, "cdn:cleanStorageUnit start cleaning storage unit")

		nb, err := storage.CountItemsForUnit(s.mustDBWithCtx(ctx), unit.ID)
		if err != nil {
			return err
		}
		if nb == 0 {
			if err := storage.DeleteUnit(s.Mapper, s.mustDBWithCtx(ctx), &unit); err != nil {
				return err
			}
			return nil
		}

		offset := int64(0)
		limit := int64(1000)
		for {
			ids, err := storage.LoadAllItemUnitsIDsByUnitID(s.mustDBWithCtx(ctx), offset, limit)
			if err != nil {
				return err
			}
			tx, err := s.mustDBWithCtx(ctx).Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			if _, err := storage.MarkItemUnitToDelete(tx, ids); err != nil {
				_ = tx.Rollback()
				return err
			}
			if err := tx.Commit(); err != nil {
				_ = tx.Rollback()
				return sdk.WithStack(err)
			}
			if int64(len(ids)) < limit {
				return nil
			}
		}
	}
	return nil
}

func (s *Service) cleanItemToDelete(ctx context.Context) error {
	ids, err := item.LoadIDsToDelete(s.mustDBWithCtx(ctx), 1000)
	if err != nil {
		return err
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })

	if len(ids) > 0 {
		log.Info(ctx, "cdn:purge:item: %d items to delete", len(ids))
	}

	for _, id := range ids {
		nbUnitItemToDelete, err := s.markUnitItemToDeleteByItemID(ctx, id)
		if err != nil {
			log.Error(ctx, "cdn:purge:item: unable to mark unit item %q to delete: %v", id, err)
			continue
		}

		log.Debug(ctx, "cdn:purge:item: %d unit items to delete for item %q", nbUnitItemToDelete, id)

		// If and only If there is not more unit item to mark as delete,
		// let's delete the item in database
		if nbUnitItemToDelete == 0 {
			nbItemUnits, err := storage.CountItemUnitsToDeleteByItemID(s.mustDBWithCtx(ctx), id)
			if err != nil {
				log.Error(ctx, "cdn:purge:item: unable to count unit item %q to delete: %v", id, err)
				continue
			}

			if nbItemUnits > 0 {
				log.Debug(ctx, "cdn:purge:item: %d unit items to delete for item %q", nbItemUnits, id)
			} else {
				if err := s.LogCache.Remove([]string{id}); err != nil {
					return sdk.WrapError(err, "cdn:purge:item: unable to remove from logCache for item %q", id)
				}
				if err := item.DeleteByID(s.mustDBWithCtx(ctx), id); err != nil {
					return sdk.WrapError(err, "cdn:purge:item: unable to delete from item with id %q", id)
				}
				for _, sto := range s.Units.Storages {
					s.Units.RemoveFromRedisSyncQueue(ctx, sto, id)
				}

				log.Debug(ctx, "cdn:purge:item: %s item deleted", id)
			}
		}
	}
	return nil
}

func (s *Service) cleanBuffer(ctx context.Context) error {
	storageCount := int64(len(s.Units.Storages) + 1)
	for _, bu := range s.Units.Buffers {
		itemIDs, err := storage.LoadAllSynchronizedItemIDs(s.mustDBWithCtx(ctx), bu.ID(), storageCount)
		if err != nil {
			return err
		}
		log.Debug(ctx, "item to remove from buffer: %d", len(itemIDs))
		if len(itemIDs) == 0 {
			return nil
		}

		itemUnitsIDs, err := storage.LoadAllItemUnitsIDsByItemIDsAndUnitID(s.mustDBWithCtx(ctx), bu.ID(), itemIDs)
		if err != nil {
			return err
		}

		tx, err := s.mustDBWithCtx(ctx).Begin()
		if err != nil {
			return sdk.WrapError(err, "unable to start transaction")
		}

		if _, err := storage.MarkItemUnitToDelete(tx, itemUnitsIDs); err != nil {
			_ = tx.Rollback()
			return err
		}

		if err := tx.Commit(); err != nil {
			_ = tx.Rollback()
			return sdk.WithStack(err)
		}
	}
	return nil
}

func (s *Service) cleanWaitingItem(ctx context.Context, duration int) error {
	itemUnits, err := storage.LoadOldItemUnitByItemStatusAndDuration(ctx, s.Mapper, s.mustDBWithCtx(ctx), sdk.CDNStatusItemIncoming, duration)
	if err != nil {
		return err
	}
	for _, itemUnit := range itemUnits {
		ctx = context.WithValue(ctx, storage.FieldAPIRef, itemUnit.Item.APIRefHash)
		log.Info(ctx, "cleanWaitingItem> cleaning item %s", itemUnit.ItemID)

		tx, err := s.mustDBWithCtx(ctx).Begin()
		if err != nil {
			return sdk.WrapError(err, "unable to start transaction")
		}
		if err := s.completeItem(ctx, tx, itemUnit); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			_ = tx.Rollback()
			return err
		}
		s.Units.PushInSyncQueue(ctx, itemUnit.ItemID, itemUnit.Item.Created)
		telemetry.Record(ctx, s.Metrics.itemCompletedByGCCount, 1)
	}
	return nil
}
