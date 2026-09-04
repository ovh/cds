package event_v2

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

func stepUpdateOf(id string, steps ...string) sdk.V2WorkflowRunJob {
	rj := sdk.V2WorkflowRunJob{ID: id, StepsStatus: sdk.JobStepsStatus{}}
	for _, s := range steps {
		rj.StepsStatus[s] = sdk.JobStepStatus{}
	}
	return rj
}

func TestStepUpdates_AnnouncesTheFirstOfAWindowAndTheLast(t *testing.T) {
	s := &stepUpdates{pending: make(map[string]*heldStepUpdate)}
	at := time.Now()

	require.True(t, s.take(stepUpdateOf("job1", "one"), at), "the first update of a job opens its window and goes out")
	require.False(t, s.take(stepUpdateOf("job1", "one", "two"), at.Add(time.Millisecond)), "an update of an open window is held back")
	require.False(t, s.take(stepUpdateOf("job1", "one", "two", "three"), at.Add(2*time.Millisecond)), "and so is the next one")

	held := s.flush()
	require.Len(t, held, 1, "one announcement is held back per job, whatever the number of updates")
	require.Len(t, held[0].runJob.StepsStatus, 3, "the last update supersedes the ones it replaces")
	require.Equal(t, at.Add(2*time.Millisecond), held[0].at, "held back with the moment it was reported, not the moment it goes out")
}

func TestStepUpdates_KeepsTheJobsApart(t *testing.T) {
	s := &stepUpdates{pending: make(map[string]*heldStepUpdate)}
	at := time.Now()

	require.True(t, s.take(stepUpdateOf("job1", "one"), at))
	require.True(t, s.take(stepUpdateOf("job2", "one"), at), "a window is open per job, not for all of them")
	require.False(t, s.take(stepUpdateOf("job1", "one", "two"), at))

	held := s.flush()
	require.Len(t, held, 1)
	require.Equal(t, "job1", held[0].runJob.ID)
}

func TestStepUpdates_ForgetsAJobThatWentQuiet(t *testing.T) {
	s := &stepUpdates{pending: make(map[string]*heldStepUpdate)}
	at := time.Now()

	require.True(t, s.take(stepUpdateOf("job1", "one"), at))

	require.Empty(t, s.flush(), "nothing was held back during the window")
	require.Empty(t, s.pending, "a job that held nothing back is forgotten with its window")

	require.True(t, s.take(stepUpdateOf("job1", "one", "two"), at), "and its next update goes out right away")
}

func TestStepUpdates_ReopensTheWindowAfterAFlush(t *testing.T) {
	s := &stepUpdates{pending: make(map[string]*heldStepUpdate)}
	at := time.Now()

	require.True(t, s.take(stepUpdateOf("job1", "one"), at))
	require.False(t, s.take(stepUpdateOf("job1", "one", "two"), at))
	require.Len(t, s.flush(), 1)

	require.False(t, s.take(stepUpdateOf("job1", "one", "two", "three"), at), "the flush left a window open behind it")
	require.Len(t, s.flush(), 1)
}
