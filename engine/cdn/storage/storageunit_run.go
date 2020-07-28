package storage

import (
	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/cdn/index"
)

func Run(db gorp.SqlExecutor, s StorageUnit, i index.Item) error {
	// Find a storage unit where the item is complete

	// Read the Item one by one and pipe them in another storage unit

	// Save in database that the item is complete for the storage unit

	return nil
}
