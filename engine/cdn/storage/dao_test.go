package storage_test

import (
	"context"
	"io/ioutil"
	"testing"
	"time"

	"github.com/ovh/symmecrypt/ciphers/aesgcm"
	"github.com/ovh/symmecrypt/convergent"

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
	db, store := test.SetupPGWithMapper(t, m, sdk.TypeCDN)
	cfg := test.LoadTestingConf(t, sdk.TypeCDN)

	cdntest.ClearItem(t, context.TODO(), m, db)
	tmpDir, err := ioutil.TempDir("", t.Name()+"-cdn-1-*")

	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	t.Cleanup(cancel)

	cdnUnits, err := storage.Init(ctx, m, store, db.DbMap, sdk.NewGoRoutines(), storage.Configuration{
		HashLocatorSalt: "thisismysalt",
		Buffers: map[string]storage.BufferConfiguration{
			"redis_buffer": {
				Redis: &storage.RedisBufferConfiguration{
					Host:     cfg["redisHost"],
					Password: cfg["redisPassword"],
				},
				BufferType: storage.CDNBufferTypeLog,
			},
		},
		Storages: map[string]storage.StorageConfiguration{
			"local_storage": {
				Local: &storage.LocalStorageConfiguration{
					Path: tmpDir,
				},
			},
		},
	})
	require.NoError(t, err)

	// Clean old test
	time.Sleep(1 * time.Second)
	itemUnits, err := storage.LoadOldItemUnitByItemStatusAndDuration(context.TODO(), m, db, sdk.CDNStatusItemIncoming, 1)
	require.NoError(t, err)
	for _, itemUnit := range itemUnits {
		i, err := item.LoadByID(context.TODO(), m, db, itemUnit.ItemID)
		require.NoError(t, err)
		require.NoError(t, item.DeleteByID(db, i.ID))
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
		_ = item.DeleteByID(db, i1.ID)
	}()

	itemUnit1, err := cdnUnits.NewItemUnit(context.TODO(), cdnUnits.LogsBuffer(), i1)
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
		_ = item.DeleteByID(db, i2.ID)
	}()
	itemUnit2, err := cdnUnits.NewItemUnit(context.TODO(), cdnUnits.LogsBuffer(), i2)
	require.NoError(t, err)

	require.NoError(t, storage.InsertItemUnit(context.TODO(), m, db, itemUnit2))

	time.Sleep(2 * time.Second)

	itemUnits, err = storage.LoadOldItemUnitByItemStatusAndDuration(context.TODO(), m, db, sdk.CDNStatusItemIncoming, 1)
	require.NoError(t, err)
	require.Len(t, itemUnits, 1)
	require.Equal(t, i2.ID, itemUnits[0].ItemID)
}

func TestLoadAllItemIDUnknownByUnit(t *testing.T) {
	m := gorpmapper.New()
	item.InitDBMapping(m)
	storage.InitDBMapping(m)
	db, store := test.SetupPGWithMapper(t, m, sdk.TypeCDN)
	cfg := test.LoadTestingConf(t, sdk.TypeCDN)

	cdntest.ClearItem(t, context.TODO(), m, db)
	cdntest.ClearUnits(t, context.TODO(), m, db)

	i1 := sdk.CDNItem{ID: sdk.UUID(), APIRefHash: sdk.RandomString(10), Status: sdk.CDNStatusItemCompleted}
	require.NoError(t, item.Insert(context.TODO(), m, db, &i1))

	i2 := sdk.CDNItem{ID: sdk.UUID(), APIRefHash: sdk.RandomString(10), Status: sdk.CDNStatusItemCompleted}
	require.NoError(t, item.Insert(context.TODO(), m, db, &i2))

	i3 := sdk.CDNItem{ID: sdk.UUID(), APIRefHash: sdk.RandomString(10), Status: sdk.CDNStatusItemCompleted}
	require.NoError(t, item.Insert(context.TODO(), m, db, &i3))

	i4 := sdk.CDNItem{ID: sdk.UUID(), APIRefHash: sdk.RandomString(10), Status: sdk.CDNStatusItemCompleted}
	require.NoError(t, item.Insert(context.TODO(), m, db, &i4))

	tmpDir, err := ioutil.TempDir("", t.Name()+"-cdn-1-*")
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	t.Cleanup(cancel)

	cdnUnits, err := storage.Init(ctx, m, store, db.DbMap, sdk.NewGoRoutines(), storage.Configuration{
		HashLocatorSalt: "thisismysalt",
		Buffers: map[string]storage.BufferConfiguration{
			"redis_buffer": {
				Redis: &storage.RedisBufferConfiguration{
					Host:     cfg["redisHost"],
					Password: cfg["redisPassword"],
				},
				BufferType: storage.CDNBufferTypeLog,
			},
		},
		Storages: map[string]storage.StorageConfiguration{
			"local_storage": {
				Local: &storage.LocalStorageConfiguration{
					Path: tmpDir,
					Encryption: []convergent.ConvergentEncryptionConfig{
						{
							Cipher:      aesgcm.CipherName,
							LocatorSalt: "secret_locator_salt",
							SecretValue: "secret_value",
						},
					},
				},
			},
			"local_storage2": {
				Local: &storage.LocalStorageConfiguration{
					Path: tmpDir,
					Encryption: []convergent.ConvergentEncryptionConfig{
						{
							Cipher:      aesgcm.CipherName,
							LocatorSalt: "secret_locator_salt",
							SecretValue: "secret_value",
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	iu1 := sdk.CDNItemUnit{
		ID:     sdk.UUID(),
		ItemID: i1.ID,
		UnitID: cdnUnits.Storages[1].ID(),
		Type:   i1.Type,
	}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), m, db, &iu1))

	iu2 := sdk.CDNItemUnit{
		ID:     sdk.UUID(),
		ItemID: i2.ID,
		UnitID: cdnUnits.LogsBuffer().ID(),
		Type:   i2.Type,
	}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), m, db, &iu2))

	iu3 := sdk.CDNItemUnit{
		ID:     sdk.UUID(),
		ItemID: i3.ID,
		UnitID: cdnUnits.Storages[0].ID(),
		Type:   i3.Type,
	}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), m, db, &iu3))

	iu4 := sdk.CDNItemUnit{
		ID:     sdk.UUID(),
		ItemID: i4.ID,
		UnitID: cdnUnits.Storages[1].ID(),
		Type:   i4.Type,
	}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), m, db, &iu4))

	itemIDS, err := storage.LoadAllItemIDUnknownByUnit(db, cdnUnits.Storages[0].ID(), 0, 100)
	require.NoError(t, err)

	require.Equal(t, 3, len(itemIDS))

}
