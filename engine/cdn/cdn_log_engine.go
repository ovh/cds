package cdn

import (
	"context"
	"strings"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

var (
	keyJobLogQueue = cache.Key("cdn", "log", "job")
	keyJobLogSize  = cache.Key("cdn", "log", "incoming", "size")

	// Dequeue keys
	keyJobHearbeat = cache.Key("cdn", "log", "heartbeat")
	keyJobLock     = cache.Key("cdn", "log", "lock")
)

// Heartbeat timings are package variables so tests can shrink them.
var (
	jobHeartbeatRefreshPeriod = 5 * time.Second
	jobHeartbeatTTLSec        = 30
)

func (s *Service) maxJobLogsGoroutines() int64 {
	if v := s.Cfg.Log.NbJobLogsGoroutines; v > 0 {
		return v
	}
	return defaultNbJobLogsGoroutines
}

func (s *Service) dequeueBatchSize() int {
	if v := s.Cfg.Log.DequeueBatchSize; v > 0 {
		return v
	}
	return defaultDequeueBatchSize
}

// Check all job queues to know and start dequeue if needed
func (s *Service) waitingJobs(ctx context.Context) {
	for {
		time.Sleep(250 * time.Millisecond)
		select {
		case <-ctx.Done():
			return
		default:
			s.claimJobQueues(ctx)
		}
	}
}

// claimJobQueues claims unattended job queues and starts a dequeue goroutine for each of them,
// bounded by NbJobLogsGoroutines. The cap must be checked BEFORE canDequeue: a saturated
// instance must never set a heartbeat it will not service, since a phantom heartbeat prevents
// every other instance from claiming the queue for its whole TTL.
func (s *Service) claimJobQueues(ctx context.Context) {
	maxDequeuers := s.maxJobLogsGoroutines()
	// At capacity: don't even scan. Queues stay safely in Redis (bounded by StepMaxSize)
	// until a local slot frees up or another instance claims them.
	if s.dequeuingJobQueues.Load() >= maxDequeuers {
		return
	}

	// List all queues
	keyListQueue := cache.Key(keyJobLogQueue, "*")
	listKeys, err := s.Cache.Keys(keyListQueue)
	if err != nil {
		err = sdk.WrapError(err, "unable to list jobs queues %s", keyListQueue)
		log.Error(sdk.ContextWithStacktrace(ctx, err), err.Error())
		return
	}

	// For each key, check if heartbeat key exist
	for _, k := range listKeys {
		if s.dequeuingJobQueues.Load() >= maxDequeuers {
			return
		}

		keyParts := strings.Split(k, ":")
		queueIdentifier := keyParts[len(keyParts)-1]

		jobQueueKey, err := s.canDequeue(ctx, queueIdentifier)
		if err != nil {
			err = sdk.WrapError(err, "unable to check canDequeue %s", jobQueueKey)
			log.Error(sdk.ContextWithStacktrace(ctx, err), err.Error())
			continue
		}
		if jobQueueKey == "" {
			continue
		}

		telemetry.Record(ctx, s.Metrics.dequeuedJobQueues, s.dequeuingJobQueues.Add(1))
		s.GoRoutines.Exec(ctx, "cdn-dequeue-job-message", func(ctx context.Context) {
			defer func() {
				telemetry.Record(ctx, s.Metrics.dequeuedJobQueues, s.dequeuingJobQueues.Add(-1))
			}()
			if err := s.dequeueMessages(ctx, jobQueueKey, queueIdentifier); err != nil {
				err = sdk.WrapError(err, "unable to dequeue redis incoming job queue")
				log.Error(sdk.ContextWithStacktrace(ctx, err), err.Error())
			}
		})
	}
}

// Run dequeue of a job log
func (s *Service) dequeueMessages(ctx context.Context, jobLogsQueueKey string, queueIdentifier string) error {
	log.Info(ctx, "dequeueJobMessages: %s BEGIN dequeue", jobLogsQueueKey)
	var t0 = time.Now()
	var t1 = time.Now()
	var nbMessages int
	defer func() {
		delta := t1.Sub(t0)
		log.Info(ctx, "dequeueJobMessages: %s END dequeue - %d messages received in %v", jobLogsQueueKey, nbMessages, delta)
	}()

	// Refresh the heartbeat from a dedicated goroutine so that queue ownership does not
	// depend on batch processing latency: a batch slower than the heartbeat TTL would
	// otherwise let another dequeuer claim the same queue and duplicate the load.
	heartbeatKey := cache.Key(keyJobHearbeat, queueIdentifier)
	heartbeatCtx, cancelHeartbeat := context.WithCancel(ctx)
	defer cancelHeartbeat()
	s.GoRoutines.Exec(heartbeatCtx, "cdn-dequeue-heartbeat", func(ctx context.Context) {
		tick := time.NewTicker(jobHeartbeatRefreshPeriod)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				if err := s.Cache.SetWithTTL(heartbeatKey, s.dequeueOwnerID, jobHeartbeatTTLSec); err != nil {
					err = sdk.WrapError(err, "unable to heartbeat %s", heartbeatKey)
					log.Error(sdk.ContextWithStacktrace(ctx, err), err.Error())
				}
			}
		}
	})

	defer s.releaseJobQueueHeartbeat(queueIdentifier)

	batchSize := s.dequeueBatchSize()
	lastQueueCheck := time.Now()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		dequeuCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
		msgs, err := s.Cache.DequeueJSONRawMessagesWithContext(dequeuCtx, jobLogsQueueKey, 1*time.Millisecond, batchSize)
		cancel()
		if len(msgs) > 0 {
			hms := make([]handledMessage, 0, len(msgs))
			for _, m := range msgs {
				var hm handledMessage
				if err := sdk.JSONUnmarshal(m, &hm); err != nil {
					return sdk.WithStack(err)
				}
				hms = append(hms, hm)
			}

			// Send TO CDN Buffer
			if err := s.sendToBufferWithRetry(ctx, hms); err != nil {
				err = sdk.WrapError(err, "unable to send log into buffer")
				log.Error(sdk.ContextWithStacktrace(ctx, err), err.Error())
			}
			telemetry.Record(ctx, s.Metrics.dequeuedMessages, int64(len(msgs)))
			nbMessages += len(msgs)
			t1 = time.Now()
		}
		if err != nil && !strings.Contains(err.Error(), "context deadline exceeded") {
			err = sdk.WrapError(err, "unable to dequeue job logs queue %s", jobLogsQueueKey)
			log.Error(sdk.ContextWithStacktrace(ctx, err), err.Error())
			continue
		}

		// Leave dequeue if the queue does not exist anymore (redis deletes an emptied list)
		if time.Since(lastQueueCheck) >= 5*time.Second {
			lastQueueCheck = time.Now()
			b, err := s.Cache.Exist(jobLogsQueueKey)
			if err != nil {
				err = sdk.WrapError(err, "unable to check if queue still exist")
				log.Error(sdk.ContextWithStacktrace(ctx, err), err.Error())
				continue
			}
			if !b {
				log.Info(ctx, "dequeueJobMessages: leaving job queue %s (queue no more exists)", jobLogsQueueKey)
				return nil
			}
		}
	}
}

// releaseJobQueueHeartbeat deletes the heartbeat only if this instance still owns it.
// Legacy heartbeats (boolean value) and heartbeats without owner fall back to an
// unconditional delete.
func (s *Service) releaseJobQueueHeartbeat(queueIdentifier string) {
	key := cache.Key(keyJobHearbeat, queueIdentifier)
	if s.dequeueOwnerID != "" {
		var owner string
		if ok, _ := s.Cache.Get(key, &owner); ok && owner != "" && owner != s.dequeueOwnerID {
			// another instance claimed the queue in the meantime; don't kill its heartbeat
			return
		}
	}
	_ = s.Cache.Delete(key)
}

// Return queue name if jobID need to be dequeue or empty
func (s *Service) canDequeue(ctx context.Context, jobID string) (string, error) {
	jobQueueKey := cache.Key(keyJobLogQueue, jobID)
	heartbeatKey := cache.Key(keyJobHearbeat, jobID)

	// Take a lock
	lockKey := cache.Key(keyJobLock, jobID)
	b, err := s.Cache.Lock(lockKey, 5*time.Second, 0, 1)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = s.Cache.Unlock(lockKey)
	}()
	if !b {
		return "", nil
	}

	exist, err := s.Cache.Exist(heartbeatKey)
	if err != nil {
		return "", err
	}
	// if key exist, that mean that someone is already dequeuing
	if exist {
		return "", nil
	}

	//hearbeat
	log.Info(ctx, "heartbeat: take job %s", jobQueueKey)
	if err := s.Cache.SetWithTTL(heartbeatKey, s.dequeueOwnerID, jobHeartbeatTTLSec); err != nil {
		return "", err
	}
	return jobQueueKey, nil
}
