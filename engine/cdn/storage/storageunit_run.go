package storage

import (
	"context"
	"io"
	"math/rand"
	"time"

	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/sdk/log"
)

type withNewReader interface {
	NewReader(i index.Item) (io.ReadCloser, error)
}

func (x *RunningStorageUnits) Run(ctx context.Context, s StorageUnit) error {
	su, err := LoadUnitByID(ctx, s.GorpMapper(), s.DB(), s.ID())
	if err != nil {
		return err
	}

	rs := rand.NewSource(time.Now().Unix())
	r := rand.New(rs)

	// Load items to sync
	itemIDs, err := LoadAllItemIDUnknownByUnit(ctx, s.GorpMapper(), s.DB(), s.ID(), 5)
	if err != nil {
		return err
	}

	log.Info(ctx, "storage.Run> unit %s has %d items to sync", s.Name(), len(itemIDs))

	for _, id := range itemIDs {
		tx, err := s.DB().Begin()
		if err != nil {
			return err
		}

		// Find a storage unit where the item is complete
		itemUnits, err := LoadAllItemUnitsByItemID(ctx, s.GorpMapper(), tx, id)
		if err != nil {
			log.Error(ctx, "unable to load item unit index: %v", err)
			tx.Rollback() // nolint
			continue
		}

		if len(itemUnits) == 0 {
			log.Info(ctx, "item %s can't be sync. No unit knows it...", id)
			tx.Rollback() // nolint
			continue
		}

		// Load the item
		item, err := index.LoadItemByID(ctx, s.GorpMapper(), tx, id)
		if err != nil {
			log.Error(ctx, "unable to load item index: %v", err)
			tx.Rollback() // nolint
			continue
		}

		// Random pick a unit
		idx := 0
		if len(itemUnits) > 1 {
			idx = r.Intn(len(itemUnits))
		}
		refUnitID := itemUnits[idx].UnitID
		refUnit, err := LoadUnitByID(ctx, s.GorpMapper(), tx, refUnitID)
		if err != nil {
			log.Error(ctx, "unable to load unit %s: %v", refUnitID, err)
			tx.Rollback() // nolint
			continue
		}

		// Read & Write the content
		var refStorage withNewReader
		refStorage = x.Storage(refUnit.Name)
		if refStorage == nil {
			refStorage = x.Buffer
		}

		if refStorage == nil {
			log.Error(ctx, "unable to find unit %s", refUnit.Name)
			tx.Rollback() // nolint
			continue
		}

		reader, err := refStorage.NewReader(*item)
		if err != nil {
			log.Error(ctx, "unable to get reader for item %s: %v", item.ID, err)
			tx.Rollback() // nolint
			continue
		}

		// Prepare the destination
		writer, err := s.NewWriter(*item)
		if err != nil {
			log.Error(ctx, "unable to get writer for item %s: %v", item.ID, err)
			tx.Rollback() // nolint
			continue
		}

		// Copy
		n, err := io.Copy(writer, reader)
		if err != nil {
			log.Error(ctx, "unable to copy item %s: %v", item.ID, err)
			tx.Rollback() // nolint
			continue
		}

		if err := reader.Close(); err != nil {
			log.Error(ctx, "unable to close reader: %v", err)
			tx.Rollback() // nolint
			continue
		}

		if err := writer.Close(); err != nil {
			log.Error(ctx, "unable to close writer: %v", err)
			tx.Rollback() // nolint
			continue
		}

		log.Debug("%d bytes copied", n)

		// Save in database that the item is complete for the storage unit
		if _, err := InsertItemUnit(ctx, s.GorpMapper(), tx, su.ID, item.ID); err != nil {
			log.Error(ctx, "unable to insert item unit: %v", err)
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
