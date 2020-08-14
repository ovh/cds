package cdn

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var (
	keyJobLogQueue = cache.Key("cdn", "log", "job")
	keyJobHearbeat = cache.Key("cdn", "log", "heartbeat")
	keyJobLock     = cache.Key("cdn", "log", "lock")
)

func (s *Service) waitingJobs(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// List all queues
			keyListQueue := cache.Key(keyJobLogQueue, "*")
			listKeys, err := s.Cache.Keys(keyListQueue)
			if err != nil {
				log.Error(ctx, "waitingJobs: unable to list jobs queues %s", keyListQueue)
				continue
			}

			// For each key, check if heartbeat key exist
			for _, k := range listKeys {
				keyParts := strings.Split(k, ":")
				jobID := keyParts[len(keyParts)-1]

				jobQueueKey, err := s.canDequeue(jobID)
				if err != nil {
					log.Error(ctx, "waitingJobs: unable to check canDequeue %s: %v", jobQueueKey, err)
					continue
				}
				if jobQueueKey == "" {
					continue
				}

				sdk.GoRoutine(ctx, "cdn-dequeue-job-message", func(ctx context.Context) {
					if err := s.dequeueJobMessages(ctx, jobQueueKey, jobID); err != nil {
						log.Error(ctx, "waitingJobs: unable to dequeue redis incoming job queue: %v", err)
					}
				})
			}
			time.Sleep(250 * time.Millisecond)
		}
	}
}

func (s *Service) dequeueJobMessages(ctx context.Context, jobLogsQueueKey string, jobID string) error {
	log.Info(ctx, "dequeueJobMessages: Dequeue %s", jobLogsQueueKey)
	var t0 = time.Now()
	var t1 = time.Now()
	var nbMessages int
	defer func() {
		delta := t1.Sub(t0)
		log.Info(ctx, "dequeueJobMessages: processLogs[%s] - %d messages received in %v", jobLogsQueueKey, nbMessages, delta)
	}()

	defer func() {
		// Remove heartbeat
		_ = s.Cache.Delete(cache.Key(keyJobHearbeat, jobID))
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
				log.Error(ctx, "dequeueJobMessages: unable to check if queue still exist: %v", err)
				continue
			} else if !b {
				// leave dequeue if queue does not exist anymore
				log.Info(ctx, "dequeueJobMessages: leaving job queue %s (queue no more exists)", jobLogsQueueKey)
				return nil
			}
			// heartbeat
			heartbeatKey := cache.Key(keyJobHearbeat, jobID)
			if err := s.Cache.SetWithTTL(heartbeatKey, true, 30); err != nil {
				log.Error(ctx, "dequeueJobMessages: unable to hearbeat %s: %v", heartbeatKey, err)
				continue
			}
		default:
			dequeuCtx, cancel := context.WithTimeout(ctx, 15*time.Second)

			msgs, err := s.Cache.DequeueJSONRawMessagesWithContext(dequeuCtx, jobLogsQueueKey, 10*time.Millisecond, 100)
			cancel()

			if len(msgs) > 0 {
				hms := make(map[string]handledMessage, len(msgs))
				for _, msg := range msgs {
					var hm handledMessage
					if err := json.Unmarshal(msg, &hm); err != nil {
						return sdk.WrapError(err, "redis.DequeueWithContext> error on unmarshal value on queue:%s", jobLogsQueueKey)
					}
					if hm.Signature.Worker == nil {
						continue
					}
					nbMessages++
					k := fmt.Sprintf("%d-%d-%d", hm.Signature.JobID, hm.Signature.NodeRunID, hm.Signature.Worker.StepOrder)
					if _, ok := hms[k]; ok {
						full := hms[k].Msg.Full
						if !strings.HasSuffix(full, "\n") {
							full += "\n"
						}
						hm.Msg.Full = fmt.Sprintf("%s[%s] %s", full, getLevelString(hm.Msg.Level), hm.Msg.Full)
						hms[k] = hm
					} else {
						hms[k] = hm
					}
				}

				for _, hm := range hms {
					now := time.Now()

					currentLog := buildMessage(hm.Signature, hm.Msg)
					l := sdk.Log{
						JobID:        hm.Signature.JobID,
						NodeRunID:    hm.Signature.NodeRunID,
						LastModified: &now,
						StepOrder:    hm.Signature.Worker.StepOrder,
						Val:          currentLog,
					}
					if err := s.Client.QueueSendLogs(ctx, hm.Signature.JobID, l); err != nil {
						log.Error(ctx, "dequeueJobMessages: unable to send log to API: %v", err)
						continue
					}
				}

				t1 = time.Now()

			} else if err != nil {
				if strings.Contains(err.Error(), "context deadline exceeded") {
					continue
				}
				log.Error(ctx, "dequeueJobMessages: unable to dequeue job logs queue %s: %v", jobLogsQueueKey, err)
				continue
			}
		}
	}
}

func (s *Service) canDequeue(jobID string) (string, error) {
	jobQueueKey := cache.Key(keyJobLogQueue, jobID)
	heatbeatKey := cache.Key(keyJobHearbeat, jobID)

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

	exist, err := s.Cache.Exist(heatbeatKey)
	if err != nil {
		return "", err
	}
	// if key exist, that mean that someone is already dequeuing
	if exist {
		return "", nil
	}

	//hearbeat
	heartbeatKey := cache.Key(keyJobHearbeat, jobID)
	if err := s.Cache.SetWithTTL(heartbeatKey, true, 30); err != nil {
		return "", err
	}
	return jobQueueKey, nil
}
