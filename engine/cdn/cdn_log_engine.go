package cdn

import (
	"context"
	"strings"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

var (
	keyJobLogQueue = cache.Key("cdn", "log", "job")
	keyJobLogSize  = cache.Key("cdn", "log", "incoming", "size")

	// Dequeue keys
	keyJobHearbeat = cache.Key("cdn", "log", "heartbeat")
	keyJobLock     = cache.Key("cdn", "log", "lock")
)

// Check all job queues to know and start dequeue if needed
func (s *Service) waitingJobs(ctx context.Context) {
	for {
		time.Sleep(250 * time.Millisecond)
		select {
		case <-ctx.Done():
			return
		default:
			// List all queues
			keyListQueue := cache.Key(keyJobLogQueue, "*")
			listKeys, err := s.Cache.Keys(keyListQueue)
			if err != nil {
				err = sdk.WrapError(err, "unable to list jobs queues %s", keyListQueue)
				ctx = sdk.ContextWithStacktrace(ctx, err)
				log.Error(ctx, err.Error())
				continue
			}

			// For each key, check if heartbeat key exist
			for _, k := range listKeys {
				keyParts := strings.Split(k, ":")
				queueIdentifier := keyParts[len(keyParts)-1]

				jobQueueKey, err := s.canDequeue(ctx, queueIdentifier)
				if err != nil {
					err = sdk.WrapError(err, "unable to check canDequeue %s", jobQueueKey)
					ctx = sdk.ContextWithStacktrace(ctx, err)
					log.Error(ctx, err.Error())
					continue
				}
				if jobQueueKey == "" {
					continue
				}

				s.GoRoutines.Exec(ctx, "cdn-dequeue-job-message", func(ctx context.Context) {
					if err := s.dequeueMessages(ctx, jobQueueKey, queueIdentifier); err != nil {
						err = sdk.WrapError(err, "unable to dequeue redis incoming job queue")
						ctx = sdk.ContextWithStacktrace(ctx, err)
						log.Error(ctx, err.Error())
					}
				})
			}
		}
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

	defer func() {
		// Remove heartbeat
		_ = s.Cache.Delete(cache.Key(keyJobHearbeat, queueIdentifier))
	}()

	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			b, err := s.Cache.Exist(jobLogsQueueKey)
			if err != nil {
				err = sdk.WrapError(err, "unable to check if queue still exist")
				ctx = sdk.ContextWithStacktrace(ctx, err)
				log.Error(ctx, err.Error())
				continue
			} else if !b {
				// leave dequeue if queue does not exist anymore
				log.Info(ctx, "dequeueJobMessages: leaving job queue %s (queue no more exists)", jobLogsQueueKey)
				return nil
			}
			// heartbeat
			heartbeatKey := cache.Key(keyJobHearbeat, queueIdentifier)
			if err := s.Cache.SetWithTTL(heartbeatKey, true, 30); err != nil {
				err = sdk.WrapError(err, "unable to heartbeat %s", heartbeatKey)
				ctx = sdk.ContextWithStacktrace(ctx, err)
				log.Error(ctx, err.Error())
				continue
			}
		default:
			dequeuCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
			msgs, err := s.Cache.DequeueJSONRawMessagesWithContext(dequeuCtx, jobLogsQueueKey, 1*time.Millisecond, 1000)
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
					ctx = sdk.ContextWithStacktrace(ctx, err)
					log.Error(ctx, err.Error())
				}
				nbMessages += len(msgs)
				t1 = time.Now()
			}
			if err != nil {
				if strings.Contains(err.Error(), "context deadline exceeded") {
					continue
				}
				err = sdk.WrapError(err, "unable to dequeue job logs queue %s", jobLogsQueueKey)
				ctx = sdk.ContextWithStacktrace(ctx, err)
				log.Error(ctx, err.Error())
				continue
			}
		}
	}
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
	if err := s.Cache.SetWithTTL(heartbeatKey, true, 30); err != nil {
		return "", err
	}
	return jobQueueKey, nil
}
