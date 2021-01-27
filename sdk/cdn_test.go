package sdk

import (
	"encoding/json"
	"github.com/ovh/cds/sdk/cdn"
	"github.com/stretchr/testify/require"
	"testing"
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
		APIRef: NewCDNArtifactApiRef(cdn.Signature{
			Worker: &cdn.SignatureWorker{
				WorkerName:   "workername",
				ArtifactName: "myartifact",
			},
			ProjectKey:   "KEY",
			WorkflowName: "NAME",
			JobName:      "JobName",
			JobID:        1,
			RunID:        1,
			WorkflowID:   1,
		}),
		ID:   "AAA",
		Type: CDNTypeItemArtifact,
	}
	bts, err := json.Marshal(item)
	require.NoError(t, err)

	var itemU CDNItem
	require.NoError(t, json.Unmarshal(bts, &itemU))

	_, is := itemU.GetCDNArtifactApiRef()
	require.True(t, is)
	require.Equal(t, "myartifact", itemU.APIRef.ToFilename())
}
