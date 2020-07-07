package cdn

import (
	"bufio"
	"context"
	"crypto/rsa"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	gocache "github.com/patrickmn/go-cache"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/log/hook"
)

var (
	logCache       = gocache.New(20*time.Minute, 30*time.Minute)
	keyJobLogQueue = cache.Key("cdn", "log", "job")
	keyJobHearbeat = cache.Key("cdn", "log", "heartbeat")
	keyJobLock     = cache.Key("cdn", "log", "lock")
)

func (s *Service) RunTcpLogServer(ctx context.Context) {
	log.Info(ctx, "Starting tcp server %s:%d", s.Cfg.TCP.Addr, s.Cfg.TCP.Port)
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.Cfg.TCP.Addr, s.Cfg.TCP.Port))
	if err != nil {
		log.Fatalf("unable to start tcp log server: %v", err)
	}

	//Gracefully shutdown the tcp server
	go func() {
		<-ctx.Done()
		log.Info(ctx, "CDN> Shutdown tcp log Server")
		_ = listener.Close()
	}()

	//  Looking for something to dequeue
	sdk.GoRoutine(ctx, "cdn-waiting-job", func(ctx context.Context) {
		s.waitingJobs(ctx)
	})

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				observability.Record(ctx, Errors, 1)
				log.Error(ctx, "unable to accept connection: %v", err)
				return
			}
			sdk.GoRoutine(ctx, "cdn-logServer", func(ctx context.Context) {
				observability.Record(ctx, Hits, 1)
				s.handleConnection(ctx, conn)
			})
		}
	}()
}

func (s *Service) handleConnection(ctx context.Context, conn net.Conn) {
	defer func() {
		_ = conn.Close()
	}()

	bufReader := bufio.NewReader(conn)
	for {
		bytes, err := bufReader.ReadBytes(byte(0))
		if err != nil {
			log.Info(ctx, "client left")
			return
		}
		// remove byte(0)
		bytes = bytes[:len(bytes)-1]

		if err := s.handleLogMessage(ctx, bytes); err != nil {
			observability.Record(ctx, Errors, 1)
			log.Error(ctx, "cdn.log> %v", err)
			continue
		}
	}
}

func (s *Service) handleLogMessage(ctx context.Context, messageReceived []byte) error {
	m := hook.Message{}
	if err := m.UnmarshalJSON(messageReceived); err != nil {
		return sdk.WrapError(err, "unable to unmarshall gelf message: %s", string(messageReceived))
	}

	sig, ok := m.Extra["_"+log.ExtraFieldSignature]
	if !ok || sig == "" {
		return sdk.WithStack(fmt.Errorf("signature not found on log message: %+v", m))
	}

	// Get worker datas
	var signature log.Signature
	if err := jws.UnsafeParse(sig.(string), &signature); err != nil {
		return err
	}

	switch {
	case signature.Worker != nil:
		observability.Record(ctx, WorkerLogReceived, 1)
		return s.handleWorkerLog(ctx, signature.Worker.WorkerID, sig, m)
	case signature.Service != nil:
		observability.Record(ctx, ServiceLogReceived, 1)
		return s.handleServiceLog(ctx, signature.Service.HatcheryID, signature.Service.HatcheryName, signature.Service.WorkerName, sig, m)
	default:
		return sdk.WithStack(sdk.ErrWrongRequest)
	}
}

func (s *Service) handleWorkerLog(ctx context.Context, workerID string, sig interface{}, m hook.Message) error {
	var signature log.Signature
	var workerData sdk.Worker
	cacheData, ok := logCache.Get(fmt.Sprintf("worker-%s", workerID))
	if !ok {
		var err error
		workerData, err = s.getWorker(ctx, workerID)
		if err != nil {
			return err
		}
	} else {
		workerData = cacheData.(sdk.Worker)
	}
	if err := jws.Verify(workerData.PrivateKey, sig.(string), &signature); err != nil {
		return err
	}
	if workerData.JobRunID == nil || *workerData.JobRunID != signature.JobID {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	hm := handledMessage{
		Signature: signature,
		Msg:       m,
	}
	cacheKey := cache.Key(keyJobLogQueue, strconv.Itoa(int(signature.JobID)))
	if err := s.Cache.Enqueue(cacheKey, hm); err != nil {
		return err
	}
	return nil
}

type handledMessage struct {
	Signature log.Signature
	Msg       hook.Message
}

func buildMessage(signature log.Signature, m hook.Message) string {
	logDate := time.Unix(0, int64(m.Time*1e9))
	logs := sdk.Log{
		JobID:        signature.JobID,
		LastModified: &logDate,
		NodeRunID:    signature.NodeRunID,
		Start:        &logDate,
		StepOrder:    signature.Worker.StepOrder,
		Val:          m.Full,
	}
	if !strings.HasSuffix(logs.Val, "\n") {
		logs.Val += "\n"
	}
	var lvl string
	switch m.Level {
	case int32(hook.LOG_DEBUG):
		lvl = "DEBUG"
	case int32(hook.LOG_INFO):
		lvl = "INFO"
	case int32(hook.LOG_NOTICE):
		lvl = "NOTICE"
	case int32(hook.LOG_WARNING):
		lvl = "WARN"
	case int32(hook.LOG_ERR):
		lvl = "ERROR"
	case int32(hook.LOG_CRIT):
		lvl = "CRITICAL"
	case int32(hook.LOG_ALERT):
		lvl = "ALERT"
	case int32(hook.LOG_EMERG):
		lvl = "EMERGENCY"
	}
	logs.Val = fmt.Sprintf("[%s] %s", lvl, logs.Val)
	return logs.Val
}

func (s *Service) handleServiceLog(ctx context.Context, hatcheryID int64, hatcheryName string, workerName string, sig interface{}, m hook.Message) error {
	var signature log.Signature

	var pk *rsa.PublicKey
	cacheData, ok := logCache.Get(fmt.Sprintf("hatchery-key-%d", hatcheryID))
	if !ok {
		var err error
		pk, err = s.getHatchery(ctx, hatcheryID, hatcheryName)
		if err != nil {
			return err
		}
	} else {
		pk = cacheData.(*rsa.PublicKey)
	}

	if err := jws.Verify(pk, sig.(string), &signature); err != nil {
		return err
	}

	// Verified that worker has been spawn by this hatchery
	workerCacheKey := fmt.Sprintf("service-worker-%s", workerName)
	_, ok = logCache.Get(workerCacheKey)
	if !ok {
		// Verify that the worker has been spawn by this hatchery
		w, err := worker.LoadWorkerByName(ctx, s.Db, workerName)
		if err != nil {
			return err
		}
		if w.HatcheryID != nil && *w.HatcheryID != signature.Service.HatcheryID {
			return sdk.WrapError(sdk.ErrWrongRequest, "hatchery and worker does not match")
		}
		logCache.Set(workerCacheKey, true, gocache.DefaultExpiration)
	}

	nodeRunJob, err := workflow.LoadNodeJobRun(ctx, s.Db, s.Cache, signature.JobID)
	if err != nil {
		return err
	}

	logs := sdk.ServiceLog{
		ServiceRequirementName: signature.Service.RequirementName,
		ServiceRequirementID:   signature.Service.RequirementID,
		WorkflowNodeJobRunID:   signature.JobID,
		WorkflowNodeRunID:      nodeRunJob.WorkflowNodeRunID,
		Val:                    m.Full,
	}
	if !strings.HasSuffix(logs.Val, "\n") {
		logs.Val += "\n"
	}

	if err := workflow.AddServiceLog(s.Db, nodeRunJob, &logs, s.Cfg.Log.ServiceMaxSize); err != nil {
		return err
	}
	return nil
}

func (s *Service) getWorker(ctx context.Context, workerID string) (sdk.Worker, error) {
	w, err := worker.LoadWorkerByIDWithDecryptKey(ctx, s.Db, workerID)
	if err != nil {
		return sdk.Worker{}, err
	}
	logCache.Set(fmt.Sprintf("worker-%s", w.ID), *w, gocache.DefaultExpiration)
	return *w, nil
}

func (s *Service) getHatchery(ctx context.Context, hatcheryID int64, hatcheryName string) (*rsa.PublicKey, error) {
	h, err := services.LoadByNameAndType(ctx, s.Db, hatcheryName, services.TypeHatchery)
	if err != nil {
		return nil, err
	}

	if h.ID != hatcheryID {
		return nil, sdk.WithStack(sdk.ErrWrongRequest)
	}

	// Verify signature
	pk, err := jws.NewPublicKeyFromPEM(h.PublicKey)
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	logCache.Set(fmt.Sprintf("hatchery-key-%d", hatcheryID), pk, gocache.DefaultExpiration)
	return pk, nil
}

func (s *Service) waitingJobs(ctx context.Context) {
	tick := time.NewTicker(250 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case _ = <-tick.C:
			// List all queues
			keyListQueue := cache.Key(keyJobLogQueue, "*")
			listKeys, err := s.Cache.Keys(keyListQueue)
			if err != nil {
				log.Error(ctx, "unable to list jobs queues %s", keyListQueue)
				continue
			}

			// For each key, check if heartbeat key exist
			for _, k := range listKeys {
				keyParts := strings.Split(k, ":")
				jobID := keyParts[len(keyParts)-1]

				jobQueueKey, err := s.canDequeue(jobID)
				if err != nil {
					log.Error(ctx, "unable to check canDequeue %s: %v", jobQueueKey, err)
					continue
				}
				if jobQueueKey == "" {
					continue
				}

				sdk.GoRoutine(ctx, "cdn-dequeue-job-message", func(ctx context.Context) {
					if err := s.dequeueJobMessages(ctx, jobQueueKey, jobID); err != nil {
						log.Error(ctx, "unable to dequeue redis incoming job queue: %v", err)
					}
				})
			}
		}
	}
}

func (s *Service) dequeueJobMessages(ctx context.Context, jobLogsQueueKey string, jobID string) error {
	log.Info(ctx, "Dequeue %s", jobLogsQueueKey)
	var t0 = time.Now()
	var t1 = time.Now()
	var nbMessages int
	defer func() {
		delta := t1.Sub(t0)
		log.Info(ctx, "processLogs[%s] - %d messages received in %v", jobLogsQueueKey, nbMessages, delta)
	}()

	defer func() {
		// Remove heartbeat
		_ = s.Cache.Delete(cache.Key(keyJobHearbeat, jobID))
	}()

	tick := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			b, err := s.Cache.Exist(jobLogsQueueKey)
			if err != nil {
				log.Error(ctx, "unable to check if queue still exist: %v", err)
				continue
			} else if !b {
				// leave dequeue if queue does not exist anymore
				log.Info(ctx, "leaving job queue %s (queue no more exists)", jobLogsQueueKey)
				return nil
			}
			// heartbeat
			heartbeatKey := cache.Key(keyJobHearbeat, jobID)
			if err := s.Cache.SetWithTTL(heartbeatKey, true, 30); err != nil {
				log.Error(ctx, "unable to hearbeat %s: %v", heartbeatKey, err)
				continue
			}
		default:
			dequeuCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			var hm handledMessage
			if err := s.Cache.DequeueWithContext(dequeuCtx, jobLogsQueueKey, 30*time.Millisecond, &hm); err != nil {
				cancel()
				if strings.Contains(err.Error(), "context deadline exceeded") {
					return nil
				}
				log.Error(ctx, "unable to dequeue job logs queue %s: %v", jobLogsQueueKey, err)
				continue
			}
			cancel()
			if hm.Signature.Worker == nil {
				continue
			}
			nbMessages++
			t1 = time.Now()

			currentLog := buildMessage(hm.Signature, hm.Msg)
			if err := workflow.AppendLog(s.Db, hm.Signature.JobID, hm.Signature.NodeRunID, hm.Signature.Worker.StepOrder, currentLog, s.Cfg.Log.StepMaxSize); err != nil {
				log.Error(ctx, "unable to process log: %+v", err)
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
