package cdn

import (
	"context"
	"strconv"
	"testing"

	"github.com/mitchellh/hashstructure"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/cdn/storage"
	_ "github.com/ovh/cds/engine/cdn/storage/local"
	_ "github.com/ovh/cds/engine/cdn/storage/redis"
	"github.com/ovh/cds/engine/gorpmapper"
	commontest "github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/log/hook"
)

func TestStoreNewStepLog(t *testing.T) {
	m := gorpmapper.New()
	index.InitDBMapping(m)
	storage.InitDBMapping(m)
	db, cache := test.SetupPGWithMapper(t, m, sdk.TypeCDN)
	cfg := commontest.LoadTestingConf(t, sdk.TypeCDN)

	// Create cdn service
	s := Service{
		DBConnectionFactory: test.DBConnectionFactory,
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

	content := buildMessage(hm.Signature, hm.Msg)
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
	require.Equal(t, index.StatusItemIncoming, item.Status)

	var logs []string
	err = cache.ScoredSetScan(context.Background(), item.ID, 0, 1, &logs)
	require.NoError(t, err)
	require.Equal(t, "[EMERGENCY] coucou\n", logs[0])
}

func TestStoreLastStepLog(t *testing.T) {
	m := gorpmapper.New()
	index.InitDBMapping(m)
	storage.InitDBMapping(m)
	db, cache := test.SetupPGWithMapper(t, m, sdk.TypeCDN)
	cfg := commontest.LoadTestingConf(t, sdk.TypeCDN)

	// Create cdn service
	s := Service{
		DBConnectionFactory: test.DBConnectionFactory,
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

	content := buildMessage(hm.Signature, hm.Msg)
	err = s.storeLogs(context.TODO(), index.TypeItemStepLog, hm.Signature, hm.Status, content, hm.Line)
	require.NoError(t, err)

	itemDB, err := index.LoadItemByID(context.TODO(), s.Mapper, db, item.ID)
	require.NoError(t, err)
	require.NotNil(t, itemDB)
	require.Equal(t, index.StatusItemCompleted, itemDB.Status)
	require.NotEmpty(t, itemDB.Hash)

	unit, err := storage.LoadUnitByName(context.TODO(), m, db, s.Units.Buffer.Name())
	require.NoError(t, err)
	require.NotNil(t, unit)

	itemUnit, err := storage.LoadItemByUnit(context.TODO(), m, db, unit.ID, itemDB.ID)
	require.NoError(t, err)
	require.NotNil(t, itemUnit)
}

func TestStoreNewServiceLogAndAppend(t *testing.T) {
	m := gorpmapper.New()
	index.InitDBMapping(m)
	storage.InitDBMapping(m)
	db, cache := test.SetupPGWithMapper(t, m, sdk.TypeCDN)
	cfg := commontest.LoadTestingConf(t, sdk.TypeCDN)

	// Create cdn service
	s := Service{
		DBConnectionFactory: test.DBConnectionFactory,
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
