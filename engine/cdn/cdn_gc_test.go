package cdn

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ovh/symmecrypt/ciphers/aesgcm"
	"github.com/ovh/symmecrypt/convergent"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/lru"
	"github.com/ovh/cds/engine/cdn/storage"
	cdntest "github.com/ovh/cds/engine/cdn/test"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
)

func TestCleanSynchronizedItem(t *testing.T) {
	m := gorpmapper.New()
	item.InitDBMapping(m)
	storage.InitDBMapping(m)

	log.Factory = log.NewTestingWrapper(t)
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
	s.GoRoutines = sdk.NewGoRoutines(context.TODO())

	tmpDir, err := os.MkdirTemp("", t.Name()+"-cdn-1-*")
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	t.Cleanup(cancel)

	cdnUnits, err := storage.Init(ctx, m, cache, db.DbMap, sdk.NewGoRoutines(ctx), storage.Configuration{
		HashLocatorSalt: "thisismysalt",
		Buffers: map[string]storage.BufferConfiguration{
			"redis_buffer": {
				Redis: &storage.RedisBufferConfiguration{
					Host:     cfg["redisHost"],
					Password: cfg["redisPassword"],
					DbIndex:  0,
				},
				BufferType: storage.CDNBufferTypeLog,
			},
		},
		Storages: map[string]storage.StorageConfiguration{
			"fs-backend": {
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
	s.Units = cdnUnits

	// Add Item in Redis and FS - have to stay in redis
	item2RedisFs := sdk.CDNItem{
		ID:         sdk.UUID(),
		Type:       sdk.CDNTypeItemStepLog,
		Status:     sdk.CDNStatusItemCompleted,
		APIRefHash: sdk.RandomString(10),
	}
	require.NoError(t, item.Insert(context.TODO(), s.Mapper, db, &item2RedisFs))
	iu2Redis := sdk.CDNItemUnit{UnitID: s.Units.LogsBuffer().ID(), ItemID: item2RedisFs.ID, Type: item2RedisFs.Type}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, &iu2Redis))
	iu2FS := sdk.CDNItemUnit{UnitID: s.Units.Storages[0].ID(), ItemID: item2RedisFs.ID, Type: item2RedisFs.Type}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, &iu2FS))

	// Add Item in FS only
	item3Fs := sdk.CDNItem{
		ID:         sdk.UUID(),
		Type:       sdk.CDNTypeItemStepLog,
		Status:     sdk.CDNStatusItemCompleted,
		APIRefHash: sdk.RandomString(10),
	}
	require.NoError(t, item.Insert(context.TODO(), s.Mapper, db, &item3Fs))
	iu3FS := sdk.CDNItemUnit{UnitID: s.Units.Storages[0].ID(), ItemID: item3Fs.ID, Type: item3Fs.Type}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, &iu3FS))

	// Add Item in redis only - have to stay in redis
	item4Redis := sdk.CDNItem{
		ID:         sdk.UUID(),
		Type:       sdk.CDNTypeItemStepLog,
		Status:     sdk.CDNStatusItemCompleted,
		APIRefHash: sdk.RandomString(10),
	}
	require.NoError(t, item.Insert(context.TODO(), s.Mapper, db, &item4Redis))
	iu4Redis := sdk.CDNItemUnit{UnitID: s.Units.LogsBuffer().ID(), ItemID: item4Redis.ID, Type: item4Redis.Type}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, &iu4Redis))

	// Add Item in redis / fs -will be delete from redis
	item6RedisFS := sdk.CDNItem{
		ID:         sdk.UUID(),
		Type:       sdk.CDNTypeItemStepLog,
		Status:     sdk.CDNStatusItemCompleted,
		APIRefHash: sdk.RandomString(10),
	}
	require.NoError(t, item.Insert(context.TODO(), s.Mapper, db, &item6RedisFS))
	iu6Redis := sdk.CDNItemUnit{UnitID: s.Units.LogsBuffer().ID(), ItemID: item6RedisFS.ID, Type: item6RedisFS.Type}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, &iu6Redis))
	iu6FS := sdk.CDNItemUnit{UnitID: s.Units.Storages[0].ID(), ItemID: item6RedisFS.ID, Type: item6RedisFS.Type}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, &iu6FS))
	oneHundred := 100
	iusRedis, err := storage.LoadItemUnitsByUnit(context.TODO(), s.Mapper, db, s.Units.LogsBuffer().ID(), &oneHundred)
	require.NoError(t, err)
	require.Equal(t, 3, len(iusRedis))

	iusFS, err := storage.LoadItemUnitsByUnit(context.TODO(), s.Mapper, db, s.Units.Storages[0].ID(), &oneHundred)
	require.NoError(t, err)
	require.Equal(t, 3, len(iusFS))

	// RUN TEST
	iusRedisBefore, err := storage.LoadItemUnitsByUnit(context.TODO(), s.Mapper, db, s.Units.LogsBuffer().ID(), &oneHundred)
	require.NoError(t, err)
	require.Equal(t, 3, len(iusRedisBefore))

	require.NoError(t, s.cleanBuffer(context.TODO()))

	iusRedisAfter, err := storage.LoadItemUnitsByUnit(context.TODO(), s.Mapper, db, s.Units.LogsBuffer().ID(), &oneHundred)
	require.NoError(t, err)
	require.Equal(t, 1, len(iusRedisAfter))

	iusFS2After, err := storage.LoadItemUnitsByUnit(context.TODO(), s.Mapper, db, s.Units.Storages[0].ID(), &oneHundred)
	require.NoError(t, err)
	require.Equal(t, 3, len(iusFS2After))
}

func TestCleanSynchronizedItemWithDisabledStorage(t *testing.T) {
	m := gorpmapper.New()
	item.InitDBMapping(m)
	storage.InitDBMapping(m)

	log.Factory = log.NewTestingWrapper(t)
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
	s.GoRoutines = sdk.NewGoRoutines(context.TODO())

	tmpDir, err := os.MkdirTemp("", t.Name()+"-cdn-1-*")
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	t.Cleanup(cancel)

	cdnUnits, err := storage.Init(ctx, m, cache, db.DbMap, sdk.NewGoRoutines(ctx), storage.Configuration{
		HashLocatorSalt: "thisismysalt",
		Buffers: map[string]storage.BufferConfiguration{
			"redis_buffer": {
				Redis: &storage.RedisBufferConfiguration{
					Host:     cfg["redisHost"],
					Password: cfg["redisPassword"],
					DbIndex:  0,
				},
				BufferType: storage.CDNBufferTypeLog,
			},
		},
		Storages: map[string]storage.StorageConfiguration{
			"fs-backend": {
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
	s.Units = cdnUnits

	// Add Item in Redis and FS  -- Must be sync
	item2RedisFs := sdk.CDNItem{
		ID:         sdk.UUID(),
		Type:       sdk.CDNTypeItemStepLog,
		Status:     sdk.CDNStatusItemCompleted,
		APIRefHash: sdk.RandomString(10),
	}
	require.NoError(t, item.Insert(context.TODO(), s.Mapper, db, &item2RedisFs))
	iu2Redis := sdk.CDNItemUnit{UnitID: s.Units.LogsBuffer().ID(), ItemID: item2RedisFs.ID, Type: item2RedisFs.Type}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, &iu2Redis))
	iu2FS := sdk.CDNItemUnit{UnitID: s.Units.Storages[0].ID(), ItemID: item2RedisFs.ID, Type: item2RedisFs.Type}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, &iu2FS))

	// Add Item in redis only - have to stay in redis
	item4Redis := sdk.CDNItem{
		ID:         sdk.UUID(),
		Type:       sdk.CDNTypeItemStepLog,
		Status:     sdk.CDNStatusItemCompleted,
		APIRefHash: sdk.RandomString(10),
	}
	require.NoError(t, item.Insert(context.TODO(), s.Mapper, db, &item4Redis))
	iu4Redis := sdk.CDNItemUnit{UnitID: s.Units.LogsBuffer().ID(), ItemID: item4Redis.ID, Type: item4Redis.Type}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, &iu4Redis))

	// Add Item in redis / fs  -- Must be sync
	item6RedisFS := sdk.CDNItem{
		ID:         sdk.UUID(),
		Type:       sdk.CDNTypeItemStepLog,
		Status:     sdk.CDNStatusItemCompleted,
		APIRefHash: sdk.RandomString(10),
	}
	require.NoError(t, item.Insert(context.TODO(), s.Mapper, db, &item6RedisFS))
	iu6Redis := sdk.CDNItemUnit{UnitID: s.Units.LogsBuffer().ID(), ItemID: item6RedisFS.ID, Type: item6RedisFS.Type}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, &iu6Redis))
	iu6FS := sdk.CDNItemUnit{UnitID: s.Units.Storages[0].ID(), ItemID: item6RedisFS.ID, Type: item6RedisFS.Type}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, &iu6FS))

	oneHundred := 100

	iusFS, err := storage.LoadItemUnitsByUnit(context.TODO(), s.Mapper, db, s.Units.Storages[0].ID(), &oneHundred)
	require.NoError(t, err)
	require.Equal(t, 2, len(iusFS))

	// RUN TEST
	iusRedisBefore, err := storage.LoadItemUnitsByUnit(context.TODO(), s.Mapper, db, s.Units.LogsBuffer().ID(), &oneHundred)
	require.NoError(t, err)
	require.Equal(t, 3, len(iusRedisBefore))

	require.NoError(t, s.cleanBuffer(context.TODO()))

	iusRedisAfter, err := storage.LoadItemUnitsByUnit(context.TODO(), s.Mapper, db, s.Units.LogsBuffer().ID(), &oneHundred)
	require.NoError(t, err)
	require.Equal(t, 1, len(iusRedisAfter))

	iusFS2After, err := storage.LoadItemUnitsByUnit(context.TODO(), s.Mapper, db, s.Units.Storages[0].ID(), &oneHundred)
	require.NoError(t, err)
	require.Equal(t, 2, len(iusFS2After))
}

func TestCleanWaitingItem(t *testing.T) {
	m := gorpmapper.New()
	item.InitDBMapping(m)
	storage.InitDBMapping(m)

	log.Factory = log.NewTestingWrapper(t)
	db, factory, cache, cancel := test.SetupPGToCancel(t, m, sdk.TypeCDN)
	t.Cleanup(cancel)

	cdntest.ClearItem(t, context.TODO(), m, db)

	// Create cdn service
	s := Service{
		DBConnectionFactory: factory,
		Cache:               cache,
		Mapper:              m,
	}
	s.GoRoutines = sdk.NewGoRoutines(context.TODO())

	ctx, cancel := context.WithCancel(context.TODO())
	t.Cleanup(cancel)
	s.Units = newRunningStorageUnits(t, m, s.DBConnectionFactory.GetDBMap(m)(), ctx, cache)

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
		UnitID: s.Units.LogsBuffer().ID(),
		Type:   it.Type,
	}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, &iu))

	time.Sleep(2 * time.Second)

	require.NoError(t, s.cleanWaitingItem(context.TODO(), 1))

	itemDB, err := item.LoadByID(context.TODO(), s.Mapper, db, it.ID)
	require.NoError(t, err)

	require.Equal(t, sdk.CDNStatusItemCompleted, itemDB.Status)
	require.False(t, itemDB.ToDelete)
}

func TestCleanWaitingItemWithoutItemUnit(t *testing.T) {
	m := gorpmapper.New()
	item.InitDBMapping(m)
	storage.InitDBMapping(m)

	log.Factory = log.NewTestingWrapper(t)
	db, factory, cache, cancel := test.SetupPGToCancel(t, m, sdk.TypeCDN)
	t.Cleanup(cancel)

	cdntest.ClearItem(t, context.TODO(), m, db)

	// Create cdn service
	s := Service{
		DBConnectionFactory: factory,
		Cache:               cache,
		Mapper:              m,
	}
	s.GoRoutines = sdk.NewGoRoutines(context.TODO())

	ctx, cancel := context.WithCancel(context.TODO())
	t.Cleanup(cancel)
	s.Units = newRunningStorageUnits(t, m, s.DBConnectionFactory.GetDBMap(m)(), ctx, cache)

	it := sdk.CDNItem{
		ID:     sdk.UUID(),
		Size:   12,
		Type:   sdk.CDNTypeItemStepLog,
		Status: sdk.CDNStatusItemIncoming,

		APIRefHash: sdk.RandomString(10),
	}
	require.NoError(t, item.Insert(context.TODO(), s.Mapper, db, &it))

	time.Sleep(2 * time.Second)

	require.NoError(t, s.cleanWaitingItem(context.TODO(), 1))

	itemDB, err := item.LoadByID(context.TODO(), s.Mapper, db, it.ID)
	require.NoError(t, err)

	require.Equal(t, sdk.CDNStatusItemCompleted, itemDB.Status)
	require.True(t, itemDB.ToDelete)
}

func TestPurgeItem(t *testing.T) {
	m := gorpmapper.New()
	item.InitDBMapping(m)
	storage.InitDBMapping(m)

	log.Factory = log.NewTestingWrapper(t)
	db, factory, cache, cancel := test.SetupPGToCancel(t, m, sdk.TypeCDN)
	t.Cleanup(cancel)

	cdntest.ClearItem(t, context.TODO(), m, db)

	// Create cdn service
	s := Service{
		DBConnectionFactory: factory,
		Cache:               cache,
		Mapper:              m,
	}
	s.GoRoutines = sdk.NewGoRoutines(context.TODO())

	ctx, cancel := context.WithCancel(context.TODO())
	t.Cleanup(cancel)
	s.Units = newRunningStorageUnits(t, m, s.DBConnectionFactory.GetDBMap(m)(), ctx, cache)

	var err error
	cfg := test.LoadTestingConf(t, sdk.TypeCDN)
	s.LogCache, err = lru.NewRedisLRU(db.DbMap, 1000, sdk.RedisConf{Host: cfg["redisHost"], Password: cfg["redisPassword"], DbIndex: 0})
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

	// LoadAll filter only item with flag to_delete set to false
	items, err := item.LoadAll(context.TODO(), s.Mapper, db, 10)
	require.NoError(t, err)
	require.Equal(t, 1, len(items))

	// Check there are 2 item to delete
	ids, err := item.LoadIDsToDelete(db, 0, 10)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(ids), 2)

	// Delete
	require.NoError(t, s.cleanItemToDelete(context.TODO()))

	// Only 1 item should remain
	items, err = item.LoadAll(context.TODO(), s.Mapper, db, 10)
	require.NoError(t, err)
	require.Equal(t, 1, len(items))
}
