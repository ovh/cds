package storage

import (
	"context"
	"io"
	"math/rand"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type withNewReader interface {
	NewReader(ItemUnit) (io.ReadCloser, error)
	Read(ItemUnit, io.Reader, io.Writer) error
	Name() string
}

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

		if err := x.runItem(ctx, tx, s, id); err != nil {
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

var (
	rs = rand.NewSource(time.Now().Unix())
	r  = rand.New(rs)
)

func (x *RunningStorageUnits) runItem(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, dest StorageUnit, id string) error {
	t0 := time.Now()
	log.Debug("storage.runItem(%s, %s)", dest.Name(), id)
	defer func() {
		log.Debug("storage.runItem(%s, %s): %fs", dest.Name(), id, time.Since(t0).Seconds())
	}()
	var m = dest.GorpMapper()

	// Find a storage unit where the item is complete
	itemUnits, err := LoadAllItemUnitsByItemID(ctx, m, tx, id)
	if err != nil {
		log.Error(ctx, "unable to load item unit index: %v", err)
		return err
	}

	if len(itemUnits) == 0 {
		log.Info(ctx, "item %s can't be sync. No unit knows it...", id)
		return err
	}

	// Load the item
	item, err := index.LoadItemByID(ctx, m, tx, id, gorpmapper.GetOptions.WithDecryption)
	if err != nil {
		log.Error(ctx, "unable to load item index: %v", err)
		return err
	}

	// Random pick a unit
	idx := 0
	if len(itemUnits) > 1 {
		idx = r.Intn(len(itemUnits))
	}
	refUnitID := itemUnits[idx].UnitID
	refUnit, err := LoadUnitByID(ctx, m, tx, refUnitID)
	if err != nil {
		log.Error(ctx, "unable to load unit %s: %v", refUnitID, err)
		return err
	}

	// Read & Write the content
	var source withNewReader
	source = x.Storage(refUnit.Name)
	if source == nil {
		source = x.Buffer
	}

	if source == nil {
		log.Error(ctx, "unable to find unit %s", refUnit.Name)
		return err
	}

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

	// Prepare the reader from the reference storage
	reader, err := source.NewReader(*iu)
	if err != nil {
		log.Error(ctx, "unable to get reader for item %s: %v", item.ID, err)
		return err
	}

	// Prepare the destination
	writer, err := dest.NewWriter(*iu)
	if err != nil {
		log.Error(ctx, "unable to get writer for item %s: %v", item.ID, err)
		return err
	}

	chanError := make(chan error)

	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		if err := source.Read(*iu, reader, pw); err != nil {
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

	log.Info(ctx, "item %s has been pushed to %s (from %s)", item.ID, dest.Name(), source.Name())
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
	}

	return &iu, nil
}
