package storage_test

import (
	"context"
	"github.com/ovh/cds/engine/cdn/storage"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/cdn/index"
	cdntest "github.com/ovh/cds/engine/cdn/test"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
)

func TestLoadOldItemUnitByItemStatusAndDuration(t *testing.T) {
	m := gorpmapper.New()
	index.InitDBMapping(m)
	storage.InitDBMapping(m)
	db, _ := test.SetupPGWithMapper(t, m, sdk.TypeCDN)
	cfg := test.LoadTestingConf(t, sdk.TypeCDN)

	cdntest.ClearIndex(t, context.TODO(), m, db)

	cdnUnits, err := storage.Init(context.TODO(), m, db.DbMap, storage.Configuration{
		Buffer: storage.BufferConfiguration{
			Name: "redis_buffer",
			Redis: storage.RedisBufferConfiguration{
				Host:     cfg["redisHost"],
				Password: cfg["redisPassword"],
			},
		},
	})

	// Clean old test
	time.Sleep(1 * time.Second)
	itemUnits, err := storage.LoadOldItemUnitByItemStatusAndDuration(context.TODO(), m, db, index.StatusItemIncoming, 1)
	require.NoError(t, err)
	for _, itemUnit := range itemUnits {
		i, err := index.LoadItemByID(context.TODO(), m, db, itemUnit.ItemID)
		require.NoError(t, err)
		require.NoError(t, index.DeleteItem(m, db, i))
	}

	i1 := &index.Item{
		ID:         sdk.UUID(),
		APIRefHash: sdk.UUID(),
		Type:       sdk.CDNTypeItemStepLog,
		Status:     index.StatusItemCompleted,
	}
	err = index.InsertItem(context.TODO(), m, db, i1)
	require.NoError(t, err)
	defer func() {
		_ = index.DeleteItem(m, db, i1)
	}()

	itemUnit1, err := cdnUnits.NewItemUnit(context.TODO(), cdnUnits.Buffer, i1)
	require.NoError(t, err)
	require.NoError(t, storage.InsertItemUnit(context.TODO(), m, db, itemUnit1))

	i2 := &index.Item{
		ID:         sdk.UUID(),
		APIRefHash: sdk.UUID(),
		Type:       sdk.CDNTypeItemStepLog,
		Status:     index.StatusItemIncoming,
	}
	err = index.InsertItem(context.TODO(), m, db, i2)
	require.NoError(t, err)
	defer func() {
		_ = index.DeleteItem(m, db, i2)
	}()
	itemUnit2, err := cdnUnits.NewItemUnit(context.TODO(), cdnUnits.Buffer, i2)
	require.NoError(t, err)

	require.NoError(t, storage.InsertItemUnit(context.TODO(), m, db, itemUnit2))

	time.Sleep(2 * time.Second)

	itemUnits, err = storage.LoadOldItemUnitByItemStatusAndDuration(context.TODO(), m, db, index.StatusItemIncoming, 1)
	require.NoError(t, err)
	require.Len(t, itemUnits, 1)
	require.Equal(t, i2.ID, itemUnits[0].ItemID)
}
