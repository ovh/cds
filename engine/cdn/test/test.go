package test

import (
	"context"
	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/stretchr/testify/require"
	"testing"
)

func ClearIndex(t *testing.T, ctx context.Context, m *gorpmapper.Mapper, db gorpmapper.SqlExecutorWithTx) {
	// clear datas
	items, err := index.LoadAllItems(ctx, m, db, 500)
	require.NoError(t, err)
	for _, i := range items {
		_ = index.DeleteItem(m, db, &i)
	}
}

func ClearUnits(t *testing.T, ctx context.Context, m *gorpmapper.Mapper, db gorpmapper.SqlExecutorWithTx) {
	units, err := storage.LoadAllUnits(ctx, m, db)
	require.NoError(t, err)
	for _, u := range units {
		require.NoError(t, m.Delete(db, &u))
	}
}
