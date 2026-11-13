package storage

import (
	"context"

	"github.com/go-gorp/gorp"
)

// ProcessItem exposes processItem to the external test package, which is the only one that can
// import the storage drivers
func (x *RunningStorageUnits) ProcessItem(ctx context.Context, db *gorp.DbMap, s StorageUnit, id string) error {
	return x.processItem(ctx, db, s, id)
}
