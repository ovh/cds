package api

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComputeWorkflowRunHookFullName(t *testing.T) {
	const (
		projectKey = "TATA"
		defaultVCS = "myvcs"
		// repositories are stored lowercase, the default is already normalized
		defaultRepo = "myorg/myrepo"
	)

	tests := []struct {
		name     string
		watched  string
		expected string
	}{
		{
			name:     "workflow name only completes vcs and repo from the workflow location",
			watched:  "myworkflow",
			expected: "TATA/myvcs/myorg/myrepo/myworkflow",
		},
		{
			name:     "org/repo/workflow lowercases the repo, keeps the workflow name case",
			watched:  "Org/Repo/MyWorkflow",
			expected: "TATA/myvcs/org/repo/MyWorkflow",
		},
		{
			name:     "vcs/org/repo/workflow lowercases the repo, keeps the vcs case",
			watched:  "TheVCS/Org/Repo/myworkflow",
			expected: "TATA/TheVCS/org/repo/myworkflow",
		},
		{
			name:     "full form lowercases only the repo segment",
			watched:  "TATA/vcs/MY/repo/myworkflow",
			expected: "TATA/vcs/my/repo/myworkflow",
		},
		{
			name:     "full form preserves project, vcs and workflow name case",
			watched:  "TaTa/VcS/MY/REPO/MyWorkflow",
			expected: "TaTa/VcS/my/repo/MyWorkflow",
		},
		{
			name:     "unsupported segment count returns empty",
			watched:  "vcs/repo",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, computeWorkflowRunHookFullName(projectKey, defaultVCS, defaultRepo, tt.watched))
		})
	}
}
