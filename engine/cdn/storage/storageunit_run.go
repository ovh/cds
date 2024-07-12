package storage

import (
	"context"
	"io"
	"time"

	"github.com/fujiwara/shapeio"
	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

var (
	KeyBackendSync = "cdn:backend:sync"
)

func (x *RunningStorageUnits) FillSyncItemChannel(ctx context.Context, s StorageUnit, nbItem int64) error {
	var itemIDs []string
	if err := x.cache.ScoredSetRevRange(ctx, cache.Key(KeyBackendSync, s.Name()), 0, nbItem, &itemIDs); err != nil {
		return err
	}
	if len(itemIDs) > 0 {
		log.Info(ctx, "FillSyncItemChannel> Item to sync for %s: %d", s.Name(), len(itemIDs))
	}
	for _, id := range itemIDs {
		select {
		case s.SyncItemChannel() <- id:
			log.Debug(ctx, "unit %s should sync item %s", s.Name(), id)
		default:
			continue
		}
	}
	return nil
}

func (x *RunningStorageUnits) FillWithUnknownItems(ctx context.Context, s StorageUnit, maxItemByLoop int64) error {
	lockKey := cache.Key("cdn", "backend", "lock", "sync", s.Name())
	b, err := x.cache.Lock(ctx, lockKey, 10*time.Minute, 0, 1)
	if err != nil {
		return err
	}
	if !b {
		return nil
	}
	defer func() {
		if err := x.cache.Unlock(ctx, lockKey); err != nil {
			log.Error(ctx, "unable to release lock %s", lockKey)
		}
	}()

	log.Info(ctx, "FillWithUnknownItems> Getting lock for backend %s sync", s.Name())

	offset := int64(0)
	for {
		itemsToSync, err := LoadAllItemIDUnknownByUnit(x.db, s.ID(), offset, maxItemByLoop)
		if err != nil {
			return err
		}
		log.Info(ctx, "FillWithUnknownItems> Get %d items", len(itemsToSync))
		k := cache.Key(KeyBackendSync, s.Name())
		for _, item := range itemsToSync {
			ctx = context.WithValue(ctx, FielID, item.ItemID)
			if err := x.cache.ScoredSetAdd(ctx, k, item.ItemID, float64(item.Created.Unix())); err != nil {
				log.Error(ctx, "FillWithUnknownItems> unable to push item %s into %s", item.ItemID, k)
				continue
			}
		}
		if int64(len(itemsToSync)) < maxItemByLoop {
			break
		}
		offset += int64(len(itemsToSync))
	}
	return nil
}

func (x *RunningStorageUnits) processItem(ctx context.Context, db *gorp.DbMap, s StorageUnit, id string) error {
	lockKey := cache.Key("cdn", "sync", "item", id)
	hasLock, err := x.cache.Lock(ctx, lockKey, 20*time.Minute, 0, 1)
	if err != nil {
		log.Error(ctx, "unable to get lock %s: %v", lockKey, err)
	}
	if !hasLock {
		return nil
	}
	defer func() {
		_ = x.cache.Unlock(ctx, lockKey)
	}()

	it, err := item.LoadByID(ctx, x.m, db, id, gorpmapper.GetOptions.WithDecryption)
	if err != nil {
		return err
	}
	ctx = context.WithValue(ctx, FieldAPIRef, it.APIRefHash)
	ctx = context.WithValue(ctx, FieldSize, it.Size)
	log.Info(ctx, "processing item %s on %s", it.ID, s.Name())
	if _, err = LoadItemUnitByUnit(ctx, x.m, db, s.ID(), id); err == nil {
		log.Info(ctx, "Item %s already sync on %s", id, s.Name())
		return nil

	}
	if !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return err
	}

	if err := x.runItem(ctx, db, s, it); err != nil {
		return err
	}
	x.RemoveFromRedisSyncQueue(ctx, s, id)
	return nil
}

func (x *RunningStorageUnits) runItem(ctx context.Context, db *gorp.DbMap, dest StorageUnit, item *sdk.CDNItem) error {
	iu, err := x.NewItemUnit(ctx, dest, item)
	if err != nil {
		return err
	}
	iu.Item = item

	// Check if the content (based on the locator) is already known from the destination unit
	has, err := x.GetItemUnitByLocatorByUnit(iu.Locator, dest.ID(), iu.Type)
	if err != nil {
		return err
	}

	if has {
		log.Info(ctx, "item %s has been pushed to %s with deduplication", item.ID, dest.Name())
		tx, err := db.Begin()
		if err != nil {
			return sdk.WrapError(err, "unable to start transaction")
		}
		defer tx.Rollback() //nolint
		// Save in database that the item is complete for the storage unit
		if err := InsertItemUnit(ctx, x.m, tx, iu); err != nil {
			return err
		}
		return sdk.WrapError(tx.Commit(), "unable to commit tx")
	}

	t1 := time.Now()

	// Prepare the destination
	writer, err := dest.NewWriter(ctx, *iu)
	if err != nil {
		return err
	}
	if writer == nil {
		return sdk.NewErrorFrom(sdk.ErrNotFound, "unable to get writer")
	}
	rateLimitWriter := shapeio.NewWriter(writer)
	rateLimitWriter.SetRateLimit(dest.SyncBandwidth())
	log.Debug(ctx, "%s write ratelimit: %v", dest.Name(), dest.SyncBandwidth())

	source, err := x.GetSource(ctx, item)
	if err != nil {
		return err
	}

	reader, err := source.NewReader(ctx)
	if err != nil {
		return err
	}

	rateLimitReader := shapeio.NewReader(reader)
	rateLimitReader.SetRateLimit(source.SyncBandwidth())
	log.Debug(ctx, "%s read ratelimit: %v", source.Name(), source.SyncBandwidth())

	chanError := make(chan error)
	pr, pw := io.Pipe()
	gr := sdk.NewGoRoutines(ctx)

	gr.Exec(ctx, "runningStorageUnits.runItem.read", func(ctx context.Context) {
		defer pw.Close()
		if err := source.Read(rateLimitReader, pw); err != nil {
			chanError <- err
		}
		close(chanError)
	})

	if err := dest.Write(*iu, pr, rateLimitWriter); err != nil {
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

	log.Info(ctx, "item %s has been pushed to %s (%.3f s)", item.ID, dest.Name(), t2.Sub(t1).Seconds())

	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "unable to start transaction")
	}
	defer tx.Rollback() //nolint
	// Save in database that the item is complete for the storage unit
	if err := InsertItemUnit(ctx, x.m, tx, iu); err != nil {
		return err
	}
	return sdk.WrapError(tx.Commit(), "unable to commit tx")
}

func (x *RunningStorageUnits) NewItemUnit(_ context.Context, su Interface, i *sdk.CDNItem) (*sdk.CDNItemUnit, error) {
	suloc, is := su.(StorageUnitWithLocator)
	var loc string
	if is {
		var err error
		loc, err = suloc.NewLocator(i.Hash)
		if err != nil {
			return nil, sdk.WrapError(err, "unable to compute convergent locator")
		}
	}

	hashLocator := x.HashLocator(loc)
	var iu = sdk.CDNItemUnit{
		ItemID:       i.ID,
		Type:         i.Type,
		UnitID:       su.ID(),
		LastModified: time.Now(),
		Locator:      loc,
		HashLocator:  hashLocator,
		Item:         i,
	}

	return &iu, nil
}
