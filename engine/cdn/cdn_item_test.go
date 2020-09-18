package cdn

import (
	"context"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/lru"
	"github.com/ovh/cds/engine/cdn/storage"
	cdntest "github.com/ovh/cds/engine/cdn/test"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func TestGetItemValue(t *testing.T) {
	m := gorpmapper.New()
	item.InitDBMapping(m)
	storage.InitDBMapping(m)

	log.SetLogger(t)
	db, factory, cache, cancel := test.SetupPGToCancel(t, m, sdk.TypeCDN)
	t.Cleanup(cancel)

	cfg := test.LoadTestingConf(t, sdk.TypeCDN)

	cdntest.ClearItem(t, context.TODO(), m, db)

	// Create cdn service
	s := Service{
		DBConnectionFactory: factory,
		Cache:               cache,
		Mapper:              m,
	}

	tmpDir, err := ioutil.TempDir("", t.Name()+"-cdn-1-*")
	require.NoError(t, err)

	cdnUnits, err := storage.Init(context.TODO(), m, db.DbMap, storage.Configuration{
		Buffer: storage.BufferConfiguration{
			Name: "redis_buffer",
			Redis: storage.RedisBufferConfiguration{
				Host:     cfg["redisHost"],
				Password: cfg["redisPassword"],
			},
		},
		Storages: []storage.StorageConfiguration{
			{
				Name: "local_storage",
				Cron: "* * * * * ?",
				Local: &storage.LocalStorageConfiguration{
					Path: tmpDir,
				},
			},
		},
	})
	require.NoError(t, err)
	s.Units = cdnUnits
	s.LogCache, err = lru.NewRedisLRU(db.DbMap, 1000, cfg["redisHost"], cfg["redisPassword"])
	require.NoError(t, err)
	require.NoError(t, s.LogCache.Clear())

	apiRef := sdk.CDNLogAPIRef{
		ProjectKey:     sdk.RandomString(10),
		WorkflowName:   sdk.RandomString(10),
		WorkflowID:     1,
		RunID:          1,
		NodeRunID:      1,
		NodeRunName:    sdk.RandomString(10),
		NodeRunJobID:   1,
		NodeRunJobName: sdk.RandomString(10),
		StepName:       sdk.RandomString(10),
		StepOrder:      0,
	}
	apiRefhash, err := apiRef.ToHash()
	require.NoError(t, err)

	it := sdk.CDNItem{
		ID:         sdk.UUID(),
		APIRefHash: apiRefhash,
		Type:       sdk.CDNTypeItemStepLog,
		Status:     sdk.CDNStatusItemIncoming,
		APIRef:     apiRef,
	}
	require.NoError(t, item.Insert(context.TODO(), s.Mapper, db, &it))
	iu := sdk.CDNItemUnit{
		Item:   &it,
		ItemID: it.ID,
		UnitID: s.Units.Buffer.ID(),
	}
	require.NoError(t, s.Units.Buffer.Add(iu, 0, "Ligne 0\n"))
	require.NoError(t, s.Units.Buffer.Add(iu, 1, "Ligne 1\n"))
	require.NoError(t, s.Units.Buffer.Add(iu, 2, "Ligne 2\n"))
	require.NoError(t, s.Units.Buffer.Add(iu, 3, "Ligne 3\n"))
	require.NoError(t, s.Units.Buffer.Add(iu, 4, "Ligne 4\n"))
	require.NoError(t, s.Units.Buffer.Add(iu, 5, "Ligne 5\n"))
	require.NoError(t, s.Units.Buffer.Add(iu, 6, "Ligne 6\n"))
	require.NoError(t, s.Units.Buffer.Add(iu, 7, "Ligne 7\n"))
	require.NoError(t, s.Units.Buffer.Add(iu, 8, "Ligne 8\n"))
	require.NoError(t, s.Units.Buffer.Add(iu, 9, "Ligne 9\n"))
	require.NoError(t, s.Units.Buffer.Add(iu, 10, "Ligne 10\n"))

	require.NoError(t, s.completeItem(context.TODO(), db, iu))
	itemDB, err := item.LoadByID(context.TODO(), s.Mapper, db, it.ID, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)
	itemUnit, err := s.Units.NewItemUnit(context.TODO(), s.Units.Buffer, itemDB)
	require.NoError(t, err)
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, itemUnit))

	// Get From Buffer
	rc, err := s.getItemLogValue(context.Background(), sdk.CDNTypeItemStepLog, apiRefhash, 3, 5)
	require.NoError(t, err)

	buf := new(strings.Builder)
	_, err = io.Copy(buf, rc)
	require.NoError(t, err)

	require.Equal(t, "Ligne 3\nLigne 4\nLigne 5\nLigne 6\nLigne 7\n", buf.String())
	n, err := s.LogCache.Len()
	require.NoError(t, err)
	require.Equal(t, 0, n)

	// Sync FS
	require.NoError(t, cdnUnits.Run(context.TODO(), cdnUnits.Storages[0]))

	_, err = storage.LoadItemUnitByUnit(context.TODO(), s.Mapper, db, s.Units.Storages[0].ID(), it.ID)
	require.NoError(t, err)
	// remove from buffer
	require.NoError(t, storage.DeleteItemUnit(s.Mapper, db, itemUnit))

	// Get From Storage
	rc2, err := s.getItemLogValue(context.Background(), sdk.CDNTypeItemStepLog, apiRefhash, 3, 3)
	require.NoError(t, err)

	buf2 := new(strings.Builder)
	_, err = io.Copy(buf2, rc2)
	require.NoError(t, err)

	require.Equal(t, "Ligne 3\nLigne 4\nLigne 5\n", buf2.String())
	n, err = s.LogCache.Len()
	require.NoError(t, err)
	require.Equal(t, 1, n)

	// Get all from cache
	rc3, err := s.getItemLogValue(context.Background(), sdk.CDNTypeItemStepLog, apiRefhash, 0, 0)
	require.NoError(t, err)

	buf3 := new(strings.Builder)
	_, err = io.Copy(buf3, rc3)
	require.NoError(t, err)
	require.Equal(t, "Ligne 0\nLigne 1\nLigne 2\nLigne 3\nLigne 4\nLigne 5\nLigne 6\nLigne 7\nLigne 8\nLigne 9\nLigne 10\n", buf3.String())
}
