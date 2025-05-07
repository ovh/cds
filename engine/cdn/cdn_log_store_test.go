package cdn

import (
	"context"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/mitchellh/hashstructure"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	_ "github.com/ovh/cds/engine/cdn/storage/local"
	_ "github.com/ovh/cds/engine/cdn/storage/redis"
	cdntest "github.com/ovh/cds/engine/cdn/test"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdn"
	"github.com/ovh/cds/sdk/log/hook/graylog"
)

func TestStoreNewStepLog(t *testing.T) {
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
	cdnUnits := newRunningStorageUnits(t, m, db.DbMap, ctx, cache)
	s.Units = cdnUnits

	hm := handledMessage{
		Msg: graylog.Message{
			Full: "this is a message",
		},
		Signature: cdn.Signature{
			ProjectKey:   sdk.RandomString(10),
			WorkflowID:   1,
			WorkflowName: "MyWorklow",
			RunID:        1,
			NodeRunID:    1,
			NodeRunName:  "MyPipeline",
			JobName:      "MyJob",
			JobID:        1,
			Worker: &cdn.SignatureWorker{
				StepName:  "script1",
				StepOrder: 1,
			},
		},
	}

	content := buildMessage(hm)
	require.NoError(t, s.storeLogs(context.TODO(), sdk.CDNTypeItemStepLog, hm.Signature, false, content))

	apiRef := sdk.CDNLogAPIRef{
		ProjectKey:     hm.Signature.ProjectKey,
		WorkflowName:   hm.Signature.WorkflowName,
		WorkflowID:     hm.Signature.WorkflowID,
		RunID:          hm.Signature.RunID,
		NodeRunName:    hm.Signature.NodeRunName,
		NodeRunID:      hm.Signature.NodeRunID,
		NodeRunJobName: hm.Signature.JobName,
		NodeRunJobID:   hm.Signature.JobID,
		StepName:       hm.Signature.Worker.StepName,
		StepOrder:      hm.Signature.Worker.StepOrder,
	}
	hashRef, err := hashstructure.Hash(apiRef, nil)
	require.NoError(t, err)
	it, err := item.LoadByAPIRefHashAndType(context.TODO(), s.Mapper, db, strconv.FormatUint(hashRef, 10), sdk.CDNTypeItemStepLog)
	require.NoError(t, err)
	require.NotNil(t, it)
	defer func() {
		_ = item.DeleteByID(db, it.ID)
	}()
	require.Equal(t, sdk.CDNStatusItemIncoming, it.Status)

	iu, err := storage.LoadItemUnitByUnit(context.TODO(), s.Mapper, db, s.Units.LogsBuffer().ID(), it.ID, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)

	bufferReader, err := s.Units.LogsBuffer().NewReader(context.TODO(), *iu)
	require.NoError(t, err)

	buf := new(strings.Builder)
	_, err = io.Copy(buf, bufferReader)
	require.NoError(t, err)

	require.NoError(t, err)
	require.Equal(t, "this is a message\n", buf.String())
}

func TestStoreLastStepLog(t *testing.T) {
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
	cdnUnits := newRunningStorageUnits(t, m, db.DbMap, ctx, cache)
	s.Units = cdnUnits

	hm := handledMessage{
		Msg: graylog.Message{
			Full: "message",
		},
		IsTerminated: sdk.StatusTerminated,
		Signature: cdn.Signature{
			ProjectKey:   sdk.RandomString(10),
			WorkflowID:   1,
			WorkflowName: "MyWorklow",
			RunID:        1,
			NodeRunID:    1,
			NodeRunName:  "MyPipeline",
			JobName:      "MyJob",
			JobID:        1,
			Worker: &cdn.SignatureWorker{
				StepName:  "script1",
				StepOrder: 1,
			},
		},
	}
	apiRef := &sdk.CDNLogAPIRef{
		ProjectKey:     hm.Signature.ProjectKey,
		WorkflowName:   hm.Signature.WorkflowName,
		WorkflowID:     hm.Signature.WorkflowID,
		RunID:          hm.Signature.RunID,
		NodeRunName:    hm.Signature.NodeRunName,
		NodeRunID:      hm.Signature.NodeRunID,
		NodeRunJobName: hm.Signature.JobName,
		NodeRunJobID:   hm.Signature.JobID,
		StepName:       hm.Signature.Worker.StepName,
		StepOrder:      hm.Signature.Worker.StepOrder,
	}
	hashRef, err := hashstructure.Hash(apiRef, nil)
	require.NoError(t, err)

	it := sdk.CDNItem{
		Status:     sdk.CDNStatusItemIncoming,
		APIRefHash: strconv.FormatUint(hashRef, 10),
		APIRef:     apiRef,
		Type:       sdk.CDNTypeItemStepLog,
	}
	require.NoError(t, item.Insert(context.TODO(), m, db, &it))
	defer func() {
		_ = item.DeleteByID(db, it.ID)

	}()
	content := buildMessage(hm)
	err = s.storeLogs(context.TODO(), sdk.CDNTypeItemStepLog, hm.Signature, hm.IsTerminated, content)
	require.NoError(t, err)

	itemDB, err := item.LoadByID(context.TODO(), s.Mapper, db, it.ID)
	require.NoError(t, err)
	require.NotNil(t, itemDB)
	require.Equal(t, sdk.CDNStatusItemCompleted, itemDB.Status)
	require.NotEmpty(t, itemDB.Hash)
	require.NotEmpty(t, itemDB.MD5)
	require.NotZero(t, itemDB.Size)

	unit, err := storage.LoadUnitByName(context.TODO(), m, db, s.Units.LogsBuffer().Name())
	require.NoError(t, err)
	require.NotNil(t, unit)

	itemUnit, err := storage.LoadItemUnitByUnit(context.TODO(), m, db, unit.ID, itemDB.ID)
	require.NoError(t, err)
	require.NotNil(t, itemUnit)
}

func TestStoreNewServiceLog(t *testing.T) {
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
	cdnUnits := newRunningStorageUnits(t, m, db.DbMap, ctx, cache)
	s.Units = cdnUnits

	hm := handledMessage{
		Msg: graylog.Message{
			Full: "this is a message",
		},
		Signature: cdn.Signature{
			ProjectKey:   sdk.RandomString(10),
			WorkflowID:   1,
			WorkflowName: "MyWorklow",
			RunID:        1,
			NodeRunID:    1,
			NodeRunName:  "MyPipeline",
			JobName:      "MyJob",
			JobID:        1,
		},
	}

	content := buildMessage(hm)
	require.NoError(t, s.storeLogs(context.TODO(), sdk.CDNTypeItemServiceLog, hm.Signature, hm.IsTerminated, content))

	apiRef := sdk.CDNLogAPIRef{
		ProjectKey:     hm.Signature.ProjectKey,
		WorkflowName:   hm.Signature.WorkflowName,
		WorkflowID:     hm.Signature.WorkflowID,
		RunID:          hm.Signature.RunID,
		NodeRunName:    hm.Signature.NodeRunName,
		NodeRunID:      hm.Signature.NodeRunID,
		NodeRunJobName: hm.Signature.JobName,
		NodeRunJobID:   hm.Signature.JobID,
	}
	hashRef, err := hashstructure.Hash(apiRef, nil)
	require.NoError(t, err)
	it, err := item.LoadByAPIRefHashAndType(context.TODO(), s.Mapper, db, strconv.FormatUint(hashRef, 10), sdk.CDNTypeItemServiceLog)
	require.NoError(t, err)
	require.NotNil(t, it)
	defer func() {
		_ = item.DeleteByID(db, it.ID)
	}()
	require.Equal(t, sdk.CDNStatusItemIncoming, it.Status)

	iu, err := storage.LoadItemUnitByUnit(context.TODO(), s.Mapper, db, s.Units.LogsBuffer().ID(), it.ID, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)

	bufferReader, err := s.Units.LogsBuffer().NewReader(context.TODO(), *iu)
	require.NoError(t, err)

	buf := new(strings.Builder)
	_, err = io.Copy(buf, bufferReader)
	require.NoError(t, err)

	require.NoError(t, err)
	require.Equal(t, "this is a message\n", buf.String())
}
