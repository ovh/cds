package cdn

import (
	"context"
	"fmt"
	"time"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/cdn/storage/cds"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

const (
	ItemLogGC = 24 * 3600
)

func (s *Service) itemPurge(ctx context.Context) {
	tickPurge := time.NewTicker(1 * time.Minute)
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
				log.ErrorWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
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
				log.ErrorWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
			}
			if err := s.cleanWaitingItem(ctx, ItemLogGC); err != nil {
				log.ErrorWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
			}
		}
	}
}

func (s *Service) markUnitItemToDeleteByItemID(ctx context.Context, itemID string) (int, error) {
	db := s.mustDBWithCtx(ctx)
	mapItemUnits, err := storage.LoadAllItemUnitsByItemIDs(ctx, s.Mapper, db, []string{itemID})
	if err != nil {
		return 0, err
	}
	uis, has := mapItemUnits[itemID]
	if !has {
		return 0, nil
	}

	ids := make([]string, len(uis))
	for i := range uis {
		ids[i] = uis[i].ID
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, sdk.WithStack(err)
	}

	defer tx.Rollback() // nolint

	n, err := storage.MarkItemUnitToDelete(ctx, s.Mapper, tx, ids)
	if err != nil {
		return 0, err
	}

	return n, sdk.WithStack(tx.Commit())
}

func (s *Service) cleanItemToDelete(ctx context.Context) error {
	ids, err := item.LoadIDsToDelete(s.mustDBWithCtx(ctx), 100)
	if err != nil {
		return err
	}

	if len(ids) > 0 {
		log.Info(ctx, "cdn:purge:item: %d items to delete", len(ids))
	}

	for _, id := range ids {
		nbUnitItemToDelete, err := s.markUnitItemToDeleteByItemID(ctx, id)
		if err != nil {
			log.Error(ctx, "unable to mark unit item %q to delete: %v", id, err)
			continue
		}

		// If and only If there is not more unit item to mark as delete,
		// let's delete the item in database
		if nbUnitItemToDelete == 0 {
			itemUnits, err := storage.LoadAllItemUnitsToDeleteByID(ctx, s.Mapper, s.mustDBWithCtx(ctx), id)
			if err != nil {
				log.Error(ctx, "unable to count unit item %q to delete: %v", id, err)
				continue
			}

			if len(itemUnits) > 0 {
				log.Debug("cdn:purge:item: %d unit items to delete for item %s", len(itemUnits), id)
			} else {
				if err := s.LogCache.Remove([]string{id}); err != nil {
					return err
				}
				if err := item.DeleteByID(s.mustDBWithCtx(ctx), id); err != nil {
					return err
				}
				log.Debug("cdn:purge:item: %s item deleted", id)
			}
			continue
		}

		log.Debug("cdn:purge:item: %d unit items to delete for item %s", nbUnitItemToDelete, id)
	}
	return nil
}

func (s *Service) cleanBuffer(ctx context.Context) error {
	var cdsBackendID string
	for _, sto := range s.Units.Storages {
		_, ok := sto.(*cds.CDS)
		if !ok {
			continue
		}
		cdsBackendID = sto.ID()
		break
	}

	itemIDs, err := storage.LoadAllSynchronizedItemIDs(s.mustDBWithCtx(ctx))
	if err != nil {
		return err
	}

	var itemUnitIDsToRemove []string
	mapItemunits, err := storage.LoadAllItemUnitsByItemIDs(ctx, s.Mapper, s.mustDBWithCtx(ctx), itemIDs)
	if err != nil {
		return err
	}

	if len(mapItemunits) == 0 {
		return nil
	}

	for _, itemunits := range mapItemunits {
		var countWithoutCDSBackend = len(itemunits)
		var bufferItemUnit string
		for _, iu := range itemunits {
			switch iu.UnitID {
			case cdsBackendID:
				countWithoutCDSBackend--
			case s.Units.Buffer.ID():
				bufferItemUnit = iu.ID
			}

			if countWithoutCDSBackend > 1 {
				itemUnitIDsToRemove = append(itemUnitIDsToRemove, bufferItemUnit)
			}
		}
	}

	if len(itemUnitIDsToRemove) == 0 {
		return nil
	}

	log.Debug("removing %d from buffer unit", len(itemUnitIDsToRemove))

	tx, err := s.mustDBWithCtx(ctx).Begin()
	if err != nil {
		return sdk.WrapError(err, "unable to start transaction")
	}
	defer tx.Rollback() //nolint

	if _, err := storage.MarkItemUnitToDelete(ctx, s.Mapper, tx, itemUnitIDsToRemove); err != nil {
		return err
	}

	return sdk.WithStack(tx.Commit())
}

func (s *Service) cleanWaitingItem(ctx context.Context, duration int) error {
	itemUnits, err := storage.LoadOldItemUnitByItemStatusAndDuration(ctx, s.Mapper, s.mustDBWithCtx(ctx), sdk.CDNStatusItemIncoming, duration)
	if err != nil {
		return err
	}
	for _, itemUnit := range itemUnits {
		log.InfoWithFields(ctx, log.Fields{"item_apiref": itemUnit.Item.APIRef}, "cleanWaitingItem> cleaning item %s", itemUnit.ItemID)
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
		telemetry.Record(ctx, s.Metrics.itemCompletedByGCCount, 1)
	}
	return nil
}
