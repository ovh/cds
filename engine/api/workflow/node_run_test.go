package workflow

import (
	"testing"

	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

// Test merge function with multiple subrun but not a restartfailedjob use case
// Must return artifact from 1st noderun
func TestMergeArtifactWithPreviousSubRun_MultiRunNoRestart(t *testing.T) {
	run1 := sdk.WorkflowNodeRun{
		Artifacts: []sdk.WorkflowNodeRunArtifact{
			{
				Name: "art1",
			},
			{
				Name: "art2",
			},
			{
				Name: "art3",
			},
		},
	}
	run2 := sdk.WorkflowNodeRun{
		Artifacts: []sdk.WorkflowNodeRunArtifact{
			{
				Name: "art4",
			},
			{
				Name: "art5",
			},
			{
				Name: "art6",
			},
		},
	}
	arts := MergeArtifactWithPreviousSubRun([]sdk.WorkflowNodeRun{run1, run2})

	assert.Equal(t, run1.Artifacts, arts)
}

// Test merge function with only 1 subrun.
// Must return artifact from 1st noderun
func TestMergeArtifactWithPreviousSubRun_1Run(t *testing.T) {
	run := sdk.WorkflowNodeRun{
		Artifacts: []sdk.WorkflowNodeRunArtifact{
			{
				Name: "art1",
			},
			{
				Name: "art2",
			},
			{
				Name: "art3",
			},
		},
	}
	arts := MergeArtifactWithPreviousSubRun([]sdk.WorkflowNodeRun{run})

	assert.Equal(t, run.Artifacts, arts)
}

// Test merge function with only 0 subrun.
// Must returnempty slice
func TestMergeArtifactWithPreviousSubRun_NoRuns(t *testing.T) {
	arts := MergeArtifactWithPreviousSubRun(nil)
	assert.Equal(t, 0, len(arts))
}

// Test merge function with only multiple subrun with restartfailedjob usecase.
// Must must merge artifacts form all subrun
func TestMergeArtifactWithPreviousSubRun_MultipleRestartFailedJobs(t *testing.T) {
	run1 := sdk.WorkflowNodeRun{
		Artifacts: []sdk.WorkflowNodeRunArtifact{
			{
				Name:   "art1",
				MD5sum: "11",
			},
			{
				Name:   "art2",
				MD5sum: "12",
			},
		},
		Manual: &sdk.WorkflowNodeRunManual{OnlyFailedJobs: true},
	}
	run2 := sdk.WorkflowNodeRun{
		Artifacts: []sdk.WorkflowNodeRunArtifact{
			{
				Name:   "art2",
				MD5sum: "21",
			},
			{
				Name:   "art3",
				MD5sum: "22",
			},
		},
		Manual: &sdk.WorkflowNodeRunManual{OnlyFailedJobs: true},
	}
	run3 := sdk.WorkflowNodeRun{
		Artifacts: []sdk.WorkflowNodeRunArtifact{
			{
				Name:   "art3",
				MD5sum: "31",
			},
			{
				Name:   "art4",
				MD5sum: "32",
			},
		},
	}
	arts := MergeArtifactWithPreviousSubRun([]sdk.WorkflowNodeRun{run1, run2, run3})

	assert.Equal(t, 4, len(arts))
	assert.Equal(t, "11", arts[0].MD5sum)
	assert.Equal(t, "12", arts[1].MD5sum)
	assert.Equal(t, "22", arts[2].MD5sum)
	assert.Equal(t, "32", arts[3].MD5sum)
}

// Test merge function with only multiple subrun with only 1 restartfailedjob.
// Must must merge artifacts form run1 and 2 only.
func TestMergeArtifactWithPreviousSubRun_NonMultipleRestartFailedJobs(t *testing.T) {
	run1 := sdk.WorkflowNodeRun{
		Artifacts: []sdk.WorkflowNodeRunArtifact{
			{
				Name:   "art1",
				MD5sum: "11",
			},
			{
				Name:   "art2",
				MD5sum: "12",
			},
		},
		Manual: &sdk.WorkflowNodeRunManual{OnlyFailedJobs: true},
	}
	run2 := sdk.WorkflowNodeRun{
		Artifacts: []sdk.WorkflowNodeRunArtifact{
			{
				Name:   "art2",
				MD5sum: "21",
			},
			{
				Name:   "art3",
				MD5sum: "22",
			},
		},
	}
	run3 := sdk.WorkflowNodeRun{
		Artifacts: []sdk.WorkflowNodeRunArtifact{
			{
				Name:   "art3",
				MD5sum: "31",
			},
			{
				Name:   "art4",
				MD5sum: "32",
			},
		},
	}
	arts := MergeArtifactWithPreviousSubRun([]sdk.WorkflowNodeRun{run1, run2, run3})

	assert.Equal(t, 3, len(arts))
	assert.Equal(t, "11", arts[0].MD5sum)
	assert.Equal(t, "12", arts[1].MD5sum)
	assert.Equal(t, "22", arts[2].MD5sum)
}
