package cdn

import (
	"context"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/ovh/cds/engine/cdn/redis"

	"github.com/mitchellh/hashstructure"
	"github.com/stretchr/testify/require"

	cacheP "github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	_ "github.com/ovh/cds/engine/cdn/storage/local"
	_ "github.com/ovh/cds/engine/cdn/storage/redis"
	cdntest "github.com/ovh/cds/engine/cdn/test"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/log/hook"
)

func TestStoreNewStepLog(t *testing.T) {
	m := gorpmapper.New()
	item.InitDBMapping(m)
	storage.InitDBMapping(m)

	log.SetLogger(t)
	db, factory, cache, cancel := test.SetupPGToCancel(t, m, sdk.TypeCDN)
	t.Cleanup(cancel)

	cdntest.ClearItem(t, context.TODO(), m, db)

	// Create cdn service
	s := Service{
		DBConnectionFactory: factory,
		Cache:               cache,
		Mapper:              m,
	}

	cdnUnits := newRunningStorageUnits(t, m, db.DbMap)
	s.Units = cdnUnits

	hm := handledMessage{
		Msg: hook.Message{
			Full: "coucou",
		},
		Status: "Building",
		Line:   0,
		Signature: log.Signature{
			ProjectKey:   sdk.RandomString(10),
			WorkflowID:   1,
			WorkflowName: "MyWorklow",
			RunID:        1,
			NodeRunID:    1,
			NodeRunName:  "MyPipeline",
			JobName:      "MyJob",
			JobID:        1,
			Worker: &log.SignatureWorker{
				StepName:  "script1",
				StepOrder: 1,
			},
		},
	}

	content := buildMessage(hm)
	err := s.storeLogs(context.TODO(), sdk.CDNTypeItemStepLog, hm.Signature, hm.Status, content, hm.Line)
	require.NoError(t, err)

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
		_ = item.DeleteByIDs(db, []string{it.ID})
	}()
	require.Equal(t, sdk.CDNStatusItemIncoming, it.Status)

	iu, err := storage.LoadItemUnitByUnit(context.TODO(), s.Mapper, db, s.Units.Buffer.ID(), it.ID, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)

	bufferReader, err := s.Units.Buffer.NewReader(context.TODO(), *iu)
	require.NoError(t, err)

	buf := new(strings.Builder)
	_, err = io.Copy(buf, bufferReader)
	require.NoError(t, err)

	require.NoError(t, err)
	require.Equal(t, "[EMERGENCY] coucou\n", buf.String())
}

func TestStoreLastStepLog(t *testing.T) {
	m := gorpmapper.New()
	item.InitDBMapping(m)
	storage.InitDBMapping(m)

	log.SetLogger(t)
	db, factory, cache, cancel := test.SetupPGToCancel(t, m, sdk.TypeCDN)
	t.Cleanup(cancel)

	cdntest.ClearItem(t, context.TODO(), m, db)

	// Create cdn service
	s := Service{
		DBConnectionFactory: factory,
		Cache:               cache,
		Mapper:              m,
	}

	cdnUnits := newRunningStorageUnits(t, m, db.DbMap)
	s.Units = cdnUnits

	hm := handledMessage{
		Msg:    hook.Message{},
		Status: sdk.StatusSuccess,
		Line:   0,
		Signature: log.Signature{
			ProjectKey:   sdk.RandomString(10),
			WorkflowID:   1,
			WorkflowName: "MyWorklow",
			RunID:        1,
			NodeRunID:    1,
			NodeRunName:  "MyPipeline",
			JobName:      "MyJob",
			JobID:        1,
			Worker: &log.SignatureWorker{
				StepName:  "script1",
				StepOrder: 1,
			},
		},
	}
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

	it := sdk.CDNItem{
		Status:     sdk.CDNStatusItemIncoming,
		APIRefHash: strconv.FormatUint(hashRef, 10),
		APIRef:     apiRef,
		Type:       sdk.CDNTypeItemStepLog,
	}
	require.NoError(t, item.Insert(context.TODO(), m, db, &it))
	defer func() {
		_ = item.DeleteByIDs(db, []string{it.ID})

	}()
	content := buildMessage(hm)
	err = s.storeLogs(context.TODO(), sdk.CDNTypeItemStepLog, hm.Signature, hm.Status, content, hm.Line)
	require.NoError(t, err)

	itemDB, err := item.LoadByID(context.TODO(), s.Mapper, db, it.ID)
	require.NoError(t, err)
	require.NotNil(t, itemDB)
	require.Equal(t, sdk.CDNStatusItemCompleted, itemDB.Status)
	require.NotEmpty(t, itemDB.Hash)
	require.NotEmpty(t, itemDB.MD5)
	require.NotZero(t, itemDB.Size)

	unit, err := storage.LoadUnitByName(context.TODO(), m, db, s.Units.Buffer.Name())
	require.NoError(t, err)
	require.NotNil(t, unit)

	itemUnit, err := storage.LoadItemUnitByUnit(context.TODO(), m, db, unit.ID, itemDB.ID)
	require.NoError(t, err)
	require.NotNil(t, itemUnit)
}

func TestStoreLogWrongOrder(t *testing.T) {
	m := gorpmapper.New()
	item.InitDBMapping(m)
	storage.InitDBMapping(m)

	log.SetLogger(t)
	db, factory, cache, cancel := test.SetupPGToCancel(t, m, sdk.TypeCDN)
	t.Cleanup(cancel)

	cdntest.ClearItem(t, context.TODO(), m, db)

	// Create cdn service
	s := Service{
		DBConnectionFactory: factory,
		Cache:               cache,
		Mapper:              m,
	}

	cdnUnits := newRunningStorageUnits(t, m, db.DbMap)

	s.Units = cdnUnits

	hm := handledMessage{
		Msg: hook.Message{
			Full: "voici un message",
		},
		Status: sdk.StatusSuccess,
		Line:   1,
		Signature: log.Signature{
			ProjectKey:   sdk.RandomString(10),
			WorkflowID:   1,
			WorkflowName: "MyWorklow",
			RunID:        1,
			NodeRunID:    1,
			NodeRunName:  "MyPipeline",
			JobName:      "MyJob",
			JobID:        1,
			Worker: &log.SignatureWorker{
				StepName:  "script1",
				StepOrder: 1,
			},
		},
	}
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

	it := sdk.CDNItem{
		Status:     sdk.CDNStatusItemIncoming,
		APIRefHash: strconv.FormatUint(hashRef, 10),
		APIRef:     apiRef,
		Type:       sdk.CDNTypeItemStepLog,
	}
	require.NoError(t, item.Insert(context.TODO(), m, db, &it))
	defer func() {
		_ = item.DeleteByIDs(db, []string{it.ID})
	}()

	content := buildMessage(hm)
	err = s.storeLogs(context.TODO(), sdk.CDNTypeItemStepLog, hm.Signature, hm.Status, content, hm.Line)
	require.NoError(t, err)

	itemDB, err := item.LoadByID(context.TODO(), s.Mapper, db, it.ID)
	require.NoError(t, err)
	require.NotNil(t, itemDB)
	require.Equal(t, sdk.CDNStatusItemIncoming, itemDB.Status)
	require.NotEmpty(t, itemDB.Hash)

	unit, err := storage.LoadUnitByName(context.TODO(), m, db, s.Units.Buffer.Name())
	require.NoError(t, err)
	require.NotNil(t, unit)

	// Must exist
	iu, err := storage.LoadItemUnitByUnit(context.TODO(), m, db, unit.ID, itemDB.ID)
	require.NoError(t, err)
	require.NotNil(t, iu)

	// Received Missing log
	hm.Line = 0
	hm.Status = ""
	content = buildMessage(hm)

	err = s.storeLogs(context.TODO(), sdk.CDNTypeItemStepLog, hm.Signature, hm.Status, content, hm.Line)
	require.NoError(t, err)

	itemDB2, err := item.LoadByID(context.TODO(), s.Mapper, db, it.ID)
	require.NoError(t, err)
	require.NotNil(t, itemDB2)
	require.Equal(t, sdk.CDNStatusItemCompleted, itemDB2.Status)
	require.NotEmpty(t, itemDB2.Hash)

	iu, err = storage.LoadItemUnitByUnit(context.TODO(), m, db, unit.ID, itemDB2.ID)
	require.NoError(t, err)
	require.NotNil(t, iu)

	bufferReader, err := s.Units.Buffer.NewReader(context.TODO(), *iu)
	require.NoError(t, err)
	bufferReader.(*redis.Reader).From = 0
	bufferReader.(*redis.Reader).Size = 2

	buf := new(strings.Builder)
	_, err = io.Copy(buf, bufferReader)
	require.NoError(t, err)

	require.Equal(t, "[EMERGENCY] voici un message\n[EMERGENCY] voici un message\n", buf.String())

}

func TestStoreNewServiceLogAndAppend(t *testing.T) {
	m := gorpmapper.New()
	item.InitDBMapping(m)
	storage.InitDBMapping(m)

	log.SetLogger(t)
	db, factory, cache, cancel := test.SetupPGToCancel(t, m, sdk.TypeCDN)
	t.Cleanup(cancel)

	cdntest.ClearItem(t, context.TODO(), m, db)

	// Create cdn service
	s := Service{
		DBConnectionFactory: factory,
		Cache:               cache,
		Mapper:              m,
	}

	cdnUnits := newRunningStorageUnits(t, m, db.DbMap)
	s.Units = cdnUnits

	hm := handledMessage{
		Msg: hook.Message{
			Full: "log1",
		},
		Status: "Building",
		Line:   0,
		Signature: log.Signature{
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

	err := s.storeLogs(context.TODO(), sdk.CDNTypeItemServiceLog, hm.Signature, hm.Status, hm.Msg.Full, 0)
	require.NoError(t, err)

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
	t.Cleanup(func() { _ = item.DeleteByIDs(db, []string{it.ID}) })
	require.Equal(t, sdk.CDNStatusItemIncoming, it.Status)

	var logs []string
	require.NoError(t, cache.ScoredSetScan(context.Background(), cacheP.Key("cdn", "buffer", it.ID), 0, 1, &logs))
	require.Len(t, logs, 1)
	require.Equal(t, "log1", logs[0])

	hm2 := handledMessage{
		Msg: hook.Message{
			Full: "log2",
		},
		Status:    "Building",
		Signature: hm.Signature,
	}
	require.NoError(t, s.storeLogs(context.TODO(), sdk.CDNTypeItemServiceLog, hm2.Signature, hm2.Status, hm2.Msg.Full, 0))

	require.NoError(t, cache.ScoredSetScan(context.TODO(), cacheP.Key("cdn", "buffer", it.ID), 0, 2, &logs))
	require.Len(t, logs, 2)
	require.Equal(t, "log1", logs[0])
	require.Equal(t, "log2", logs[1])
}
