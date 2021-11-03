package storage_test

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha512"
	"encoding/hex"
	"io"
	"os"
	"testing"
	"time"

	"github.com/ovh/symmecrypt/ciphers/aesgcm"
	"github.com/ovh/symmecrypt/convergent"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	_ "github.com/ovh/cds/engine/cdn/storage/local"
	_ "github.com/ovh/cds/engine/cdn/storage/redis"
	_ "github.com/ovh/cds/engine/cdn/storage/swift"
	cdntest "github.com/ovh/cds/engine/cdn/test"
	"github.com/ovh/cds/engine/gorpmapper"
	commontest "github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
)

func TestDeduplicationCrossType(t *testing.T) {
	m := gorpmapper.New()
	item.InitDBMapping(m)
	storage.InitDBMapping(m)

	db, cache := commontest.SetupPGWithMapper(t, m, sdk.TypeCDN)
	cfg := commontest.LoadTestingConf(t, sdk.TypeCDN)

	cdntest.ClearItem(t, context.TODO(), m, db)
	cdntest.ClearSyncRedisSet(t, cache, "local_storage")

	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	t.Cleanup(cancel)

	bufferDir, err := os.MkdirTemp("", t.Name()+"-cdnbuffer-1-*")
	require.NoError(t, err)
	tmpDir, err := os.MkdirTemp("", t.Name()+"-cdn-1-*")
	require.NoError(t, err)

	cdnUnits, err := storage.Init(ctx, m, cache, db.DbMap, sdk.NewGoRoutines(ctx), storage.Configuration{
		SyncSeconds:     10,
		SyncNbElements:  100,
		HashLocatorSalt: "thisismysalt",
		Buffers: map[string]storage.BufferConfiguration{
			"redis_buffer": {
				Redis: &storage.RedisBufferConfiguration{
					Host:     cfg["redisHost"],
					Password: cfg["redisPassword"],
				},
				BufferType: storage.CDNBufferTypeLog,
			},
			"fs_buffer": {
				Local: &storage.LocalBufferConfiguration{
					Path: bufferDir,
				},
				BufferType: storage.CDNBufferTypeFile,
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
		},
	})
	require.NoError(t, err)
	require.NotNil(t, cdnUnits)
	cdnUnits.Start(ctx, sdk.NewGoRoutines(ctx))

	units, err := storage.LoadAllUnits(ctx, m, db.DbMap)
	require.NoError(t, err)
	require.NotNil(t, units)
	require.NotEmpty(t, units)

	// Create Item Empty Log
	apiRef := &sdk.CDNLogAPIRef{
		ProjectKey: sdk.RandomString(5),
	}
	apiRefHash, err := apiRef.ToHash()
	require.NoError(t, err)

	i := &sdk.CDNItem{
		APIRef:     apiRef,
		APIRefHash: apiRefHash,
		Created:    time.Now(),
		Type:       sdk.CDNTypeItemStepLog,
		Status:     sdk.CDNStatusItemIncoming,
	}
	require.NoError(t, item.Insert(ctx, m, db, i))
	defer func() {
		_ = item.DeleteByID(db, i.ID)
	}()

	itemUnit, err := cdnUnits.NewItemUnit(ctx, cdnUnits.LogsBuffer(), i)
	require.NoError(t, err)
	require.NoError(t, storage.InsertItemUnit(ctx, m, db, itemUnit))
	itemUnit, err = storage.LoadItemUnitByID(ctx, m, db, itemUnit.ID, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)
	reader, err := cdnUnits.LogsBuffer().NewReader(context.TODO(), *itemUnit)
	require.NoError(t, err)
	h, err := convergent.NewHash(reader)
	require.NoError(t, err)
	i.Hash = h
	i.Status = sdk.CDNStatusItemCompleted
	require.NoError(t, item.Update(ctx, m, db, i))
	require.NoError(t, err)
	i, err = item.LoadByID(ctx, m, db, i.ID, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)

	localUnit, err := storage.LoadUnitByName(ctx, m, db, "local_storage")
	require.NoError(t, err)

	localUnitDriver := cdnUnits.Storage(localUnit.Name)
	require.NotNil(t, localUnitDriver)

	// Sync item in backend
	require.NoError(t, cdnUnits.FillWithUnknownItems(ctx, cdnUnits.Storages[0], 100))
	require.NoError(t, cdnUnits.FillSyncItemChannel(ctx, cdnUnits.Storages[0], 100))
	time.Sleep(1 * time.Second)

	<-ctx.Done()

	// Check that the first unit has been resync
	exists, err := localUnitDriver.ItemExists(context.TODO(), m, db, *i)
	require.NoError(t, err)
	require.True(t, exists)

	logItemUnit, err := storage.LoadItemUnitByUnit(ctx, m, db, localUnitDriver.ID(), i.ID, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)

	// Add empty artifact
	// Create Item Empty Log
	apiRefArtifact := &sdk.CDNRunResultAPIRef{
		ProjectKey:   sdk.RandomString(5),
		ArtifactName: "myfile.txt",
		Perm:         0777,
	}
	apiRefHashArtifact, err := apiRefArtifact.ToHash()
	require.NoError(t, err)

	itemArtifact := &sdk.CDNItem{
		APIRef:     apiRefArtifact,
		APIRefHash: apiRefHashArtifact,
		Created:    time.Now(),
		Type:       sdk.CDNTypeItemRunResult,
		Status:     sdk.CDNStatusItemCompleted,
	}
	iuArtifact, err := cdnUnits.NewItemUnit(ctx, cdnUnits.FileBuffer(), itemArtifact)
	require.NoError(t, err)

	// Create Destination Writer
	writer, err := cdnUnits.FileBuffer().NewWriter(ctx, *iuArtifact)
	require.NoError(t, err)

	// Compute md5 and sha512
	artifactContent := make([]byte, 0)
	artifactReader := bytes.NewReader(artifactContent)
	md5Hash := md5.New()
	sha512Hash := sha512.New()
	pagesize := os.Getpagesize()
	mreader := bufio.NewReaderSize(artifactReader, pagesize)
	multiWriter := io.MultiWriter(md5Hash, sha512Hash)
	teeReader := io.TeeReader(mreader, multiWriter)

	require.NoError(t, cdnUnits.FileBuffer().Write(*iuArtifact, teeReader, writer))
	sha512S := hex.EncodeToString(sha512Hash.Sum(nil))
	md5S := hex.EncodeToString(md5Hash.Sum(nil))

	itemArtifact.Hash = sha512S
	itemArtifact.MD5 = md5S
	itemArtifact.Size = 0
	itemArtifact.Status = sdk.CDNStatusItemCompleted
	iuArtifact, err = cdnUnits.NewItemUnit(ctx, cdnUnits.FileBuffer(), itemArtifact)
	require.NoError(t, err)
	require.NoError(t, item.Insert(ctx, m, db, itemArtifact))
	defer func() {
		_ = item.DeleteByID(db, itemArtifact.ID)
	}()
	// Insert Item Unit
	iuArtifact.ItemID = iuArtifact.Item.ID
	require.NoError(t, storage.InsertItemUnit(ctx, m, db, iuArtifact))

	// Check if the content (based on the locator) is already known from the destination unit
	has, err := cdnUnits.GetItemUnitByLocatorByUnit(logItemUnit.Locator, cdnUnits.Storages[0].ID(), iuArtifact.Type)
	require.NoError(t, err)
	require.False(t, has)

	require.NoError(t, cdnUnits.FillWithUnknownItems(ctx, cdnUnits.Storages[0], 100))
	require.NoError(t, cdnUnits.FillSyncItemChannel(ctx, cdnUnits.Storages[0], 100))
	time.Sleep(2 * time.Second)

	<-ctx.Done()

	// Check that the first unit has been resync
	exists, err = localUnitDriver.ItemExists(context.TODO(), m, db, *itemArtifact)
	require.NoError(t, err)
	require.True(t, exists)

	artItemUnit, err := storage.LoadItemUnitByUnit(ctx, m, db, localUnitDriver.ID(), itemArtifact.ID, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)

	require.Equal(t, logItemUnit.HashLocator, artItemUnit.HashLocator)
	require.NotEqual(t, logItemUnit.Type, artItemUnit.Type)

}

func TestRun(t *testing.T) {
	m := gorpmapper.New()
	item.InitDBMapping(m)
	storage.InitDBMapping(m)

	db, cache := commontest.SetupPGWithMapper(t, m, sdk.TypeCDN)
	cfg := commontest.LoadTestingConf(t, sdk.TypeCDN)

	cdntest.ClearItem(t, context.TODO(), m, db)
	cdntest.ClearSyncRedisSet(t, cache, "local_storage")
	cdntest.ClearSyncRedisSet(t, cache, "local_storage_2")

	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	t.Cleanup(cancel)

	tmpDir, err := os.MkdirTemp("", t.Name()+"-cdn-1-*")
	require.NoError(t, err)
	tmpDir2, err := os.MkdirTemp("", t.Name()+"-cdn-2-*")
	require.NoError(t, err)

	cdnUnits, err := storage.Init(ctx, m, cache, db.DbMap, sdk.NewGoRoutines(ctx), storage.Configuration{
		SyncSeconds:     10,
		SyncNbElements:  100,
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
			"local_storage_2": {
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
	cdnUnits.Start(ctx, sdk.NewGoRoutines(ctx))

	units, err := storage.LoadAllUnits(ctx, m, db.DbMap)
	require.NoError(t, err)
	require.NotNil(t, units)
	require.NotEmpty(t, units)

	apiRef := &sdk.CDNLogAPIRef{
		ProjectKey: sdk.RandomString(5),
	}

	apiRefHash, err := apiRef.ToHash()
	require.NoError(t, err)

	i := &sdk.CDNItem{
		APIRef:     apiRef,
		APIRefHash: apiRefHash,
		Created:    time.Now(),
		Type:       sdk.CDNTypeItemStepLog,
		Status:     sdk.CDNStatusItemIncoming,
	}
	require.NoError(t, item.Insert(ctx, m, db, i))
	defer func() {
		_ = item.DeleteByID(db, i.ID)
	}()

	itemUnit, err := cdnUnits.NewItemUnit(ctx, cdnUnits.LogsBuffer(), i)
	require.NoError(t, err)

	err = storage.InsertItemUnit(ctx, m, db, itemUnit)
	require.NoError(t, err)

	itemUnit, err = storage.LoadItemUnitByID(ctx, m, db, itemUnit.ID, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)

	require.NoError(t, cdnUnits.LogsBuffer().Add(*itemUnit, 1, 0, "this is the first log\n"))

	require.NoError(t, cdnUnits.LogsBuffer().Add(*itemUnit, 2, 0, "this is the second log\n"))

	reader, err := cdnUnits.LogsBuffer().NewReader(context.TODO(), *itemUnit)
	require.NoError(t, err)

	h, err := convergent.NewHash(reader)
	require.NoError(t, err)
	i.Hash = h
	i.Status = sdk.CDNStatusItemCompleted

	err = item.Update(ctx, m, db, i)
	require.NoError(t, err)

	i, err = item.LoadByID(ctx, m, db, i.ID, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)

	localUnit, err := storage.LoadUnitByName(ctx, m, db, "local_storage")
	require.NoError(t, err)

	localUnit2, err := storage.LoadUnitByName(ctx, m, db, "local_storage_2")
	require.NoError(t, err)

	localUnitDriver := cdnUnits.Storage(localUnit.Name)
	require.NotNil(t, localUnitDriver)

	localUnitDriver2 := cdnUnits.Storage(localUnit2.Name)
	require.NotNil(t, localUnitDriver)

	exists, err := localUnitDriver.ItemExists(context.TODO(), m, db, *i)
	require.NoError(t, err)
	require.False(t, exists)

	require.NoError(t, cdnUnits.FillWithUnknownItems(ctx, cdnUnits.Storages[0], 100))
	require.NoError(t, cdnUnits.FillSyncItemChannel(ctx, cdnUnits.Storages[0], 100))
	time.Sleep(1 * time.Second)
	require.NoError(t, cdnUnits.FillWithUnknownItems(ctx, cdnUnits.Storages[1], 100))
	require.NoError(t, cdnUnits.FillSyncItemChannel(ctx, cdnUnits.Storages[1], 100))
	time.Sleep(1 * time.Second)

	<-ctx.Done()

	// Check that the first unit has been resync
	exists, err = localUnitDriver.ItemExists(context.TODO(), m, db, *i)
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = localUnitDriver2.ItemExists(context.TODO(), m, db, *i)
	require.NoError(t, err)
	require.True(t, exists)

	itemUnit, err = storage.LoadItemUnitByUnit(ctx, m, db, localUnitDriver.ID(), i.ID, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)

	reader, err = localUnitDriver.NewReader(context.TODO(), *itemUnit)
	btes := new(bytes.Buffer)
	err = localUnitDriver.Read(*itemUnit, reader, btes)
	require.NoError(t, err)

	require.NoError(t, reader.Close())

	actual := btes.String()
	require.Equal(t, "this is the first log\nthis is the second log\n", actual, "item %s content should match", i.ID)

	itemIDs, err := storage.LoadAllItemIDUnknownByUnit(db, localUnitDriver.ID(), 0, 100)
	require.NoError(t, err)
	require.Len(t, itemIDs, 0)

	// Check that the second unit has been resync
	itemUnit, err = storage.LoadItemUnitByUnit(ctx, m, db, localUnitDriver2.ID(), i.ID, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)

	reader, err = localUnitDriver2.NewReader(context.TODO(), *itemUnit)
	btes = new(bytes.Buffer)
	err = localUnitDriver2.Read(*itemUnit, reader, btes)
	require.NoError(t, err)

	require.NoError(t, reader.Close())

	actual = btes.String()
	require.Equal(t, "this is the first log\nthis is the second log\n", actual, "item %s content should match", i.ID)

	itemIDs, err = storage.LoadAllItemIDUnknownByUnit(db, localUnitDriver2.ID(), 0, 100)
	require.NoError(t, err)
	require.Len(t, itemIDs, 0)
}
