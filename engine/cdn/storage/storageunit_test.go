package storage_test

import (
	"context"
	"io/ioutil"
	"testing"
	"time"

	"github.com/ovh/cds/engine/cdn/index"
	_ "github.com/ovh/cds/engine/cdn/storage/local"
	_ "github.com/ovh/cds/engine/cdn/storage/redis"

	"github.com/ovh/cds/engine/api/test"
	commontest "github.com/ovh/cds/engine/test"

	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	m := gorpmapper.New()
	index.InitDBMapping(m)
	storage.InitDBMapping(m)

	db, _ := test.SetupPGWithMapper(t, m, sdk.TypeCDN)
	cfg := commontest.LoadTestingConf(t, sdk.TypeCDN)

	ctx, cancel := context.WithTimeout(context.TODO(), 2*time.Second)
	defer cancel()

	tmpDir, err := ioutil.TempDir("", t.Name()+"-cdn-*")
	require.NoError(t, err)

	cdnUnits, err := storage.Init(ctx, m, db.DbMap, storage.Configuration{
		Buffer: storage.BufferConfiguration{
			Name: "redis_buffer",
			Redis: storage.RedisBufferConfiguration{
				Host:     cfg["redisHost"],
				Password: cfg["redisPassword"],
			},
		},
		Storages: []storage.StorageConfiguration{
			{
				Name:     "local_storage",
				CronExpr: "* * * * * ?",
				Local: &storage.LocalStorageConfiguration{
					Path: tmpDir,
				},
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, cdnUnits)

	units, err := storage.LoadAllUnits(ctx, m, db.DbMap)
	require.NoError(t, err)
	require.NotNil(t, units)
	require.NotEmpty(t, units)

	i := index.Item{
		ID: sdk.UUID(),
	}
	require.NoError(t, index.InsertItem(ctx, m, db, &i))

	require.NoError(t, cdnUnits.Buffer.Add(i, 1.0, "this is the first log"))
	require.NoError(t, cdnUnits.Buffer.Add(i, 1.0, "this is the second log"))

	redisUnit, err := storage.LoadUnitByName(ctx, m, db, "redis_buffer")
	require.NoError(t, err)

	itemUnit, err := storage.InsertItemUnit(ctx, m, db, *redisUnit, i)
	require.NoError(t, err)
	require.NotNil(t, itemUnit)

	localUnit, err := storage.LoadUnitByName(ctx, m, db, "local_storage")
	require.NoError(t, err)

	localUnitDriver := cdnUnits.Storage(localUnit.Name)
	require.NotNil(t, localUnitDriver)

	exists, err := localUnitDriver.ItemExists(i)
	require.NoError(t, err)
	require.False(t, exists)

	<-ctx.Done()

	exists, err = localUnitDriver.ItemExists(i)
	require.NoError(t, err)
	require.False(t, exists)

	reader, err := localUnitDriver.NewReader(i)
	require.NoError(t, err)
	btes, err := ioutil.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	require.Equal(t, `this is the first log
this is the second log`, string(btes))

}
