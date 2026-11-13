package storage_test

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ovh/symmecrypt/ciphers/aesgcm"
	"github.com/ovh/symmecrypt/convergent"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	cdntest "github.com/ovh/cds/engine/cdn/test"
	"github.com/ovh/cds/engine/gorpmapper"
	commontest "github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
)

// recordingStorageUnit is a local storage unit that records the writers it opens, and can make
// the copy fail
type recordingStorageUnit struct {
	storage.StorageUnitWithLocator
	writers  []*recordingWriter
	writeErr error
}

func (s *recordingStorageUnit) NewWriter(_ context.Context, _ sdk.CDNItemUnit) (io.WriteCloser, error) {
	w := &recordingWriter{}
	s.writers = append(s.writers, w)
	return w, nil
}

func (s *recordingStorageUnit) Write(i sdk.CDNItemUnit, r io.Reader, w io.Writer) error {
	if s.writeErr != nil {
		return s.writeErr
	}
	return s.StorageUnitWithLocator.Write(i, r, w)
}

// recordingWriter can be aborted, like the swift writer
type recordingWriter struct {
	closed   bool
	abortErr error
}

func (w *recordingWriter) Write(p []byte) (int, error) { return len(p), nil }

func (w *recordingWriter) Close() error {
	w.closed = true
	return nil
}

func (w *recordingWriter) CloseWithError(err error) error {
	w.abortErr = err
	return nil
}

func initRecordingStorageUnit(t *testing.T) (context.Context, *gorpmapper.Mapper, *commontest.FakeTransaction, cache.Store, *storage.RunningStorageUnits, *recordingStorageUnit) {
	m := gorpmapper.New()
	item.InitDBMapping(m)
	storage.InitDBMapping(m)

	db, store := commontest.SetupPGWithMapper(t, m, sdk.TypeCDN)
	cfg := commontest.LoadTestingConf(t, sdk.TypeCDN)

	cdntest.ClearItem(t, context.TODO(), m, db)
	cdntest.ClearSyncRedisSet(t, store, "local_storage")
	cdntest.ClearSyncRedisSet(t, store, "local_storage_2")

	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	t.Cleanup(cancel)

	tmpDir, err := os.MkdirTemp("", t.Name()+"-cdn-1-*")
	require.NoError(t, err)
	tmpDir2, err := os.MkdirTemp("", t.Name()+"-cdn-2-*")
	require.NoError(t, err)

	cdnUnits, err := storage.Init(ctx, m, store, db.DbMap, sdk.NewGoRoutines(ctx), storage.Configuration{
		SyncSeconds:     10,
		SyncNbElements:  100,
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

	dest := &recordingStorageUnit{}
	replaceStorageUnit(t, cdnUnits, "local_storage", func(u storage.StorageUnitWithLocator) storage.StorageUnit {
		dest.StorageUnitWithLocator = u
		return dest
	})

	return ctx, m, db, store, cdnUnits, dest
}

// replaceStorageUnit swaps the storage unit with the given name for the one built by wrap
func replaceStorageUnit(t *testing.T, cdnUnits *storage.RunningStorageUnits, name string, wrap func(storage.StorageUnitWithLocator) storage.StorageUnit) {
	for x, s := range cdnUnits.Storages {
		if s.Name() == name {
			u, ok := s.(storage.StorageUnitWithLocator)
			require.True(t, ok)
			cdnUnits.Storages[x] = wrap(u)
			return
		}
	}
	require.Fail(t, "storage unit not found", name)
}

func insertCompletedLogItem(t *testing.T, ctx context.Context, m *gorpmapper.Mapper, db *commontest.FakeTransaction, hash string) *sdk.CDNItem {
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
		Status:     sdk.CDNStatusItemCompleted,
		Hash:       hash,
	}
	require.NoError(t, item.Insert(ctx, m, db, i))
	t.Cleanup(func() {
		_ = item.DeleteByID(db, i.ID)
	})
	// Insert replaces the encrypted hash by a placeholder, the locator of an item unit needs the clear one
	i.Hash = hash
	return i
}

// randomHash returns a valid item hash: the convergent locator of an item unit is derived from it
func randomHash(t *testing.T) string {
	h, err := convergent.NewHash(strings.NewReader(sdk.RandomString(20)))
	require.NoError(t, err)
	return h
}

func syncQueue(t *testing.T, ctx context.Context, store cache.Store, s storage.StorageUnit) []string {
	var ids []string
	require.NoError(t, store.ScoredSetRevRange(ctx, cache.Key(storage.KeyBackendSync, s.Name()), 0, 100, &ids))
	return ids
}

// An item that no unit holds can never be synced. The sync must neither open a writer on the
// destination (a swift writer starts an upload that leaks its goroutines when the copy is given
// up), nor keep the item in the sync queue (it would be retried forever, at each sync loop).
func TestProcessItemWithoutSourceUnit(t *testing.T) {
	ctx, m, db, store, cdnUnits, dest := initRecordingStorageUnit(t)

	i := insertCompletedLogItem(t, ctx, m, db, randomHash(t))
	cdnUnits.PushInSyncQueue(ctx, i.ID, i.Created)
	require.Contains(t, syncQueue(t, ctx, store, dest), i.ID)

	require.NoError(t, cdnUnits.ProcessItem(ctx, db.DbMap, dest, i.ID))

	require.Empty(t, dest.writers, "no writer must be opened on the destination when there is nothing to copy")
	require.NotContains(t, syncQueue(t, ctx, store, dest), i.ID, "an item that no unit holds must leave the sync queue")
}

// A source that is only missing in database while the buffer still has its lines can be a
// transient state: that item must stay in the sync queue, only the no source unit case leaves it.
func TestProcessItemWithSourceMissingInDatabase(t *testing.T) {
	ctx, m, db, store, cdnUnits, dest := initRecordingStorageUnit(t)

	i := insertCompletedLogItem(t, ctx, m, db, randomHash(t))
	bufferItemUnit := sdk.CDNItemUnit{ItemID: i.ID}
	require.NoError(t, cdnUnits.LogsBuffer().Add(bufferItemUnit, 1, 0, "this is the first log\n"))
	t.Cleanup(func() {
		_ = cdnUnits.LogsBuffer().Remove(context.TODO(), bufferItemUnit)
	})
	cdnUnits.PushInSyncQueue(ctx, i.ID, i.Created)

	err := cdnUnits.ProcessItem(ctx, db.DbMap, dest, i.ID)
	require.True(t, sdk.ErrorIs(err, sdk.ErrNotFound), "unexpected error: %v", err)

	require.Empty(t, dest.writers)
	require.Contains(t, syncQueue(t, ctx, store, dest), i.ID)
}

// When the copy fails, the destination writer must be aborted, not closed: closing a swift
// writer commits the partial object.
func TestProcessItemAbortsWriterOnCopyFailure(t *testing.T) {
	ctx, m, db, store, cdnUnits, dest := initRecordingStorageUnit(t)

	i := insertCompletedLogItem(t, ctx, m, db, "")
	itemUnit, err := cdnUnits.NewItemUnit(ctx, cdnUnits.LogsBuffer(), i)
	require.NoError(t, err)
	require.NoError(t, storage.InsertItemUnit(ctx, m, db, itemUnit))
	itemUnit, err = storage.LoadItemUnitByID(ctx, m, db, itemUnit.ID, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)
	require.NoError(t, cdnUnits.LogsBuffer().Add(*itemUnit, 1, 0, "this is the first log\n"))
	t.Cleanup(func() {
		_ = cdnUnits.LogsBuffer().Remove(context.TODO(), *itemUnit)
	})

	reader, err := cdnUnits.LogsBuffer().NewReader(context.TODO(), *itemUnit)
	require.NoError(t, err)
	h, err := convergent.NewHash(reader)
	require.NoError(t, err)
	i.Hash = h
	require.NoError(t, item.Update(ctx, m, db, i))
	cdnUnits.PushInSyncQueue(ctx, i.ID, i.Created)

	dest.writeErr = errors.New("copy failed")
	err = cdnUnits.ProcessItem(ctx, db.DbMap, dest, i.ID)
	require.Error(t, err)

	require.Len(t, dest.writers, 1)
	require.Equal(t, dest.writeErr, dest.writers[0].abortErr, "the writer must be aborted with the copy error")
	require.False(t, dest.writers[0].closed, "the writer must not be committed after a failed copy")

	_, err = storage.LoadItemUnitByUnit(ctx, m, db, dest.ID(), i.ID)
	require.True(t, sdk.ErrorIs(err, sdk.ErrNotFound), "a failed copy must not be recorded on the destination")
	require.Contains(t, syncQueue(t, ctx, store, dest), i.ID, "a failed copy must be retried")
}

// failingSourceUnit is a storage unit whose read fails after a first part of the content
type failingSourceUnit struct {
	storage.StorageUnitWithLocator
	readErr error
}

func (s *failingSourceUnit) NewReader(_ context.Context, _ sdk.CDNItemUnit) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

func (s *failingSourceUnit) Read(_ sdk.CDNItemUnit, _ io.Reader, w io.Writer) error {
	if _, err := w.Write([]byte("the first part of the content\n")); err != nil {
		return err
	}
	return s.readErr
}

// When the source fails in the middle of the copy, the destination must not see a complete
// copy: the writer would commit a truncated object.
func TestProcessItemAbortsWriterOnSourceFailure(t *testing.T) {
	ctx, m, db, store, cdnUnits, dest := initRecordingStorageUnit(t)

	src := &failingSourceUnit{readErr: errors.New("source read failed")}
	replaceStorageUnit(t, cdnUnits, "local_storage_2", func(u storage.StorageUnitWithLocator) storage.StorageUnit {
		src.StorageUnitWithLocator = u
		return src
	})

	// The item is only held by the source unit, so that the sync reads it from there
	i := insertCompletedLogItem(t, ctx, m, db, randomHash(t))
	srcItemUnit, err := cdnUnits.NewItemUnit(ctx, src, i)
	require.NoError(t, err)
	require.NoError(t, storage.InsertItemUnit(ctx, m, db, srcItemUnit))
	cdnUnits.PushInSyncQueue(ctx, i.ID, i.Created)

	err = cdnUnits.ProcessItem(ctx, db.DbMap, dest, i.ID)
	require.Error(t, err)

	require.Len(t, dest.writers, 1)
	require.False(t, dest.writers[0].closed, "the writer must not be committed after a failed read")
	require.Error(t, dest.writers[0].abortErr, "the writer must be aborted")

	_, err = storage.LoadItemUnitByUnit(ctx, m, db, dest.ID(), i.ID)
	require.True(t, sdk.ErrorIs(err, sdk.ErrNotFound), "a failed copy must not be recorded on the destination")
	require.Contains(t, syncQueue(t, ctx, store, dest), i.ID, "a failed copy must be retried")
}
