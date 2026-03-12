package storage

import (
	"context"
	"sync"

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
	unitItems, err := LoadAllItemUnitsToDeleteByUnit(ctx, x.m, x.db, s.ID(), x.config.PurgeNbElements, gorpmapper.GetAllOptions.WithDecryption)
	if err != nil {
		return err
	}

	if len(unitItems) > 0 {
		log.Info(ctx, "cdn:purge:%s: %d unit item to delete with %d workers", s.Name(), len(unitItems), x.config.PurgeNbWorkers)
	}

	// Fan-out: distribute items to workers via a channel
	ch := make(chan sdk.CDNItemUnit, len(unitItems))
	for _, ui := range unitItems {
		ch <- ui
	}
	close(ch)

	var wg sync.WaitGroup
	for i := 0; i < x.config.PurgeNbWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ui := range ch {
				x.purgeItem(ctx, s, ui)
			}
		}()
	}
	wg.Wait()

	return nil
}

func (x *RunningStorageUnits) purgeItem(ctx context.Context, s Interface, ui sdk.CDNItemUnit) {
	ctx = context.WithValue(ctx, FieldAPIRef, ui.Item.APIRefHash)
	ctx = context.WithValue(ctx, FieldSize, ui.Item.Size)

	exists, err := s.ItemExists(ctx, x.m, x.db, *ui.Item)
	if err != nil {
		log.Error(ctx, "error on ItemExists: err:%s", err)
		return
	}

	if exists {
		var hasItemUnit bool
		if _, hasLocator := s.(StorageUnitWithLocator); hasLocator {
			var err error
			hasItemUnit, err = x.GetItemUnitByLocatorByUnit(ui.Locator, s.ID(), ui.Type)
			if err != nil {
				log.Error(ctx, "unable to check item unit locator %s: %v", ui.ID, err)
				return
			}
		}

		if hasItemUnit {
			log.Info(ctx, "item %s will not be deleted from %s", ui.ID, s.Name())
		} else {
			if err := s.Remove(ctx, ui); err != nil {
				if sdk.ErrorIs(err, sdk.ErrNotFound) {
					log.Info(ctx, "Item %s has already been deleted from %s", ui.ItemID, s.Name())
				} else {
					ctx = sdk.ContextWithStacktrace(ctx, err)
					log.Error(ctx, "unable to remove item %s on %s: %v", ui.ID, s.Name(), err)
				}
				return
			}
			log.Info(ctx, "item %s deleted on %s", ui.ID, s.Name())
		}
	}

	tx, err := x.db.Begin()
	if err != nil {
		log.Error(ctx, "unable to begin transaction for item unit %s: %v", ui.ID, err)
		return
	}

	if err := DeleteItemUnit(x.m, tx, &ui); err != nil {
		ctx = sdk.ContextWithStacktrace(ctx, err)
		log.Error(ctx, "unable to delete item unit %s: %v", ui.ID, err)
		_ = tx.Rollback() // nolint
		return
	}

	if err := tx.Commit(); err != nil {
		_ = tx.Rollback() // nolint
		log.Error(ctx, "unable to commit transaction for item unit %s: %v", ui.ID, err)
		return
	}

	log.Info(ctx, "item unit %s deleted on %s", ui.ID, s.Name())
}
