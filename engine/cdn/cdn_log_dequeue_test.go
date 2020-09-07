package cdn

import (
	"context"
	"strconv"
	"testing"

	"github.com/mitchellh/hashstructure"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/cdn/index"
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
	index.InitDBMapping(m)
	storage.InitDBMapping(m)

	log.SetLogger(t)
	db, factory, cache, cancel := test.SetupPGToCancel(t, m, sdk.TypeCDN)
	t.Cleanup(cancel)

	cfg := test.LoadTestingConf(t, sdk.TypeCDN)

	cdntest.ClearIndex(t, context.TODO(), m, db)

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

	hm := handledMessage{
		Msg: hook.Message{
			Full: "coucou",
		},
		Status: "Building",
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

	content := buildMessage(hm)
	err = s.storeLogs(context.TODO(), index.TypeItemStepLog, hm.Signature, hm.Status, content, hm.Line)
	require.NoError(t, err)

	apiRef := index.ApiRef{
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
	item, err := index.LoadItemByApiRefHashAndType(context.TODO(), s.Mapper, db, strconv.FormatUint(hashRef, 10), index.TypeItemStepLog)
	require.NoError(t, err)
	require.NotNil(t, item)
	defer func() {
		_ = index.DeleteItem(m, db, item)
	}()
	require.Equal(t, index.StatusItemIncoming, item.Status)

	iu, err := storage.LoadItemUnitByUnit(context.TODO(), s.Mapper, db, s.Units.Buffer.ID(), item.ID, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)

	logs, err := s.Units.Buffer.Get(*iu, 0, 1)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	require.Equal(t, "[EMERGENCY] coucou\n", logs[0])
}

func TestStoreLastStepLog(t *testing.T) {
	m := gorpmapper.New()
	index.InitDBMapping(m)
	storage.InitDBMapping(m)

	log.SetLogger(t)
	db, factory, cache, cancel := test.SetupPGToCancel(t, m, sdk.TypeCDN)
	t.Cleanup(cancel)

	cfg := test.LoadTestingConf(t, sdk.TypeCDN)

	cdntest.ClearIndex(t, context.TODO(), m, db)

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

	hm := handledMessage{
		Msg:    hook.Message{},
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
	apiRef := index.ApiRef{
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

	item := index.Item{
		Status:     index.StatusItemIncoming,
		ApiRefHash: strconv.FormatUint(hashRef, 10),
		ApiRef:     apiRef,
		Type:       index.TypeItemStepLog,
	}
	require.NoError(t, index.InsertItem(context.TODO(), m, db, &item))
	defer func() {
		_ = index.DeleteItem(m, db, &item)
	}()
	content := buildMessage(hm)
	err = s.storeLogs(context.TODO(), index.TypeItemStepLog, hm.Signature, hm.Status, content, hm.Line)
	require.NoError(t, err)

	itemDB, err := index.LoadItemByID(context.TODO(), s.Mapper, db, item.ID)
	require.NoError(t, err)
	require.NotNil(t, itemDB)
	require.Equal(t, index.StatusItemCompleted, itemDB.Status)
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
	index.InitDBMapping(m)
	storage.InitDBMapping(m)

	log.SetLogger(t)
	db, factory, cache, cancel := test.SetupPGToCancel(t, m, sdk.TypeCDN)
	t.Cleanup(cancel)

	cfg := test.LoadTestingConf(t, sdk.TypeCDN)

	cdntest.ClearIndex(t, context.TODO(), m, db)

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

	hm := handledMessage{
		Msg: hook.Message{
			Full: "voici un message",
		},
		Status: sdk.StatusSuccess,
		Line:   2,
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
	apiRef := index.ApiRef{
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

	item := index.Item{
		Status:     index.StatusItemIncoming,
		ApiRefHash: strconv.FormatUint(hashRef, 10),
		ApiRef:     apiRef,
		Type:       index.TypeItemStepLog,
	}
	require.NoError(t, index.InsertItem(context.TODO(), m, db, &item))
	defer func() {
		_ = index.DeleteItem(m, db, &item)
	}()

	content := buildMessage(hm)
	err = s.storeLogs(context.TODO(), index.TypeItemStepLog, hm.Signature, hm.Status, content, hm.Line)
	require.NoError(t, err)

	itemDB, err := index.LoadItemByID(context.TODO(), s.Mapper, db, item.ID)
	require.NoError(t, err)
	require.NotNil(t, itemDB)
	require.Equal(t, index.StatusItemIncoming, itemDB.Status)
	require.NotEmpty(t, itemDB.Hash)

	unit, err := storage.LoadUnitByName(context.TODO(), m, db, s.Units.Buffer.Name())
	require.NoError(t, err)
	require.NotNil(t, unit)

	// Must exist
	iu, err := storage.LoadItemUnitByUnit(context.TODO(), m, db, unit.ID, itemDB.ID)
	require.NoError(t, err)
	require.NotNil(t, iu)

	// Received Missing log
	hm.Line = 1
	hm.Status = ""
	content = buildMessage(hm)

	err = s.storeLogs(context.TODO(), index.TypeItemStepLog, hm.Signature, hm.Status, content, hm.Line)
	require.NoError(t, err)

	itemDB2, err := index.LoadItemByID(context.TODO(), s.Mapper, db, item.ID)
	require.NoError(t, err)
	require.NotNil(t, itemDB2)
	require.Equal(t, index.StatusItemCompleted, itemDB2.Status)
	require.NotEmpty(t, itemDB2.Hash)

	iu, err = storage.LoadItemUnitByUnit(context.TODO(), m, db, unit.ID, itemDB2.ID)
	require.NoError(t, err)
	require.NotNil(t, iu)

	lines, err := s.Units.Buffer.Get(*iu, 0, 2)
	require.NoError(t, err)
	require.Len(t, lines, 2)
	require.Equal(t, "[EMERGENCY] voici un message\n", lines[0])
	require.Equal(t, "[EMERGENCY] voici un message\n", lines[1])
}

func TestStoreNewServiceLogAndAppend(t *testing.T) {
	m := gorpmapper.New()
	index.InitDBMapping(m)
	storage.InitDBMapping(m)

	log.SetLogger(t)
	db, factory, cache, cancel := test.SetupPGToCancel(t, m, sdk.TypeCDN)
	t.Cleanup(cancel)

	cfg := test.LoadTestingConf(t, sdk.TypeCDN)

	cdntest.ClearIndex(t, context.TODO(), m, db)

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

	hm := handledMessage{
		Msg: hook.Message{
			Full: "log1",
		},
		Status: "Building",
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
		},
	}

	err = s.storeLogs(context.TODO(), index.TypeItemServiceLog, hm.Signature, hm.Status, hm.Msg.Full, 0)
	require.NoError(t, err)

	apiRef := index.ApiRef{
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
	item, err := index.LoadItemByApiRefHashAndType(context.TODO(), s.Mapper, db, strconv.FormatUint(hashRef, 10), index.TypeItemServiceLog)
	require.NoError(t, err)
	require.NotNil(t, item)
	defer func() {
		_ = index.DeleteItem(m, db, item)
	}()
	require.Equal(t, index.StatusItemIncoming, item.Status)

	var logs []string
	err = cache.ScoredSetScan(context.Background(), item.ID, 0, 1, &logs)
	require.NoError(t, err)

	hm2 := handledMessage{
		Msg: hook.Message{
			Full: "log2",
		},
		Status: "Building",
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
	err = s.storeLogs(context.TODO(), index.TypeItemServiceLog, hm.Signature, hm.Status, hm2.Msg.Full, 0)
	require.NoError(t, err)

	err = cache.ScoredSetScan(context.TODO(), item.ID, 0, 2, &logs)
	require.NoError(t, err)
	require.Len(t, logs, 2)
	require.Equal(t, "log2", logs[1])
}
