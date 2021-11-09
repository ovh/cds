package cdn

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	cdntest "github.com/ovh/cds/engine/cdn/test"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/service"
	commontest "github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
)

func TestSyncBuffer(t *testing.T) {
	m := gorpmapper.New()
	item.InitDBMapping(m)
	storage.InitDBMapping(m)
	db, factory, cache, end := commontest.SetupPGToCancel(t, m, sdk.TypeCDN)
	t.Cleanup(end)
	cfg := commontest.LoadTestingConf(t, sdk.TypeCDN)

	cdntest.ClearItem(t, context.Background(), m, db)
	cdntest.ClearUnits(t, context.Background(), m, db)

	tmpDir, err := os.MkdirTemp("", t.Name()+"-cdn-*")
	require.NoError(t, err)

	// Create cdn service
	s := Service{
		DBConnectionFactory: factory,
		Cache:               cache,
		Mapper:              m,
		Common: service.Common{
			GoRoutines: sdk.NewGoRoutines(context.TODO()),
		},
	}
	cdnUnits, err := storage.Init(context.Background(), m, cache, db.DbMap, sdk.NewGoRoutines(context.TODO()), storage.Configuration{
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
			"test-local.TestSyncBuffer": {
				Local: &storage.LocalStorageConfiguration{
					Path: tmpDir,
				},
			},
		},
	})
	require.NoError(t, err)
	s.Units = cdnUnits

	_ = cache.Set("cdn:buffer:my-item", "foo")

	s.Units.SyncBuffer(context.Background())

	b, err := cache.Exist("cdn:buffer:my-item")
	require.NoError(t, err)
	require.False(t, b)
}
