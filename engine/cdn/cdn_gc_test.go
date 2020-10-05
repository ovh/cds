package cdn

import (
	"context"
	"io/ioutil"
	"testing"
	"time"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/lru"
	"github.com/ovh/cds/engine/cdn/storage"
	cdntest "github.com/ovh/cds/engine/cdn/test"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/symmecrypt/ciphers/aesgcm"
	"github.com/ovh/symmecrypt/convergent"
	"github.com/stretchr/testify/require"
)

func TestCleanSynchronizedItem(t *testing.T) {
	m := gorpmapper.New()
	item.InitDBMapping(m)
	storage.InitDBMapping(m)

	log.SetLogger(t)
	db, factory, cache, cancel := test.SetupPGToCancel(t, m, sdk.TypeCDN)
	t.Cleanup(cancel)

	cfg := test.LoadTestingConf(t, sdk.TypeCDN)

	cdntest.ClearItem(t, context.TODO(), m, db)
	cdntest.ClearUnits(t, context.TODO(), m, db)

	// Create cdn service
	s := Service{
		DBConnectionFactory: factory,
		Cache:               cache,
		Mapper:              m,
	}

	tmpDir, err := ioutil.TempDir("", t.Name()+"-cdn-1-*")
	require.NoError(t, err)

	cdnUnits, err := storage.Init(context.TODO(), m, db.DbMap, sdk.NewGoRoutines(), storage.Configuration{
		Buffer: storage.BufferConfiguration{
			Name: "redis_buffer",
			Redis: storage.RedisBufferConfiguration{
				Host:     cfg["redisHost"],
				Password: cfg["redisPassword"],
			},
		},
		Storages: []storage.StorageConfiguration{
			{
				Name: "fs-backend",
				Cron: "* * * * * ?",
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
			{
				Name: "cds-backend",
				Cron: "* * * * * ?",
				CDS: &storage.CDSStorageConfiguration{
					Host:  "lolcat.host",
					Token: "mytoken",
				},
			},
		},
	})
	require.NoError(t, err)
	s.Units = cdnUnits

	// Add Item in CDS and FS
	item1CDSFs := sdk.CDNItem{
		ID:         sdk.UUID(),
		Type:       sdk.CDNTypeItemStepLog,
		Status:     sdk.CDNStatusItemCompleted,
		APIRefHash: sdk.RandomString(10),
	}
	require.NoError(t, item.Insert(context.TODO(), s.Mapper, db, &item1CDSFs))
	iu1CDS := sdk.CDNItemUnit{UnitID: s.Units.Storages[1].ID(), ItemID: item1CDSFs.ID}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, &iu1CDS))
	iu1FS := sdk.CDNItemUnit{UnitID: s.Units.Storages[0].ID(), ItemID: item1CDSFs.ID}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, &iu1FS))

	// Add Item in Redis and FS
	item2RedisFs := sdk.CDNItem{
		ID:         sdk.UUID(),
		Type:       sdk.CDNTypeItemStepLog,
		Status:     sdk.CDNStatusItemCompleted,
		APIRefHash: sdk.RandomString(10),
	}
	require.NoError(t, item.Insert(context.TODO(), s.Mapper, db, &item2RedisFs))
	iu2Redis := sdk.CDNItemUnit{UnitID: s.Units.Buffer.ID(), ItemID: item2RedisFs.ID}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, &iu2Redis))
	iu2FS := sdk.CDNItemUnit{UnitID: s.Units.Storages[0].ID(), ItemID: item2RedisFs.ID}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, &iu2FS))

	// Add Item in FS only
	item3Fs := sdk.CDNItem{
		ID:         sdk.UUID(),
		Type:       sdk.CDNTypeItemStepLog,
		Status:     sdk.CDNStatusItemCompleted,
		APIRefHash: sdk.RandomString(10),
	}
	require.NoError(t, item.Insert(context.TODO(), s.Mapper, db, &item3Fs))
	iu3FS := sdk.CDNItemUnit{UnitID: s.Units.Storages[0].ID(), ItemID: item3Fs.ID}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, &iu3FS))

	// Add Item in redis only
	item4Redis := sdk.CDNItem{
		ID:         sdk.UUID(),
		Type:       sdk.CDNTypeItemStepLog,
		Status:     sdk.CDNStatusItemCompleted,
		APIRefHash: sdk.RandomString(10),
	}
	require.NoError(t, item.Insert(context.TODO(), s.Mapper, db, &item4Redis))
	iu4Redis := sdk.CDNItemUnit{UnitID: s.Units.Buffer.ID(), ItemID: item4Redis.ID}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, &iu4Redis))

	// Add Item in cds only
	item5CDS := sdk.CDNItem{
		ID:         sdk.UUID(),
		Type:       sdk.CDNTypeItemStepLog,
		Status:     sdk.CDNStatusItemCompleted,
		APIRefHash: sdk.RandomString(10),
	}
	require.NoError(t, item.Insert(context.TODO(), s.Mapper, db, &item5CDS))
	iu5CDS := sdk.CDNItemUnit{UnitID: s.Units.Storages[1].ID(), ItemID: item5CDS.ID}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, &iu5CDS))

	// Add Item in redis / fs/ cds
	item6RedisFSCDS := sdk.CDNItem{
		ID:         sdk.UUID(),
		Type:       sdk.CDNTypeItemStepLog,
		Status:     sdk.CDNStatusItemCompleted,
		APIRefHash: sdk.RandomString(10),
	}
	require.NoError(t, item.Insert(context.TODO(), s.Mapper, db, &item6RedisFSCDS))
	iu6CDS := sdk.CDNItemUnit{UnitID: s.Units.Storages[1].ID(), ItemID: item6RedisFSCDS.ID}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, &iu6CDS))
	iu6Redis := sdk.CDNItemUnit{UnitID: s.Units.Buffer.ID(), ItemID: item6RedisFSCDS.ID}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, &iu6Redis))
	iu6FS := sdk.CDNItemUnit{UnitID: s.Units.Storages[0].ID(), ItemID: item6RedisFSCDS.ID}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, &iu6FS))

	iusRedis, err := storage.LoadItemUnitsByUnit(context.TODO(), s.Mapper, db, s.Units.Buffer.ID(), 100)
	require.NoError(t, err)
	require.Equal(t, 3, len(iusRedis))

	iusFS, err := storage.LoadItemUnitsByUnit(context.TODO(), s.Mapper, db, s.Units.Storages[0].ID(), 100)
	require.NoError(t, err)
	require.Equal(t, 4, len(iusFS))

	iusCDS, err := storage.LoadItemUnitsByUnit(context.TODO(), s.Mapper, db, s.Units.Storages[1].ID(), 100)
	require.NoError(t, err)
	require.Equal(t, 3, len(iusCDS))

	// RUN TEST
	require.NoError(t, s.cleanBuffer(context.TODO()))

	iusRedisAfter, err := storage.LoadItemUnitsByUnit(context.TODO(), s.Mapper, db, s.Units.Buffer.ID(), 100)
	require.NoError(t, err)
	require.Equal(t, 1, len(iusRedisAfter))
	require.Equal(t, item4Redis.ID, iusRedisAfter[0].ItemID)

	iusFS2After, err := storage.LoadItemUnitsByUnit(context.TODO(), s.Mapper, db, s.Units.Storages[0].ID(), 100)
	require.NoError(t, err)
	require.Equal(t, 4, len(iusFS2After))

	iusCDSAfter, err := storage.LoadItemUnitsByUnit(context.TODO(), s.Mapper, db, s.Units.Storages[1].ID(), 100)
	require.NoError(t, err)
	require.Equal(t, 3, len(iusCDSAfter))

}

func TestCleanWaitingItem(t *testing.T) {
	m := gorpmapper.New()
	item.InitDBMapping(m)
	storage.InitDBMapping(m)

	log.SetLogger(t)
	db, factory, cache, cancel := test.SetupPGToCancel(t, m, sdk.TypeCDN)
	t.Cleanup(cancel)

	cdntest.ClearItem(t, context.TODO(), m, db)

	// Create cdn service
	s := Service{
		DBConnectionFactory: factory,
		Cache:               cache,
		Mapper:              m,
	}

	cdnUnits := newRunningStorageUnits(t, m, s.DBConnectionFactory.GetDBMap(m)())
	s.Units = cdnUnits

	it := sdk.CDNItem{
		ID:     sdk.UUID(),
		Size:   12,
		Type:   sdk.CDNTypeItemStepLog,
		Status: sdk.CDNStatusItemIncoming,

		APIRefHash: sdk.RandomString(10),
	}
	require.NoError(t, item.Insert(context.TODO(), s.Mapper, db, &it))

	iu := sdk.CDNItemUnit{
		ItemID: it.ID,
		UnitID: s.Units.Buffer.ID(),
	}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, &iu))

	time.Sleep(2 * time.Second)

	require.NoError(t, s.cleanWaitingItem(context.TODO(), 1))

	itemDB, err := item.LoadByID(context.TODO(), s.Mapper, db, it.ID)
	require.NoError(t, err)

	require.Equal(t, sdk.CDNStatusItemCompleted, itemDB.Status)
}

func TestPurgeItem(t *testing.T) {
	m := gorpmapper.New()
	item.InitDBMapping(m)
	storage.InitDBMapping(m)

	log.SetLogger(t)
	db, factory, cache, cancel := test.SetupPGToCancel(t, m, sdk.TypeCDN)
	t.Cleanup(cancel)

	cdntest.ClearItem(t, context.TODO(), m, db)

	// Create cdn service
	s := Service{
		DBConnectionFactory: factory,
		Cache:               cache,
		Mapper:              m,
	}

	var err error
	cfg := test.LoadTestingConf(t, sdk.TypeCDN)
	s.LogCache, err = lru.NewRedisLRU(db.DbMap, 1000, cfg["redisHost"], cfg["redisPassword"])
	require.NoError(t, err)

	// Add Item in CDS and FS
	item1 := sdk.CDNItem{
		ID:         sdk.UUID(),
		Type:       sdk.CDNTypeItemStepLog,
		Status:     sdk.CDNStatusItemCompleted,
		APIRefHash: sdk.RandomString(10),
		ToDelete:   true,
	}
	require.NoError(t, item.Insert(context.TODO(), s.Mapper, db, &item1))

	item2 := sdk.CDNItem{
		ID:         sdk.UUID(),
		Type:       sdk.CDNTypeItemStepLog,
		Status:     sdk.CDNStatusItemCompleted,
		APIRefHash: sdk.RandomString(10),
		ToDelete:   false,
	}
	require.NoError(t, item.Insert(context.TODO(), s.Mapper, db, &item2))

	item3 := sdk.CDNItem{
		ID:         sdk.UUID(),
		Type:       sdk.CDNTypeItemStepLog,
		Status:     sdk.CDNStatusItemCompleted,
		APIRefHash: sdk.RandomString(10),
		ToDelete:   true,
	}
	require.NoError(t, item.Insert(context.TODO(), s.Mapper, db, &item3))

	require.NoError(t, s.cleanItemToDelete(context.TODO()))

	items, err := item.LoadAll(context.TODO(), s.Mapper, db, 10)
	require.NoError(t, err)
	require.Equal(t, 1, len(items))
}
