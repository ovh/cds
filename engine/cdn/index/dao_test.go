package index_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func TestLoadItem(t *testing.T) {
	m := gorpmapper.New()
	index.InitDBMapping(m)

	db, _ := test.SetupPGWithMapper(t, m, sdk.TypeCDN)

	i := index.Item{
		Type:       index.TypeItemStepLog,
		ApiRefHash: sdk.UUID(),
	}
	require.NoError(t, index.InsertItem(context.TODO(), m, db, &i))

	res, err := index.LoadItemByID(context.TODO(), m, db, i.ID)
	require.NoError(t, err)
	require.Equal(t, i.ID, res.ID)
	require.Equal(t, i.Type, res.Type)
}

func TestLoadOldItemIDsByStatusAndDuration(t *testing.T) {
	m := gorpmapper.New()
	index.InitDBMapping(m)
	db, _ := test.SetupPGWithMapper(t, m, sdk.TypeCDN)

	// Clean old test
	time.Sleep(1 * time.Second)
	ids, err := index.LoadOldItemIDsByStatusAndDuration(db, index.StatusItemIncoming, 1)
	require.NoError(t, err)
	for _, id := range ids {
		i, err := index.LoadItemByID(context.TODO(), m, db, id)
		require.NoError(t, err)
		require.NoError(t, index.DeleteItem(m, db, i))
	}

	i1 := &index.Item{
		ID:         sdk.UUID(),
		ApiRefHash: sdk.UUID(),
		Type:       index.TypeItemStepLog,
		Status:     index.StatusItemCompleted,
	}
	err = index.InsertItem(context.TODO(), m, db, i1)
	require.NoError(t, err)

	i2 := &index.Item{
		ID:         sdk.UUID(),
		ApiRefHash: sdk.UUID(),
		Type:       index.TypeItemStepLog,
		Status:     index.StatusItemIncoming,
	}
	err = index.InsertItem(context.TODO(), m, db, i2)
	require.NoError(t, err)
	defer func() {
		_ = index.DeleteItem(m, db, i2)
	}()

	time.Sleep(2 * time.Second)

	ids, err = index.LoadOldItemIDsByStatusAndDuration(db, index.StatusItemIncoming, 1)
	require.NoError(t, err)
	require.Len(t, ids, 1)
	require.Equal(t, i2.ID, ids[0])
}
