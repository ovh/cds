package cdn

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	cdntest "github.com/ovh/cds/engine/cdn/test"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func newServiceLogV2APIRef(runJobID string, attempt int64) *sdk.CDNLogAPIRefV2 {
	return &sdk.CDNLogAPIRefV2{
		ProjectKey:   "PROJ",
		WorkflowName: "wkf",
		RunID:        sdk.UUID(),
		RunJobID:     runJobID,
		RunAttempt:   attempt,
		ItemType:     sdk.CDNTypeItemServiceLogV2,
		ServiceName:  "mypostgres",
	}
}

func TestPostDuplicateItemForJobHandlerIncomingSource(t *testing.T) {
	s, db := newTestService(t)
	cdntest.ClearItem(t, context.TODO(), s.Mapper, db)

	ctx, cancel := context.WithCancel(context.TODO())
	t.Cleanup(cancel)
	s.Units = newRunningStorageUnits(t, s.Mapper, db.DbMap, ctx, s.Cache)

	fromJob, toJob := sdk.UUID(), sdk.UUID()
	apiRef := newServiceLogV2APIRef(fromJob, 1)
	apiRefHash, err := apiRef.ToHash()
	require.NoError(t, err)

	// A service log still being received: its lines are in the buffer, and a storage copy
	// already exists as if the sync had run in between
	src := sdk.CDNItem{
		ID:         sdk.UUID(),
		Type:       sdk.CDNTypeItemServiceLogV2,
		Status:     sdk.CDNStatusItemIncoming,
		APIRef:     apiRef,
		APIRefHash: apiRefHash,
	}
	require.NoError(t, item.Insert(context.TODO(), s.Mapper, db, &src))
	t.Cleanup(func() { _ = s.Units.LogsBuffer().Remove(context.TODO(), sdk.CDNItemUnit{ItemID: src.ID}) })

	bufferIU := sdk.CDNItemUnit{ItemID: src.ID, UnitID: s.Units.LogsBuffer().ID(), Type: src.Type, Item: &src}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, &bufferIU))
	firstLine, secondLine := "this is the first log\n", "this is the second log\n"
	require.NoError(t, s.Units.LogsBuffer().Add(bufferIU, 0, 0, firstLine))
	require.NoError(t, s.Units.LogsBuffer().Add(bufferIU, 1, 0, secondLine))

	storageLocator := sdk.RandomString(64)
	storageIU := sdk.CDNItemUnit{ItemID: src.ID, UnitID: s.Units.Storages[0].ID(), Type: src.Type, Locator: storageLocator, Item: &src}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, &storageIU))

	uri := s.Router.GetRoute("POST", s.postDuplicateItemForJobHandler, nil)
	require.NotEmpty(t, uri)
	req := newRequest(t, "POST", uri, sdk.CDNDuplicateItemRequest{FromJob: fromJob, ToJob: toJob})
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 204, rec.Code)

	// The copy is attached to the new attempt and completed from the copied lines
	copyAPIRef := newServiceLogV2APIRef(toJob, 2)
	copyAPIRef.RunID = apiRef.RunID
	copyAPIRefHash, err := copyAPIRef.ToHash()
	require.NoError(t, err)
	copyItem, err := item.LoadByAPIRefHashAndType(context.TODO(), s.Mapper, db, copyAPIRefHash, sdk.CDNTypeItemServiceLogV2, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Units.LogsBuffer().Remove(context.TODO(), sdk.CDNItemUnit{ItemID: copyItem.ID}) })

	require.Equal(t, sdk.CDNStatusItemCompleted, copyItem.Status)
	require.False(t, copyItem.ToDelete)
	require.Equal(t, int64(len(firstLine)+len(secondLine)), copyItem.Size)
	require.NotEmpty(t, copyItem.Hash)
	require.Equal(t, src.Created.Unix(), copyItem.Created.Unix())

	nbLines, err := s.Units.LogsBuffer().Card(sdk.CDNItemUnit{ItemID: copyItem.ID})
	require.NoError(t, err)
	require.Equal(t, 2, nbLines)

	// Only the buffer copy was duplicated: the storage unit item of the source was not carried over
	copyUnits, err := storage.LoadAllItemUnitsByItemIDs(context.TODO(), s.Mapper, db, copyItem.ID, gorpmapper.GetAllOptions.WithDecryption)
	require.NoError(t, err)
	var hasBuffer bool
	for _, iu := range copyUnits {
		require.NotEqual(t, storageLocator, iu.Locator, "storage unit item must not be copied from an incoming source")
		if iu.UnitID == s.Units.LogsBuffer().ID() {
			hasBuffer = true
		}
	}
	require.True(t, hasBuffer)

	// The source keeps receiving its lines: it is left untouched
	srcDB, err := item.LoadByID(context.TODO(), s.Mapper, db, src.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.CDNStatusItemIncoming, srcDB.Status)
}

func TestPostDuplicateItemForJobHandlerCompletedSource(t *testing.T) {
	s, db := newTestService(t)
	cdntest.ClearItem(t, context.TODO(), s.Mapper, db)

	ctx, cancel := context.WithCancel(context.TODO())
	t.Cleanup(cancel)
	s.Units = newRunningStorageUnits(t, s.Mapper, db.DbMap, ctx, s.Cache)

	fromJob, toJob := sdk.UUID(), sdk.UUID()
	apiRef := newServiceLogV2APIRef(fromJob, 1)
	apiRefHash, err := apiRef.ToHash()
	require.NoError(t, err)

	// Insert resets encrypted fields on the struct, keep the clear hash aside for the comparison
	srcHash := sdk.RandomString(64)
	src := sdk.CDNItem{
		ID:         sdk.UUID(),
		Type:       sdk.CDNTypeItemServiceLogV2,
		Status:     sdk.CDNStatusItemCompleted,
		Hash:       srcHash,
		Size:       45,
		APIRef:     apiRef,
		APIRefHash: apiRefHash,
	}
	require.NoError(t, item.Insert(context.TODO(), s.Mapper, db, &src))
	t.Cleanup(func() { _ = s.Units.LogsBuffer().Remove(context.TODO(), sdk.CDNItemUnit{ItemID: src.ID}) })

	bufferIU := sdk.CDNItemUnit{ItemID: src.ID, UnitID: s.Units.LogsBuffer().ID(), Type: src.Type, Item: &src}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, &bufferIU))
	require.NoError(t, s.Units.LogsBuffer().Add(bufferIU, 0, 0, "this is the first log\n"))

	storageLocator := sdk.RandomString(64)
	storageIU := sdk.CDNItemUnit{ItemID: src.ID, UnitID: s.Units.Storages[0].ID(), Type: src.Type, Locator: storageLocator, Item: &src}
	require.NoError(t, storage.InsertItemUnit(context.TODO(), s.Mapper, db, &storageIU))

	uri := s.Router.GetRoute("POST", s.postDuplicateItemForJobHandler, nil)
	require.NotEmpty(t, uri)
	req := newRequest(t, "POST", uri, sdk.CDNDuplicateItemRequest{FromJob: fromJob, ToJob: toJob})
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 204, rec.Code)

	// A completed source is copied as is, storage copies included
	copyAPIRef := newServiceLogV2APIRef(toJob, 2)
	copyAPIRef.RunID = apiRef.RunID
	copyAPIRefHash, err := copyAPIRef.ToHash()
	require.NoError(t, err)
	copyItem, err := item.LoadByAPIRefHashAndType(context.TODO(), s.Mapper, db, copyAPIRefHash, sdk.CDNTypeItemServiceLogV2, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Units.LogsBuffer().Remove(context.TODO(), sdk.CDNItemUnit{ItemID: copyItem.ID}) })

	require.Equal(t, sdk.CDNStatusItemCompleted, copyItem.Status)
	require.Equal(t, srcHash, copyItem.Hash)
	require.Equal(t, src.Size, copyItem.Size)

	copyUnits, err := storage.LoadAllItemUnitsByItemIDs(context.TODO(), s.Mapper, db, copyItem.ID, gorpmapper.GetAllOptions.WithDecryption)
	require.NoError(t, err)
	var hasBuffer, hasStorage bool
	for _, iu := range copyUnits {
		switch iu.UnitID {
		case s.Units.LogsBuffer().ID():
			hasBuffer = true
		case s.Units.Storages[0].ID():
			hasStorage = true
			require.Equal(t, storageLocator, iu.Locator)
		}
	}
	require.True(t, hasBuffer)
	require.True(t, hasStorage)
}
