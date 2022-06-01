package sdk_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ovh/cds/sdk"

	"github.com/stretchr/testify/require"
)

func TestWorkflowRunResults_Unique(t *testing.T) {
	r1, err := json.Marshal(sdk.WorkflowRunResultArtifact{
		WorkflowRunResultArtifactCommon: sdk.WorkflowRunResultArtifactCommon{
			Name: "r1",
		},
	})
	require.NoError(t, err)
	r2, err := json.Marshal(sdk.WorkflowRunResultArtifactManager{
		WorkflowRunResultArtifactCommon: sdk.WorkflowRunResultArtifactCommon{
			Name: "r2",
		},
		RepoType: "docker",
	})
	require.NoError(t, err)
	r3, err := json.Marshal(sdk.WorkflowRunResultArtifactManager{
		WorkflowRunResultArtifactCommon: sdk.WorkflowRunResultArtifactCommon{
			Name: "r3",
		},
		RepoType: "helm",
	})
	require.NoError(t, err)

	now := time.Now()

	rs := sdk.WorkflowRunResults{
		{
			ID:      "A",
			Type:    sdk.WorkflowRunResultTypeArtifact,
			SubNum:  0,
			DataRaw: r1,
			Created: now.Add(time.Second),
		},
		{
			ID:      "B",
			Type:    sdk.WorkflowRunResultTypeArtifact,
			SubNum:  1,
			DataRaw: r1,
			Created: now.Add(2 * time.Second),
		},
		{
			ID:      "C",
			Type:    sdk.WorkflowRunResultTypeArtifactManager,
			SubNum:  0,
			DataRaw: r2,
			Created: now.Add(3 * time.Second),
		},
		{
			ID:      "D",
			Type:    sdk.WorkflowRunResultTypeArtifactManager,
			SubNum:  0,
			DataRaw: r3,
			Created: now.Add(4 * time.Second),
		},
		{
			ID:      "E",
			Type:    sdk.WorkflowRunResultTypeArtifactManager,
			SubNum:  1,
			DataRaw: r3,
			Created: now.Add(5 * time.Second),
		},
	}

	res, err := rs.Unique()
	require.NoError(t, err)
	require.Len(t, res, 3)
	require.Equal(t, "B", res[0].ID)
	require.Equal(t, "C", res[1].ID)
	require.Equal(t, "E", res[2].ID)
}
