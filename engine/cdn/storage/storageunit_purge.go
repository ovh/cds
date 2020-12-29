package storage

import (
	"context"
	"fmt"

	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (x *RunningStorageUnits) Purge(ctx context.Context, s Interface) error {
	unitItems, err := LoadAllItemUnitsToDeleteByUnit(ctx, x.m, x.db, s.ID(), gorpmapper.GetOptions.WithDecryption)
	if err != nil {
		return err
	}

	if len(unitItems) > 0 {
		log.Info(ctx, "cdn:purge:%s: %d unit item to delete", s.Name(), len(unitItems))
	}

	for _, ui := range unitItems {
		exists, err := s.ItemExists(ctx, x.m, x.db, *ui.Item)
		if err != nil {
			return err
		}
		if exists {
			nbItemUnits, err := x.GetItemUnitByLocatorByUnit(ctx, ui.Locator, s.ID())
			if err != nil {
				return err
			}

			if nbItemUnits > 0 {
				log.Debug("cdn:purge:%s: item unit %s content will not be deleted because there is %d other item units with the same content ", s.Name(), ui.ID, nbItemUnits)
			} else {
				if err := s.Remove(ctx, ui); err != nil {
					if sdk.ErrorIs(err, sdk.ErrNotFound) {
						log.Info(ctx, "Item %s has already been deleted from %s", ui.ItemID, s.Name())
						continue
					}
					log.ErrorWithFields(ctx, log.Fields{
						"item_apiref":   ui.Item.APIRefHash,
						"item_size_num": ui.Item.Size,
						"stack_trace":   fmt.Sprintf("%+v", err),
					}, "unable to remove item %s on %s: %v", ui.ID, s.Name(), err)
					continue
				}
			}

			log.InfoWithFields(ctx, log.Fields{
				"item_apiref":   ui.Item.APIRefHash,
				"item_size_num": ui.Item.Size,
			}, "item %s deleted on %s", ui.ID, s.Name())
		}

		tx, err := x.db.Begin()
		if err != nil {
			return sdk.WithStack(err)
		}

		if err := DeleteItemUnit(x.m, tx, &ui); err != nil {
			log.ErrorWithFields(ctx, log.Fields{
				"item_apiref":   ui.Item.APIRefHash,
				"item_size_num": ui.Item.Size,
				"stack_trace":   fmt.Sprintf("%+v", err),
			}, "unable to delete item unit %s: %v", ui.ID, err)
			_ = tx.Rollback() // nolint
			continue
		}

		if err := tx.Commit(); err != nil {
			_ = tx.Rollback() // nolint
			return sdk.WithStack(err)
		}

		log.InfoWithFields(ctx, log.Fields{
			"item_apiref":   ui.Item.APIRefHash,
			"item_size_num": ui.Item.Size,
		}, "item %s deleted on %s", ui.ID, s.Name())
	}

	return nil
}
