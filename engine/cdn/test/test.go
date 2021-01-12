package test

import (
	"context"
	"github.com/ovh/cds/engine/cache"
	"testing"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/stretchr/testify/require"
)

func ClearSyncRedisSet(t *testing.T, store cache.Store, name string) {
	require.NoError(t, store.Delete(cache.Key(storage.KeyBackendSync, name)))
}

func ClearItem(t *testing.T, ctx context.Context, m *gorpmapper.Mapper, db gorpmapper.SqlExecutorWithTx) {
	// clear datas
	items, err := item.LoadAll(ctx, m, db, 500)
	require.NoError(t, err)
	for _, i := range items {
		_ = item.DeleteByID(db, i.ID)
	}
}

func ClearUnits(t *testing.T, ctx context.Context, m *gorpmapper.Mapper, db gorpmapper.SqlExecutorWithTx) {
	units, err := storage.LoadAllUnits(ctx, m, db)
	require.NoError(t, err)
	for _, u := range units {
		storage.DeleteUnit(t, m, db, &u)
	}
}
