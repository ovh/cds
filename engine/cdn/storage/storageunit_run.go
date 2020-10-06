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
	"github.com/ovh/cds/sdk/telemetry"
)

func (x *RunningStorageUnits) Run(ctx context.Context, s StorageUnit, nbItem int64) error {
	// s.Lock()
	// defer s.Unlock()
	if _, err := LoadUnitByID(ctx, x.m, x.db, s.ID()); err != nil {
		return err
	}

	// Load items to sync
	itemIDs, err := LoadAllItemIDUnknownByUnitOrderByUnitID(x.db, s.ID(), x.Buffer.ID(), nbItem)
	if err != nil {
		return err
	}

	if len(itemIDs) > 0 {
		log.Info(ctx, "storage.Run> unit %s has %d items to sync (max: %d)", s.Name(), len(itemIDs), nbItem)
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
			_ = tx.Rollback()
			continue
		}

		_, err = LoadItemUnitByUnit(ctx, x.m, tx, s.ID(), id)
		if err == nil {
			_ = tx.Rollback()
			continue
		}
		if !sdk.ErrorIs(err, sdk.ErrNotFound) {
			log.ErrorWithFields(ctx, logrus.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
			_ = tx.Rollback()
			continue
		}

		if err := x.runItem(ctx, tx, s, it); err != nil {
			log.ErrorWithFields(ctx, logrus.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
			_ = tx.Rollback()
			continue
		}

		if err := tx.Commit(); err != nil {
			err = sdk.WrapError(err, "unable to commit txt")
			log.ErrorWithFields(ctx, logrus.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
			_ = tx.Rollback()
			continue
		}
	}
	return nil
}

func (x *RunningStorageUnits) runItem(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, dest StorageUnit, item *sdk.CDNItem) error {
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

	t1 := time.Now()

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

	t2 := time.Now()

	for err := range chanError {
		if err != nil {
			return err
		}
	}

	var throughput = item.Size / t2.Sub(t1).Milliseconds()
	if x.Metrics.StorageThroughput != nil {
		ctxMetrics := telemetry.ContextWithTag(ctx, "storage_source", source.Name(), "storage_dest", dest.Name())
		telemetry.Record(ctxMetrics, *x.Metrics.StorageThroughput, throughput)
	}

	log.InfoWithFields(ctx, logrus.Fields{
		"item_apiref":               item.APIRefHash,
		"source":                    source.Name(),
		"destination":               dest.Name(),
		"duration_milliseconds_num": t2.Sub(t1).Milliseconds(),
		"item_size_num":             item.Size,
		"throughput_num":            throughput,
	}, "item %s has been pushed to %s (%.3f s)", item.ID, dest.Name(), t2.Sub(t1).Seconds())
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
