package cdn

import (
	"context"
	"math/rand"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

const (
	ItemLogGC = 24 * 3600
)

func (s *Service) itemPurge(ctx context.Context) {
	tickPurge := time.NewTicker(15 * time.Minute)
	defer tickPurge.Stop()
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
		}
	}
}

// ItemsGC clean long incoming item + delete item from buffer when synchronized everywhere
func (s *Service) itemsGC(ctx context.Context) {
	tickGC := time.NewTicker(30 * time.Minute)
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

func (s *Service) cleanItemToDelete(ctx context.Context) error {
	offset := 0
	limit := 1000

	for {
		ids, err := item.LoadIDsToDelete(s.mustDBWithCtx(ctx), offset, limit)
		if err != nil {
			return err
		}

		if len(ids) == 0 {
			return nil
		}

		log.Info(ctx, "cdn:purge:item: %d items to delete", len(ids))
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		r.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })

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
					continue
				}

				if err := s.LogCache.Remove(ctx, []string{id}); err != nil {
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
		if len(ids) < limit {
			return nil
		}
	}
}

func (s *Service) cleanBuffer(ctx context.Context) error {
	storageCount := int64(1)
	for _, s := range s.Units.Storages {
		if !s.CanSync() {
			continue
		}
		storageCount++
	}
	for _, bu := range s.Units.Buffers {
		itemIDs, err := storage.LoadAllSynchronizedItemIDs(s.mustDBWithCtx(ctx), bu.ID(), storageCount)
		if err != nil {
			return err
		}
		log.Debug(ctx, "item to remove from buffer: %d", len(itemIDs))
		if len(itemIDs) == 0 {
			continue
		}
		itemUnitsIDs, err := storage.LoadAllItemUnitsIDsByItemIDsAndUnitID(s.mustDBWithCtx(ctx), bu.ID(), itemIDs)
		if err != nil {
			ctx := sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "unable to load item units: %v", err)
			continue
		}
		tx, err := s.mustDBWithCtx(ctx).Begin()
		if err != nil {
			ctx := sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "unable to start transaction: %v", err)
			continue
		}

		if _, err := storage.MarkItemUnitToDelete(tx, itemUnitsIDs); err != nil {
			_ = tx.Rollback()
			ctx := sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "unable to mark item as delete: %v", err)
			continue
		}

		if err := tx.Commit(); err != nil {
			_ = tx.Rollback()
			ctx := sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "unable to commit transaction: %v", err)
			continue
		}
	}
	return nil
}

func (s *Service) cleanWaitingItem(ctx context.Context, duration int) error {
	items, err := item.LoadOldItemByStatusAndDuration(ctx, s.Mapper, s.mustDBWithCtx(ctx), sdk.CDNStatusItemIncoming, duration)
	if err != nil {
		return err
	}
	for _, it := range items {
		ctx = context.WithValue(ctx, storage.FieldAPIRef, it.APIRefHash)
		log.Info(ctx, "cleanWaitingItem> cleaning item %s", it.ID)

		// Load Item Unit
		itemUnits, err := storage.LoadAllItemUnitsByItemIDs(ctx, s.Mapper, s.mustDBWithCtx(ctx), it.ID)
		if err != nil {
			log.Error(ctx, "cleanWaitingItem> unable to load storage unit: %v", err)
			continue
		}

		tx, err := s.mustDBWithCtx(ctx).Begin()
		if err != nil {
			return sdk.WrapError(err, "unable to start transaction")
		}

		// If there is no item unit, mark item as delete
		if len(itemUnits) == 0 {
			it.Status = sdk.CDNStatusItemCompleted
			it.ToDelete = true
			if err := item.Update(ctx, s.Mapper, tx, &it); err != nil {
				_ = tx.Rollback()
				return err
			}
		} else {
			// Else complete item
			if err := s.completeItem(ctx, tx, itemUnits[0]); err != nil {
				_ = tx.Rollback()
				return err
			}
		}

		if err := tx.Commit(); err != nil {
			_ = tx.Rollback()
			return err
		}

		// Push item ID to run backend sync
		if len(itemUnits) > 0 {
			s.Units.PushInSyncQueue(ctx, it.ID, it.Created)
		}
		telemetry.Record(ctx, s.Metrics.itemCompletedByGCCount, 1)
	}
	return nil
}
