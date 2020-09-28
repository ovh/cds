package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (x *RunningStorageUnits) Run(ctx context.Context, s StorageUnit) error {
	s.Lock()
	defer s.Unlock()
	_, err := LoadUnitByID(ctx, x.m, x.db, s.ID())
	if err != nil {
		return err
	}

	// Load items to sync
	itemIDs, err := LoadAllItemIDUnknownByUnitOrderByUnitID(x.db, s.ID(), x.Buffer.ID(), 100)
	if err != nil {
		return err
	}

	if len(itemIDs) > 0 {
		log.Info(ctx, "storage.Run> unit %s has %d items to sync", s.Name(), len(itemIDs))
	}

	for _, id := range itemIDs {
		tx, err := x.db.Begin()
		if err != nil {
			return err
		}

		it, err := item.LoadAndLockByID(ctx, x.m, tx, id, gorpmapper.GetOptions.WithDecryption)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrNotFound) {
				log.ErrorWithFields(ctx, logrus.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
			}
			tx.Rollback() // nolint
			continue
		}

		if err := x.runItem(ctx, tx, s, it); err != nil {
			log.ErrorWithFields(ctx, logrus.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
			tx.Rollback() // nolint
			continue
		}

		if err := tx.Commit(); err != nil {
			err = sdk.WrapError(err, "unable to commit txt")
			log.ErrorWithFields(ctx, logrus.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
			tx.Rollback() // nolint
			continue
		}
	}
	return nil
}

func (x *RunningStorageUnits) runItem(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, dest StorageUnit, item *sdk.CDNItem) error {
	t0 := time.Now()
	log.Debug("storage.runItem(%s, %s)", dest.Name(), item.ID)
	defer func() {
		log.Debug("storage.runItem(%s, %s): %fs", dest.Name(), item.ID, time.Since(t0).Seconds())
	}()

	iu, err := x.NewItemUnit(ctx, dest, item)
	if err != nil {
		return err
	}
	iu.Item = item

	// Save in database that the item is complete for the storage unit
	if err := InsertItemUnit(ctx, x.m, tx, iu); err != nil {
		return err
	}

	// Reload with decryption
	iu, err = LoadItemUnitByID(ctx, x.m, tx, iu.ID, gorpmapper.GetOptions.WithDecryption)
	if err != nil {
		return err
	}

	// Prepare the destination
	writer, err := dest.NewWriter(ctx, *iu)
	if err != nil {
		return err
	}
	if writer == nil {
		return nil
	}

	source, err := x.GetSource(ctx, item)
	if err != nil {
		return err
	}

	reader, err := source.NewReader(ctx)
	if err != nil {
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
		_ = pr.Close()
		_ = reader.Close()
		_ = writer.Close()
		return err
	}

	if err := pr.Close(); err != nil {
		_ = reader.Close()
		_ = writer.Close()
		return sdk.WithStack(err)
	}

	if err := reader.Close(); err != nil {
		_ = writer.Close()
		return sdk.WithStack(err)
	}

	_ = writer.Close()

	for err := range chanError {
		if err != nil {
			return err
		}
	}

	log.Info(ctx, "item %s has been pushed to %s", item.ID, dest.Name())
	return nil
}

func (x *RunningStorageUnits) NewItemUnit(_ context.Context, su Interface, i *sdk.CDNItem) (*sdk.CDNItemUnit, error) {
	suloc, is := su.(StorageUnitWithLocator)
	var loc string
	if is {
		var err error
		loc, err = suloc.NewLocator(i.Hash)
		if err != nil {
			return nil, sdk.WrapError(err, "unable to compyte convergent locator")
		}
	}

	var iu = sdk.CDNItemUnit{
		ItemID:       i.ID,
		UnitID:       su.ID(),
		LastModified: time.Now(),
		Locator:      loc,
		Item:         i,
	}

	return &iu, nil
}
