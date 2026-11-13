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

	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	t.Cleanup(cancel)

	tmpDir, err := os.MkdirTemp("", t.Name()+"-cdn-1-*")
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
		},
	})
	require.NoError(t, err)

	localUnit, ok := cdnUnits.Storages[0].(storage.StorageUnitWithLocator)
	require.True(t, ok)
	dest := &recordingStorageUnit{StorageUnitWithLocator: localUnit}
	cdnUnits.Storages[0] = dest

	return ctx, m, db, store, cdnUnits, dest
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
