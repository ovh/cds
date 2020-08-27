package storage

import (
	"context"
	"io"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (x *RunningStorageUnits) Run(ctx context.Context, s StorageUnit) error {
	s.Lock()
	defer s.Unlock()
	_, err := LoadUnitByID(ctx, s.GorpMapper(), s.DB(), s.ID())
	if err != nil {
		return err
	}

	// Load items to sync
	itemIDs, err := LoadAllItemIDUnknownByUnit(ctx, s.GorpMapper(), s.DB(), s.ID(), 100)
	if err != nil {
		return err
	}

	if len(itemIDs) > 0 {
		log.Info(ctx, "storage.Run> unit %s has %d items to sync", s.Name(), len(itemIDs))
	}

	for _, id := range itemIDs {
		tx, err := s.DB().Begin()
		if err != nil {
			return err
		}

		item, err := index.LoadAndLockItemByID(ctx, s.GorpMapper(), tx, id, gorpmapper.GetOptions.WithDecryption)
		if err != nil {
			log.Error(ctx, "error: %v", err)
			tx.Rollback() // nolint
			continue
		}

		if err := x.runItem(ctx, tx, s, item); err != nil {
			log.Error(ctx, "error: %v", err)
			tx.Rollback() // nolint
			continue
		}

		if err := tx.Commit(); err != nil {
			log.Error(ctx, "unable to commit txt: %v", err)
			tx.Rollback() // nolint
			continue
		}
	}
	return nil
}

func (x *RunningStorageUnits) runItem(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, dest StorageUnit, item *index.Item) error {
	t0 := time.Now()
	log.Debug("storage.runItem(%s, %s)", dest.Name(), item.ID)
	defer func() {
		log.Debug("storage.runItem(%s, %s): %fs", dest.Name(), item.ID, time.Since(t0).Seconds())
	}()
	var m = dest.GorpMapper()

	iu, err := x.NewItemUnit(ctx, m, tx, dest, item)
	if err != nil {
		log.Error(ctx, "unable to create new item unit: %v", err)
		return err
	}
	iu.Item = item

	// Save in database that the item is complete for the storage unit
	if err := InsertItemUnit(ctx, m, tx, iu); err != nil {
		log.Error(ctx, "unable to insert item unit: %v", err)
		return err
	}

	// Reload with decryption
	iu, err = LoadItemUnitByID(ctx, m, tx, iu.ID, gorpmapper.GetOptions.WithDecryption)
	if err != nil {
		return err
	}

	// Prepare the destination
	writer, err := dest.NewWriter(*iu)
	if err != nil {
		log.Error(ctx, "unable to get writer for item %s: %v", item.ID, err)
		return err
	}

	source, err := x.GetSource(ctx, iu.Item)
	if err != nil {
		log.Error(ctx, "unable to get source for item %s: %v", item.ID, err)
		return err
	}

	reader, err := source.NewReader()
	if err != nil {
		log.Error(ctx, "unable to get reader for item %s: %v", item.ID, err)
		return err
	}

	chanError := make(chan error)
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		if err := source.Read(reader, pw); err != nil {
			chanError <- err
		}
		close(chanError)
	}()

	if err := dest.Write(*iu, pr, writer); err != nil {
		return err
	}

	if err := pr.Close(); err != nil {
		return err
	}

	if err := reader.Close(); err != nil {
		return err
	}

	for err := range chanError {
		if err != nil {
			log.Error(ctx, "an error has occured: %v", err)
			return err
		}
	}

	log.Info(ctx, "item %s has been pushed to %s", item.ID, dest.Name())
	return nil
}

func (x *RunningStorageUnits) NewItemUnit(ctx context.Context, m *gorpmapper.Mapper, tx gorp.SqlExecutor, su Interface, i *index.Item) (*ItemUnit, error) {
	suloc, is := su.(StorageUnitWithLocator)
	var loc string
	if is {
		var err error
		loc, err = suloc.NewLocator(i.Hash)
		if err != nil {
			return nil, sdk.WrapError(err, "unable to compyte convergent locator")
		}
	}

	var iu = ItemUnit{
		ItemID:       i.ID,
		UnitID:       su.ID(),
		LastModified: time.Now(),
		Locator:      loc,
		Item:         i,
	}

	return &iu, nil
}
