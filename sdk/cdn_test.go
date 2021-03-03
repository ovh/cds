package sdk

import (
	"encoding/json"
	"github.com/ovh/cds/sdk/cdn"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestCDNItemLogJSON(t *testing.T) {
	item := CDNItem{
		APIRef: NewCDNLogApiRef(cdn.Signature{
			Worker: &cdn.SignatureWorker{
				WorkerName: "workername",
				StepOrder:  1,
				StepName:   "stepName",
			},
			ProjectKey:   "KEY",
			WorkflowName: "NAME",
			JobName:      "JobName",
			JobID:        1,
			RunID:        1,
			WorkflowID:   1,
			NodeRunID:    1,
			NodeRunName:  "nodename",
		}),
		ID:   "AAA",
		Type: CDNTypeItemStepLog,
	}
	bts, err := json.Marshal(item)
	require.NoError(t, err)

	var itemU CDNItem
	require.NoError(t, json.Unmarshal(bts, &itemU))

	_, is := itemU.GetCDNLogApiRef()
	require.True(t, is)
	require.Equal(t, "project.KEY-workflow.NAME-pipeline.nodename-job.JobName-step.1.log", itemU.APIRef.ToFilename())
}

func TestCDNItemArtefactJSON(t *testing.T) {
	item := CDNItem{
		APIRef: NewCDNRunResultApiRef(cdn.Signature{
			Worker: &cdn.SignatureWorker{
				WorkerName: "workername",
				FileName:   "myartifact",
			},
			ProjectKey:   "KEY",
			WorkflowName: "NAME",
			JobName:      "JobName",
			JobID:        1,
			RunID:        1,
			WorkflowID:   1,
		}),
		ID:   "AAA",
		Type: CDNTypeItemRunResult,
	}
	bts, err := json.Marshal(item)
	require.NoError(t, err)

	var itemU CDNItem
	require.NoError(t, json.Unmarshal(bts, &itemU))

	_, is := itemU.GetCDNRunResultApiRef()
	require.True(t, is)
	require.Equal(t, "myartifact", itemU.APIRef.ToFilename())
}

func TestCDNItemWorkerCacheJSON(t *testing.T) {
	item := CDNItem{
		APIRef: NewCDNWorkerCacheApiRef(cdn.Signature{
			Worker: &cdn.SignatureWorker{
				WorkerName: "workername",
				CacheTag:   "mycache",
			},
			ProjectKey:   "KEY",
			WorkflowName: "NAME",
			JobName:      "JobName",
			JobID:        1,
			RunID:        1,
			WorkflowID:   1,
		}),
		ID:   "AAA",
		Type: CDNTypeItemWorkerCache,
	}
	bts, err := json.Marshal(item)
	require.NoError(t, err)

	var itemU CDNItem
	require.NoError(t, json.Unmarshal(bts, &itemU))

	workerCacheApiRef, is := itemU.GetCDNWorkerCacheApiRef()
	require.True(t, is)
	require.True(t, workerCacheApiRef.ExpireAt.After(time.Now()))
	require.Equal(t, "mycache", itemU.APIRef.ToFilename())
}
