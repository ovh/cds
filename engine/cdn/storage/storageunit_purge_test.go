package storage_test

import (
	"context"
	"testing"

	"github.com/go-gorp/gorp"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

// goneContentUnit is a storage unit that still sees the item, but finds its content already
// removed, as when another purge removed it first
type goneContentUnit struct {
	storage.StorageUnit
}

func (s *goneContentUnit) ItemExists(_ context.Context, _ *gorpmapper.Mapper, _ gorp.SqlExecutor, _ sdk.CDNItem) (bool, error) {
	return true, nil
}

func (s *goneContentUnit) Remove(_ context.Context, _ sdk.CDNItemUnit) error {
	return sdk.WithStack(sdk.ErrNotFound)
}

// When the content of an item unit to delete is already gone, the item unit must still be
// deleted: a row left marked to delete is never purged and keeps its item from being cleaned.
func TestPurgeDeletesItemUnitWhenContentIsAlreadyGone(t *testing.T) {
	ctx, m, db, _, cdnUnits, dest := initRecordingStorageUnit(t)

	i := insertCompletedLogItem(t, ctx, m, db, randomHash(t))
	iu, err := cdnUnits.NewItemUnit(ctx, dest, i)
	require.NoError(t, err)
	require.NoError(t, storage.InsertItemUnit(ctx, m, db, iu))
	_, err = storage.MarkItemUnitToDelete(db, []string{iu.ID})
	require.NoError(t, err)

	require.NoError(t, cdnUnits.Purge(ctx, &goneContentUnit{StorageUnit: dest}))

	ius, err := storage.LoadAllItemUnitsByItemIDIncludingDeleted(ctx, m, db, i.ID)
	require.NoError(t, err)
	require.Empty(t, ius, "the item unit must be deleted even if its content was already gone")
}

// The purges of several CDN instances are not locked against each other: deleting an item unit
// that another instance has already deleted must not fail.
func TestDeleteItemUnitAlreadyDeleted(t *testing.T) {
	ctx, m, db, _, cdnUnits, dest := initRecordingStorageUnit(t)

	i := insertCompletedLogItem(t, ctx, m, db, randomHash(t))
	iu, err := cdnUnits.NewItemUnit(ctx, dest, i)
	require.NoError(t, err)
	require.NoError(t, storage.InsertItemUnit(ctx, m, db, iu))

	require.NoError(t, storage.DeleteItemUnit(m, db, iu))
	require.NoError(t, storage.DeleteItemUnit(m, db, iu))
}
