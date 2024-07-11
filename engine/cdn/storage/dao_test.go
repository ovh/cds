package storage_test

import (
	"context"
	"os"
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

	tmpDir, err := os.MkdirTemp("", t.Name()+"-cdn-1-*")
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	t.Cleanup(cancel)

	cdnUnits, err := storage.Init(ctx, m, store, db.DbMap, sdk.NewGoRoutines(ctx), storage.Configuration{
		HashLocatorSalt: "thisismysalt",
		Buffers: map[string]storage.BufferConfiguration{
			"redis_buffer": {
				Redis: &sdk.RedisConf{
					Host:     cfg["redisHost"],
					Password: cfg["redisPassword"],
					DbIndex:  0,
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
