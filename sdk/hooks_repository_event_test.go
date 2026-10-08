package sdk

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHookRepositoryEventToInsightReport(t *testing.T) {
	analyses := []HookRepositoryEventAnalysis{
		{ProjectKey: "A", Status: RepositoryAnalysisStatusError, Error: "unable to retrieve files"},
		{ProjectKey: "B", Status: RepositoryAnalysisStatusSucceed},
	}

	failed := HookRepositoryEvent{UUID: "uuid", EventName: WorkflowHookEventNamePush, Status: HookEventStatusError, LastError: "1 of 2 repository analyses failed", Analyses: analyses}
	require.Equal(t, "Event \"push\" (uuid): Error\n\n1 of 2 repository analyses failed\n\nOn project A: unable to retrieve files", failed.ToInsightReport("http://cds").Detail)

	done := HookRepositoryEvent{UUID: "uuid", EventName: WorkflowHookEventNamePush, Status: HookEventStatusDone, Analyses: analyses}
	require.Equal(t, "Event \"push\" (uuid): Done\n\nOn project A: unable to retrieve files", done.ToInsightReport("http://cds").Detail)
}
