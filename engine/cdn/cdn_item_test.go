package cdn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/lru"
	"github.com/ovh/cds/engine/cdn/redis"
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
	}, sdk.NewGoRoutines())
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
	require.NoError(t, s.Units.Buffer.Add(iu, 0, "Line 0\n"))
	require.NoError(t, s.Units.Buffer.Add(iu, 1, "Line 1\n"))
	require.NoError(t, s.Units.Buffer.Add(iu, 2, "Line 2\n"))
	require.NoError(t, s.Units.Buffer.Add(iu, 3, "Line 3\n"))
	require.NoError(t, s.Units.Buffer.Add(iu, 4, "Line 4\n"))
	require.NoError(t, s.Units.Buffer.Add(iu, 5, "Line 5\n"))
	require.NoError(t, s.Units.Buffer.Add(iu, 6, "Line 6\n"))
	require.NoError(t, s.Units.Buffer.Add(iu, 7, "Line 7\n"))
	require.NoError(t, s.Units.Buffer.Add(iu, 8, "Line 8\n"))
	require.NoError(t, s.Units.Buffer.Add(iu, 9, "Line 9\n"))
	require.NoError(t, s.Units.Buffer.Add(iu, 10, "Line 10\n"))

	require.NoError(t, s.completeItem(context.TODO(), db, iu))
	itemDB, err := item.LoadByID(context.TODO(), s.Mapper, db, it.ID, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)
	itemUnit, err := s.Units.NewItemUnit(context.TODO(), s.Units.Buffer, itemDB)
	require.NoError(t, err)
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, itemUnit))

	// Get From Buffer
	rc, err := s.getItemLogValue(context.Background(), sdk.CDNTypeItemStepLog, apiRefhash, redis.ReaderFormatText, 3, 5)
	require.NoError(t, err)

	buf := new(strings.Builder)
	_, err = io.Copy(buf, rc)
	require.NoError(t, err)

	require.Equal(t, "Line 3\nLine 4\nLine 5\nLine 6\nLine 7\n", buf.String())
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
	rc2, err := s.getItemLogValue(context.Background(), sdk.CDNTypeItemStepLog, apiRefhash, redis.ReaderFormatText, 3, 3)
	require.NoError(t, err)

	buf2 := new(strings.Builder)
	_, err = io.Copy(buf2, rc2)
	require.NoError(t, err)

	require.Equal(t, "Line 3\nLine 4\nLine 5\n", buf2.String())
	n, err = s.LogCache.Len()
	require.NoError(t, err)
	require.Equal(t, 1, n)

	// Get all from cache
	rc3, err := s.getItemLogValue(context.Background(), sdk.CDNTypeItemStepLog, apiRefhash, redis.ReaderFormatText, 0, 0)
	require.NoError(t, err)

	buf3 := new(strings.Builder)
	_, err = io.Copy(buf3, rc3)
	require.NoError(t, err)
	require.Equal(t, "Line 0\nLine 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6\nLine 7\nLine 8\nLine 9\nLine 10\n", buf3.String())

	// Get lines from end
	rc4, err := s.getItemLogValue(context.Background(), sdk.CDNTypeItemStepLog, apiRefhash, redis.ReaderFormatText, -3, 2)
	require.NoError(t, err)

	buf4 := new(strings.Builder)
	_, err = io.Copy(buf4, rc4)
	require.NoError(t, err)
	require.Equal(t, "Line 8\nLine 9\n", buf4.String())

	// Get lines from end in JSON
	rc5, err := s.getItemLogValue(context.Background(), sdk.CDNTypeItemStepLog, apiRefhash, redis.ReaderFormatJSON, -3, 2)
	require.NoError(t, err)

	buf5 := new(strings.Builder)
	_, err = io.Copy(buf5, rc5)
	require.NoError(t, err)
	require.Equal(t, "[{\"number\":8,\"value\":\"Line 8\\n\"},{\"number\":9,\"value\":\"Line 9\\n\"}]", buf5.String())
}

func TestGetItemValue_ThousandLines(t *testing.T) {
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

	cdnUnits, err := storage.Init(context.TODO(), m, db.DbMap, storage.Configuration{
		Buffer: storage.BufferConfiguration{
			Name: "redis_buffer",
			Redis: storage.RedisBufferConfiguration{
				Host:     cfg["redisHost"],
				Password: cfg["redisPassword"],
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
	for i := 0; i < 1000; i++ {
		require.NoError(t, s.Units.Buffer.Add(iu, uint(i), fmt.Sprintf("Line %d\n", i)))
	}

	require.NoError(t, s.completeItem(context.TODO(), db, iu))
	itemDB, err := item.LoadByID(context.TODO(), s.Mapper, db, it.ID, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)
	itemUnit, err := s.Units.NewItemUnit(context.TODO(), s.Units.Buffer, itemDB)
	require.NoError(t, err)
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, itemUnit))

	// Get From Buffer
	rc, err := s.getItemLogValue(context.Background(), sdk.CDNTypeItemStepLog, apiRefhash, redis.ReaderFormatJSON, 773, 200)
	require.NoError(t, err)

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, rc)
	require.NoError(t, err)

	var lines []redis.Line
	require.NoError(t, json.Unmarshal(buf.Bytes(), &lines), "given json should be valid")
	require.Len(t, lines, 200)
	require.Equal(t, int64(773), lines[0].Number)
	require.Equal(t, "Line 773\n", lines[0].Value)
	require.Equal(t, int64(972), lines[199].Number)
	require.Equal(t, "Line 972\n", lines[199].Value)

	rc, err = s.getItemLogValue(context.Background(), sdk.CDNTypeItemStepLog, apiRefhash, redis.ReaderFormatJSON, 773, 0)
	require.NoError(t, err)

	buf = new(bytes.Buffer)
	_, err = io.Copy(buf, rc)
	require.NoError(t, err)

	require.NoError(t, json.Unmarshal(buf.Bytes(), &lines), "given json should be valid")
	require.Len(t, lines, 227)
	require.Equal(t, int64(773), lines[0].Number)
	require.Equal(t, "Line 773\n", lines[0].Value)
	require.Equal(t, int64(999), lines[226].Number)
	require.Equal(t, "Line 999\n", lines[226].Value)
}
