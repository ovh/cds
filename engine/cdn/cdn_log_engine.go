package cdn

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	gocache "github.com/patrickmn/go-cache"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var (
	keyJobLogQueue = cache.Key("cdn", "log", "job")
	keyJobLogLines = cache.Key("cdn", "log", "lines")
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
				log.ErrorWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
				continue
			}

			// For each key, check if heartbeat key exist
			for _, k := range listKeys {
				keyParts := strings.Split(k, ":")
				queueIdentifier := keyParts[len(keyParts)-1]

				jobQueueKey, err := s.canDequeue(ctx, queueIdentifier)
				if err != nil {
					err = sdk.WrapError(err, "unable to check canDequeue %s", jobQueueKey)
					log.ErrorWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
					continue
				}
				if jobQueueKey == "" {
					continue
				}

				s.GoRoutines.Exec(ctx, "cdn-dequeue-job-message", func(ctx context.Context) {
					if err := s.dequeueMessages(ctx, jobQueueKey, queueIdentifier); err != nil {
						err = sdk.WrapError(err, "unable to dequeue redis incoming job queue")
						log.ErrorWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
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
				log.ErrorWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
				continue
			} else if !b {
				// leave dequeue if queue does not exist anymore
				log.Info(ctx, "dequeueJobMessages: leaving job queue %s (queue no more exists)", jobLogsQueueKey)
				return nil
			}
			// heartbeat
			heartbeatKey := cache.Key(keyJobHearbeat, queueIdentifier)
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
				hms := make([]handledMessage, 0, len(msgs))
				for _, m := range msgs {
					var hm handledMessage
					if err := json.Unmarshal(m, &hm); err != nil {
						return sdk.WithStack(err)
					}
					hms = append(hms, hm)
				}

				// Send TO CDS API
				if err := s.sendToCDS(ctx, hms); err != nil {
					err = sdk.WrapError(err, "unable to send log to API")
					log.ErrorWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
				}

				// Send TO CDN Buffer
				if s.cdnEnabled(ctx, hms[0].Signature.ProjectKey) {
					if err := s.sendToBufferWithRetry(ctx, hms); err != nil {
						err = sdk.WrapError(err, "unable to send log into buffer")
						log.ErrorWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
					}
				}
				nbMessages += len(msgs)
				t1 = time.Now()
			}
			if err != nil {
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

// Check if storage on CDN is enabled
func (s *Service) cdnEnabled(ctx context.Context, projectKey string) bool {
	cacheKey := fmt.Sprintf("cdn-job-logs-enabled-project-%s", projectKey)
	enabled, has := logCache.Get(cacheKey)
	if !has {
		m := make(map[string]string, 1)
		m["project_key"] = projectKey
		resp, err := s.Client.FeatureEnabled("cdn-job-logs", m)
		if err != nil {
			log.Error(ctx, "unable to get job logs feature for project %s: %v", projectKey, err)
			return false
		}
		logCache.Set(cacheKey, resp.Enabled, gocache.DefaultExpiration)
		return resp.Enabled
	}
	return enabled.(bool)
}
