package cdn

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

// The keys reported are the ones an operator will look for, so they must be built exactly like
// the intake builds them, for a job of either generation.
func TestDebugJobIntake_ReportsTheIntakeKeys(t *testing.T) {
	var s Service
	s.Cfg.Log.StepMaxSize = 15728640

	for _, jobID := range []string{"1b5f5d2a-0a5c-4a1e-9b0e-2b0c5a6d7e8f", "42"} {
		intake, errs := s.debugJobIntake(jobID, nil)

		require.Equal(t, "cdn:log:job:"+jobID, intake.IncomingQueueKey)
		require.Equal(t, "cdn:log:incoming:size:"+jobID, intake.SizeKey)
		require.Equal(t, "cdn:log:heartbeat:"+jobID, intake.HeartbeatKey)
		require.Equal(t, int64(15728640), intake.StepMaxSize, "the budget must be reported next to the counter")
		require.Len(t, errs, 1, "without a cache the section reports its own error and returns the keys anyway")
	}
}

// One unreachable dependency must not hide everything else: the answer stays usable and says
// what could not be read.
func TestDebugJob_DegradesWithoutCacheNorDatabase(t *testing.T) {
	var s Service

	res := s.debugJob(context.TODO(), "42")

	require.Equal(t, "42", res.JobID)
	require.Empty(t, res.Items)
	require.Len(t, res.Errors, 2, "one error for the intake, one for the items")
	require.NotEmpty(t, res.Diagnosis)
	require.Equal(t, int64(defaultNbJobLogsGoroutines), res.Dequeue.MaxQueues)
}

// The diagnosis is what the operator reads first: it must name the state that explains missing
// logs, and say nothing else.
func TestDebugJobDiagnosis(t *testing.T) {
	t.Run("no item at all", func(t *testing.T) {
		d := debugJobDiagnosis(sdk.CDNDebugJob{})
		require.Contains(t, d, "no item exists for this job: nothing has ever been stored for it")
	})

	t.Run("the size counter reached the step max size", func(t *testing.T) {
		d := debugJobDiagnosis(sdk.CDNDebugJob{
			Intake: sdk.CDNDebugJobIntake{
				SizeExists:  true,
				SizeValue:   20,
				StepMaxSize: 20,
			},
		})
		require.Contains(t, d, "the size counter is at or above the step max size (20 >= 20): further non terminated lines of this job are dropped by the intake")

		// Below the budget, nothing is said about it
		d = debugJobDiagnosis(sdk.CDNDebugJob{
			Intake: sdk.CDNDebugJobIntake{SizeExists: true, SizeValue: 19, StepMaxSize: 20},
		})
		for _, line := range d {
			require.NotContains(t, line, "step max size")
		}
	})

	t.Run("messages waiting and queue ownership", func(t *testing.T) {
		d := debugJobDiagnosis(sdk.CDNDebugJob{
			Intake: sdk.CDNDebugJobIntake{
				IncomingQueueLength: 12,
				HeartbeatExists:     true,
				HeartbeatOwner:      "instance-a",
			},
		})
		require.Contains(t, d, "12 message(s) are still waiting in the incoming queue")
		require.Contains(t, d, "the incoming queue is claimed by instance-a")
	})

	t.Run("messages waiting and nobody claimed the queue", func(t *testing.T) {
		d := debugJobDiagnosis(sdk.CDNDebugJob{
			Intake: sdk.CDNDebugJobIntake{IncomingQueueLength: 3},
		})
		require.Contains(t, d, "the incoming queue is not claimed by any instance")
	})

	t.Run("the answering instance is at its dequeue cap", func(t *testing.T) {
		d := debugJobDiagnosis(sdk.CDNDebugJob{
			Dequeue: sdk.CDNDebugJobDequeue{ClaimedQueues: 10, MaxQueues: 10},
		})
		require.Contains(t, d, "the instance that answered is at its dequeue cap (10/10)")
	})

	t.Run("items are counted by status and deletion", func(t *testing.T) {
		d := debugJobDiagnosis(sdk.CDNDebugJob{
			Items: []sdk.CDNDebugJobItem{
				{Status: sdk.CDNStatusItemCompleted},
				{Status: sdk.CDNStatusItemIncoming},
				{Status: sdk.CDNStatusItemCompleted, ToDelete: true},
			},
		})
		require.Contains(t, d, "3 item(s) found: 2 completed, 1 incoming")
		require.Contains(t, d, "1 item(s) are marked for deletion")
	})

	t.Run("a truncated list says so", func(t *testing.T) {
		d := debugJobDiagnosis(sdk.CDNDebugJob{
			Items:          []sdk.CDNDebugJobItem{{Status: sdk.CDNStatusItemCompleted}},
			ItemsTruncated: true,
		})
		require.Contains(t, d, "the item list is truncated at 100")
	})
}
