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
				log.ErrorWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
				continue
			}

			// For each key, check if heartbeat key exist
			for _, k := range listKeys {
				keyParts := strings.Split(k, ":")
				jobID := keyParts[len(keyParts)-1]

				jobQueueKey, err := s.canDequeue(jobID)
				if err != nil {
					err = sdk.WrapError(err, "unable to check canDequeue %s", jobQueueKey)
					log.ErrorWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
					continue
				}
				if jobQueueKey == "" {
					continue
				}

				s.GoRoutines.Exec(ctx, "cdn-dequeue-job-message", func(ctx context.Context) {
					if err := s.dequeueJobMessages(ctx, jobQueueKey, jobID); err != nil {
						err = sdk.WrapError(err, "unable to dequeue redis incoming job queue")
						log.ErrorWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
					}
				})
			}
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
				err = sdk.WrapError(err, "unable to check if queue still exist")
				log.ErrorWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
				continue
			} else if !b {
				// leave dequeue if queue does not exist anymore
				log.Info(ctx, "dequeueJobMessages: leaving job queue %s (queue no more exists)", jobLogsQueueKey)
				return nil
			}
			// heartbeat
			heartbeatKey := cache.Key(keyJobHearbeat, jobID)
			if err := s.Cache.SetWithTTL(heartbeatKey, true, 30); err != nil {
				err = sdk.WrapError(err, "unable to hearbeat %s", heartbeatKey)
				log.ErrorWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
				continue
			}
		default:
			dequeuCtx, cancel := context.WithTimeout(ctx, 1*time.Second)

			msgs, err := s.Cache.DequeueJSONRawMessagesWithContext(dequeuCtx, jobLogsQueueKey, 1*time.Millisecond, 1000)
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

					currentLog := buildMessage(hm)
					l := sdk.Log{
						JobID:        hm.Signature.JobID,
						NodeRunID:    hm.Signature.NodeRunID,
						LastModified: &now,
						StepOrder:    hm.Signature.Worker.StepOrder,
						Val:          currentLog,
					}
					if err := s.Client.QueueSendLogs(ctx, hm.Signature.JobID, l); err != nil {
						err = sdk.WrapError(err, "unable to send log to API")
						log.ErrorWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
						continue
					}
				}

				t1 = time.Now()

			} else if err != nil {
				if strings.Contains(err.Error(), "context deadline exceeded") {
					continue
				}
				err = sdk.WrapError(err, "unable to dequeue job logs queue %s", jobLogsQueueKey)
				log.ErrorWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
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
