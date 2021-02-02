package storage

import (
	"context"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

const (
	FieldAPIRef = log.Field("item_apiref")
	FieldSize   = log.Field("item_size_num")
	FielID      = log.Field("item_id")
)

func init() {
	log.RegisterField(FieldAPIRef)
}

func (x *RunningStorageUnits) Purge(ctx context.Context, s Interface) error {
	unitItems, err := LoadAllItemUnitsToDeleteByUnit(ctx, x.m, x.db, s.ID(), gorpmapper.GetOptions.WithDecryption)
	if err != nil {
		return err
	}

	if len(unitItems) > 0 {
		log.Info(ctx, "cdn:purge:%s: %d unit item to delete", s.Name(), len(unitItems))
	}

	for _, ui := range unitItems {
		var exists bool
		// here, item could be nil if there are many cdn instances purging the same item
		if ui.Item != nil {
			ctx = context.WithValue(ctx, FieldAPIRef, ui.Item.APIRefHash)
			ctx = context.WithValue(ctx, FieldSize, ui.Item.Size)
			var err error
			exists, err = s.ItemExists(ctx, x.m, x.db, *ui.Item)
			if err != nil {
				return err
			}
		}

		if exists {
			var hasItemUnit bool
			if _, hasLocator := s.(StorageUnitWithLocator); hasLocator {
				var err error
				hasItemUnit, err = x.GetItemUnitByLocatorByUnit(ui.Locator, s.ID())
				if err != nil {
					return err
				}
			}

			if hasItemUnit {
				log.Info(ctx, "item %s will not be deleted from %s", ui.ID, s.Name())
			} else {
				if err := s.Remove(ctx, ui); err != nil {
					if sdk.ErrorIs(err, sdk.ErrNotFound) {
						log.Info(ctx, "Item %s has already been deleted from %s", ui.ItemID, s.Name())
						continue
					}
					ctx = sdk.ContextWithStacktrace(ctx, err)
					log.Error(ctx, "unable to remove item %s on %s: %v", ui.ID, s.Name(), err)
					continue
				}
				log.Info(ctx, "item %s deleted on %s", ui.ID, s.Name())
			}
		}

		tx, err := x.db.Begin()
		if err != nil {
			return sdk.WithStack(err)
		}

		if err := DeleteItemUnit(x.m, tx, &ui); err != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "unable to delete item unit %s: %v", ui.ID, err)
			_ = tx.Rollback() // nolint
			continue
		}

		if err := tx.Commit(); err != nil {
			_ = tx.Rollback() // nolint
			return sdk.WithStack(err)
		}

		log.Info(ctx, "item %s deleted on %s", ui.ID, s.Name())
	}

	return nil
}
