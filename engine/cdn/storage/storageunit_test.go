package storage_test

import (
	"bytes"
	"context"
	"io/ioutil"
	"testing"
	"time"

	"github.com/ovh/symmecrypt/ciphers/aesgcm"
	"github.com/ovh/symmecrypt/convergent"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/cdn/storage"
	_ "github.com/ovh/cds/engine/cdn/storage/local"
	_ "github.com/ovh/cds/engine/cdn/storage/redis"
	cdntest "github.com/ovh/cds/engine/cdn/test"
	"github.com/ovh/cds/engine/gorpmapper"
	commontest "github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func TestRun(t *testing.T) {
	m := gorpmapper.New()
	index.InitDBMapping(m)
	storage.InitDBMapping(m)

	db, _ := commontest.SetupPGWithMapper(t, m, sdk.TypeCDN)
	cfg := commontest.LoadTestingConf(t, sdk.TypeCDN)

	cdntest.ClearIndex(t, context.TODO(), m, db)

	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()

	tmpDir, err := ioutil.TempDir("", t.Name()+"-cdn-1-*")
	require.NoError(t, err)
	tmpDir2, err := ioutil.TempDir("", t.Name()+"-cdn-2-*")
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
				Name: "local_storage",
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
			}, {
				Name: "local_storage_2",
				Cron: "* * * * * ?",
				Local: &storage.LocalStorageConfiguration{
					Path: tmpDir2,
					Encryption: []convergent.ConvergentEncryptionConfig{
						{
							Cipher:      aesgcm.CipherName,
							LocatorSalt: "secret_locator_salt_2",
							SecretValue: "secret_value_2",
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, cdnUnits)
	require.NoError(t, cdnUnits.Start(ctx))

	units, err := storage.LoadAllUnits(ctx, m, db.DbMap)
	require.NoError(t, err)
	require.NotNil(t, units)
	require.NotEmpty(t, units)

	apiRef := sdk.CDNLogAPIRef{
		ProjectKey: sdk.RandomString(5),
	}

	apiRefHash, err := index.ComputeApiRef(apiRef)
	require.NoError(t, err)

	i := &index.Item{
		APIRef:     apiRef,
		APIRefHash: apiRefHash,
		Created:    time.Now(),
		Type:       sdk.CDNTypeItemStepLog,
		Status:     index.StatusItemIncoming,
	}
	require.NoError(t, index.InsertItem(ctx, m, db, i))
	defer func() {
		_ = index.DeleteItem(m, db, i)
	}()

	log.Debug("item ID: %v", i.ID)

	itemUnit, err := cdnUnits.NewItemUnit(ctx, cdnUnits.Buffer, i)
	require.NoError(t, err)

	err = storage.InsertItemUnit(ctx, m, db, itemUnit)
	require.NoError(t, err)

	itemUnit, err = storage.LoadItemUnitByID(ctx, m, db, itemUnit.ID, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)

	require.NoError(t, cdnUnits.Buffer.Add(*itemUnit, 1.0, "this is the first log\n"))
	require.NoError(t, cdnUnits.Buffer.Add(*itemUnit, 1.0, "this is the second log\n"))

	reader, err := cdnUnits.Buffer.NewReader(*itemUnit)
	require.NoError(t, err)

	h, err := convergent.NewHash(reader)
	require.NoError(t, err)
	i.Hash = h
	i.Status = index.StatusItemCompleted

	err = index.UpdateItem(ctx, m, db, i)
	require.NoError(t, err)

	i, err = index.LoadItemByID(ctx, m, db, i.ID, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)

	localUnit, err := storage.LoadUnitByName(ctx, m, db, "local_storage")
	require.NoError(t, err)

	localUnit2, err := storage.LoadUnitByName(ctx, m, db, "local_storage_2")
	require.NoError(t, err)

	localUnitDriver := cdnUnits.Storage(localUnit.Name)
	require.NotNil(t, localUnitDriver)

	localUnitDriver2 := cdnUnits.Storage(localUnit2.Name)
	require.NotNil(t, localUnitDriver)

	exists, err := localUnitDriver.ItemExists(*i)
	require.NoError(t, err)
	require.False(t, exists)

	<-ctx.Done()

	// Check that the first unit has been resync
	exists, err = localUnitDriver.ItemExists(*i)
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = localUnitDriver2.ItemExists(*i)
	require.NoError(t, err)
	require.True(t, exists)

	itemUnit, err = storage.LoadItemUnitByUnit(ctx, m, db, localUnitDriver.ID(), i.ID, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)

	reader, err = localUnitDriver.NewReader(*itemUnit)
	btes := new(bytes.Buffer)
	err = localUnitDriver.Read(*itemUnit, reader, btes)
	require.NoError(t, err)

	require.NoError(t, reader.Close())

	actual := btes.String()
	require.Equal(t, "this is the first log\nthis is the second log\n", actual, "item %s content should match", i.ID)

	itemIDs, err := storage.LoadAllItemIDUnknownByUnit(db, localUnitDriver.ID(), 100)
	require.NoError(t, err)
	require.Len(t, itemIDs, 0)

	// Check that the second unit has been resync
	itemUnit, err = storage.LoadItemUnitByUnit(ctx, m, db, localUnitDriver2.ID(), i.ID, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)

	reader, err = localUnitDriver2.NewReader(*itemUnit)
	btes = new(bytes.Buffer)
	err = localUnitDriver2.Read(*itemUnit, reader, btes)
	require.NoError(t, err)

	require.NoError(t, reader.Close())

	actual = btes.String()
	require.Equal(t, "this is the first log\nthis is the second log\n", actual, "item %s content should match", i.ID)

	itemIDs, err = storage.LoadAllItemIDUnknownByUnit(db, localUnitDriver2.ID(), 100)
	require.NoError(t, err)
	require.Len(t, itemIDs, 0)
}
