package cdn

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	_ "github.com/ovh/cds/engine/cdn/storage/local"
	_ "github.com/ovh/cds/engine/cdn/storage/redis"
	cdntest "github.com/ovh/cds/engine/cdn/test"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdn"
	"github.com/ovh/cds/sdk/log/hook/graylog"
)

func newTestLogEngineService(t *testing.T) (*Service, context.Context) {
	m := gorpmapper.New()
	item.InitDBMapping(m)
	storage.InitDBMapping(m)

	log.Factory = log.NewTestingWrapper(t)
	db, factory, store, cancelDB := test.SetupPGToCancel(t, m, sdk.TypeCDN)
	t.Cleanup(cancelDB)

	cdntest.ClearItem(t, context.TODO(), m, db)

	s := &Service{
		DBConnectionFactory: factory,
		Cache:               store,
		Mapper:              m,
	}
	s.GoRoutines = sdk.NewGoRoutines(context.TODO())
	s.dequeueOwnerID = sdk.UUID()

	ctx, cancel := context.WithCancel(context.TODO())
	t.Cleanup(cancel)
	s.Units = newRunningStorageUnits(t, m, db.DbMap, ctx, store)

	// The redis instance is shared across tests: sweep stale job log keys
	for _, pattern := range []string{keyJobLogQueue + ":*", keyJobHearbeat + ":*", keyJobLock + ":*"} {
		keys, err := s.Cache.Keys(pattern)
		require.NoError(t, err)
		for _, k := range keys {
			require.NoError(t, s.Cache.Delete(k))
		}
	}

	return s, ctx
}

func newTestStepLogMessage(projectKey string, line int) handledMessage {
	return handledMessage{
		Msg: graylog.Message{
			Full: fmt.Sprintf("log line %d", line),
		},
		Signature: cdn.Signature{
			ProjectKey:   projectKey,
			WorkflowID:   1,
			WorkflowName: "MyWorkflow",
			RunID:        1,
			NodeRunID:    1,
			NodeRunName:  "MyPipeline",
			JobName:      "MyJob",
			JobID:        1,
			Worker: &cdn.SignatureWorker{
				StepName:  "script1",
				StepOrder: 1,
			},
		},
	}
}

// A CDN instance at capacity must not claim more job queues: claiming sets a 30s heartbeat
// that prevents every other instance from dequeuing the queue, so an over-claiming instance
// both balloons its own memory and starves the rest of the fleet. Unbounded claiming is what
// turned a single pod OOM into a fleet-wide crash loop (incident 2026-08-17).
func TestClaimJobQueuesRespectsCap(t *testing.T) {
	s, ctx := newTestLogEngineService(t)
	s.Cfg.Log.NbJobLogsGoroutines = 1
	s.Cfg.Log.DequeueBatchSize = 100

	queueIdentifier := sdk.UUID()
	queueKey := keyJobLogQueue + ":" + queueIdentifier
	heartbeatKey := keyJobHearbeat + ":" + queueIdentifier

	hm := newTestStepLogMessage(sdk.RandomString(10), 1)
	require.NoError(t, s.Cache.Enqueue(queueKey, hm))

	// Saturate the instance: claimJobQueues must return without touching the queue,
	// and above all without setting a heartbeat it will not service.
	s.dequeuingJobQueues.Store(1)
	s.claimJobQueues(ctx)

	exist, err := s.Cache.Exist(heartbeatKey)
	require.NoError(t, err)
	require.False(t, exist, "a saturated instance must not set a heartbeat on a queue it will not dequeue")
	queueLen, err := s.Cache.QueueLen(queueKey)
	require.NoError(t, err)
	require.Equal(t, 1, queueLen, "a saturated instance must leave the queue content in redis")

	// Free the slot: the queue must now be claimed and drained.
	s.dequeuingJobQueues.Store(0)
	s.claimJobQueues(ctx)

	require.Eventually(t, func() bool {
		queueLen, err := s.Cache.QueueLen(queueKey)
		require.NoError(t, err)
		return queueLen == 0
	}, 15*time.Second, 100*time.Millisecond, "the queue must be drained once a dequeue slot is available")

	// The dequeue goroutine exits after noticing the emptied queue and frees its slot.
	require.Eventually(t, func() bool {
		return s.dequeuingJobQueues.Load() == 0
	}, 20*time.Second, 100*time.Millisecond, "the dequeue slot must be released when the dequeuer exits")
}

// The queue ownership heartbeat must stay alive while a batch of messages is being written
// to the buffer. Before the fix, the heartbeat was only refreshed between batches: a batch
// slower than the heartbeat TTL let another dequeuer claim the same queue, duplicating the
// memory and database load — the amplification loop behind the OOM cascade (incident 2026-08-17).
func TestDequeueMessagesKeepsHeartbeatDuringSlowBatch(t *testing.T) {
	s, ctx := newTestLogEngineService(t)
	s.Cfg.Log.DequeueBatchSize = 1000

	previousRefreshPeriod, previousTTL := jobHeartbeatRefreshPeriod, jobHeartbeatTTLSec
	jobHeartbeatRefreshPeriod, jobHeartbeatTTLSec = 200*time.Millisecond, 1
	t.Cleanup(func() {
		jobHeartbeatRefreshPeriod, jobHeartbeatTTLSec = previousRefreshPeriod, previousTTL
	})

	queueIdentifier := sdk.UUID()
	heartbeatKey := keyJobHearbeat + ":" + queueIdentifier

	// Enough messages so that dequeuing and storing them outlives the 1s heartbeat TTL.
	projectKey := sdk.RandomString(10)
	nbMessages := 2000
	for i := 0; i < nbMessages; i++ {
		hm := newTestStepLogMessage(projectKey, i)
		require.NoError(t, s.Cache.Enqueue(keyJobLogQueue+":"+queueIdentifier, hm))
	}

	queueKey, err := s.canDequeue(ctx, queueIdentifier)
	require.NoError(t, err)
	require.NotEmpty(t, queueKey)

	dequeueCtx, cancelDequeue := context.WithCancel(ctx)
	t.Cleanup(cancelDequeue)
	done := make(chan error, 1)
	start := time.Now()
	go func() {
		done <- s.dequeueMessages(dequeueCtx, queueKey, queueIdentifier)
	}()

	// While the queue is being drained, the heartbeat must exist at every poll: any gap
	// means another instance could claim the queue and duplicate the dequeuer.
	for {
		queueLen, err := s.Cache.QueueLen(queueKey)
		require.NoError(t, err)
		if queueLen == 0 {
			break
		}
		exist, err := s.Cache.Exist(heartbeatKey)
		require.NoError(t, err)
		require.True(t, exist, "the heartbeat expired while messages were still being processed")
		time.Sleep(100 * time.Millisecond)
	}

	// Prove the test had power: the drain outlived the heartbeat TTL, so an unrefreshed
	// heartbeat would have expired mid-processing.
	require.Greater(t, time.Since(start), time.Duration(jobHeartbeatTTLSec)*time.Second,
		"processing must outlive the heartbeat TTL for this test to be meaningful")

	cancelDequeue()
	require.ErrorIs(t, <-done, context.Canceled)
}

// A dequeuer releasing a job queue must not delete a heartbeat now owned by another
// instance, otherwise the queue gets claimed twice.
func TestReleaseJobQueueHeartbeatOwnerCheck(t *testing.T) {
	s, _ := newTestLogEngineService(t)

	queueIdentifier := sdk.UUID()
	heartbeatKey := keyJobHearbeat + ":" + queueIdentifier

	// Heartbeat owned by another instance: must be kept.
	require.NoError(t, s.Cache.SetWithTTL(heartbeatKey, sdk.UUID(), 30))
	s.releaseJobQueueHeartbeat(queueIdentifier)
	exist, err := s.Cache.Exist(heartbeatKey)
	require.NoError(t, err)
	require.True(t, exist, "a heartbeat owned by another instance must not be deleted")

	// Heartbeat owned by this instance: must be deleted.
	require.NoError(t, s.Cache.SetWithTTL(heartbeatKey, s.dequeueOwnerID, 30))
	s.releaseJobQueueHeartbeat(queueIdentifier)
	exist, err = s.Cache.Exist(heartbeatKey)
	require.NoError(t, err)
	require.False(t, exist, "a heartbeat owned by this instance must be deleted on release")

	// Legacy heartbeat (boolean value, written by a pre-upgrade instance): fall back to
	// the unconditional delete for rolling-deploy compatibility.
	require.NoError(t, s.Cache.SetWithTTL(heartbeatKey, true, 30))
	s.releaseJobQueueHeartbeat(queueIdentifier)
	exist, err = s.Cache.Exist(heartbeatKey)
	require.NoError(t, err)
	require.False(t, exist, "a legacy heartbeat must be deleted on release")
}
