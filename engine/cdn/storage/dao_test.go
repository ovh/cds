package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/ovh/cds/engine/cdn/storage"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/cdn/item"
	cdntest "github.com/ovh/cds/engine/cdn/test"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
)

func TestLoadOldItemUnitByItemStatusAndDuration(t *testing.T) {
	m := gorpmapper.New()
	item.InitDBMapping(m)
	storage.InitDBMapping(m)
	db, _ := test.SetupPGWithMapper(t, m, sdk.TypeCDN)
	cfg := test.LoadTestingConf(t, sdk.TypeCDN)

	cdntest.ClearItem(t, context.TODO(), m, db)

	cdnUnits, err := storage.Init(context.TODO(), m, db.DbMap, sdk.NewGoRoutines(), storage.Configuration{
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
	itemUnits, err := storage.LoadOldItemUnitByItemStatusAndDuration(context.TODO(), m, db, sdk.CDNStatusItemIncoming, 1)
	require.NoError(t, err)
	for _, itemUnit := range itemUnits {
		i, err := item.LoadByID(context.TODO(), m, db, itemUnit.ItemID)
		require.NoError(t, err)
		require.NoError(t, item.DeleteByIDs(db, []string{i.ID}))
	}

	i1 := &sdk.CDNItem{
		ID:         sdk.UUID(),
		APIRefHash: sdk.UUID(),
		Type:       sdk.CDNTypeItemStepLog,
		Status:     sdk.CDNStatusItemCompleted,
	}
	err = item.Insert(context.TODO(), m, db, i1)
	require.NoError(t, err)
	defer func() {
		_ = item.DeleteByIDs(db, []string{i1.ID})
	}()

	itemUnit1, err := cdnUnits.NewItemUnit(context.TODO(), cdnUnits.Buffer, i1)
	require.NoError(t, err)
	require.NoError(t, storage.InsertItemUnit(context.TODO(), m, db, itemUnit1))

	i2 := &sdk.CDNItem{
		ID:         sdk.UUID(),
		APIRefHash: sdk.UUID(),
		Type:       sdk.CDNTypeItemStepLog,
		Status:     sdk.CDNStatusItemIncoming,
	}
	err = item.Insert(context.TODO(), m, db, i2)
	require.NoError(t, err)
	defer func() {
		_ = item.DeleteByIDs(db, []string{i2.ID})
	}()
	itemUnit2, err := cdnUnits.NewItemUnit(context.TODO(), cdnUnits.Buffer, i2)
	require.NoError(t, err)

	require.NoError(t, storage.InsertItemUnit(context.TODO(), m, db, itemUnit2))

	time.Sleep(2 * time.Second)

	itemUnits, err = storage.LoadOldItemUnitByItemStatusAndDuration(context.TODO(), m, db, sdk.CDNStatusItemIncoming, 1)
	require.NoError(t, err)
	require.Len(t, itemUnits, 1)
	require.Equal(t, i2.ID, itemUnits[0].ItemID)
}
