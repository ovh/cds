package cdn

import (
	"bufio"
	"context"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	gocache "github.com/patrickmn/go-cache"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/log/hook"
	"github.com/ovh/cds/sdk/telemetry"
)

var (
	logCache                   = gocache.New(20*time.Minute, 30*time.Minute)
	keyJobLogIncomingQueue     = cache.Key("cdn", "log", "incoming", "job")
	keyServiceLogIncomingQueue = cache.Key("cdn", "log", "incoming", "service")
)

func (s *Service) RunTcpLogServer(ctx context.Context) {
	// Init hatcheries cache
	if err := s.refreshHatcheriesPK(ctx); err != nil {
		log.Error(ctx, "unable to init hatcheries cache: %v", err)
	}

	// Start TCP server
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

	for i := int64(0); i <= s.Cfg.NbJobLogsGoroutines; i++ {
		sdk.GoRoutine(ctx, "cdn-worker-job-"+string(i), func(ctx context.Context) {
			if err := s.dequeueJobLogs(ctx); err != nil {
				log.Error(ctx, "dequeueJobLogs: unable to dequeue redis incoming job logs: %v", err)
			}
		})
	}
	for i := int64(0); i < s.Cfg.NbServiceLogsGoroutines; i++ {
		sdk.GoRoutine(ctx, "cdn-worker-service-"+string(i), func(ctx context.Context) {
			if err := s.dequeueServiceLogs(ctx); err != nil {
				log.Error(ctx, "dequeueJobLogs: unable to dequeue redis incoming service logs: %v", err)
			}
		})
	}

	// Looking for something to dequeue
	// DEPRECATED
	sdk.GoRoutine(ctx, "cdn-waiting-job", func(ctx context.Context) {
		s.waitingJobs(ctx)
	})

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				telemetry.Record(ctx, Errors, 1)
				log.Error(ctx, "unable to accept connection: %v", err)
				return
			}
			sdk.GoRoutine(ctx, "cdn-logServer", func(ctx context.Context) {
				telemetry.Record(ctx, Hits, 1)
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
			telemetry.Record(ctx, Errors, 1)
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

	// Extract Signature
	sig, ok := m.Extra["_"+log.ExtraFieldSignature]
	if !ok || sig == "" {
		return sdk.WithStack(fmt.Errorf("signature not found on log message: %+v", m))
	}

	// Unsafe parse of signature to get datas
	var signature log.Signature
	if err := jws.UnsafeParse(sig.(string), &signature); err != nil {
		return err
	}

	switch {
	case signature.Worker != nil:
		telemetry.Record(ctx, WorkerLogReceived, 1)
		return s.handleWorkerLog(ctx, signature.Worker.WorkerName, signature.Worker.WorkerID, sig, m)
	case signature.Service != nil:
		telemetry.Record(ctx, ServiceLogReceived, 1)
		return s.handleServiceLog(ctx, signature.Service.HatcheryID, signature.Service.HatcheryName, signature.Service.WorkerName, sig, m)
	default:
		return sdk.WithStack(sdk.ErrWrongRequest)
	}
}

func (s *Service) handleWorkerLog(ctx context.Context, workerName string, workerID string, sig interface{}, m hook.Message) error {
	var signature log.Signature

	// Get worker data from cache
	workerData, err := s.getClearWorker(ctx, workerName)
	if err != nil {
		return err
	}

	// Verify Signature
	if err := jws.Verify(workerData.PrivateKey, sig.(string), &signature); err != nil {
		return err
	}
	if workerData.JobRunID == nil || *workerData.JobRunID != signature.JobID || workerData.ID != workerID {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	var line int64
	lineI := m.Extra["_"+log.ExtraFieldLine]
	if lineI != nil {
		line = int64(lineI.(float64))
	}

	var status string
	statusI := m.Extra["_"+log.ExtraFieldJobStatus]
	if statusI != nil {
		status = statusI.(string)
	}

	hm := handledMessage{
		Signature: signature,
		Msg:       m,
		Line:      line,
		Status:    status,
	}

	if s.cdnEnabled(ctx, signature.ProjectKey) {
		if err := s.Cache.Enqueue(keyJobLogIncomingQueue, hm); err != nil {
			return err
		}
	}

	// DEPRECATED - Save in queue for cds api call
	cacheKey := cache.Key(keyJobLogQueue, strconv.Itoa(int(signature.JobID)))
	if err := s.Cache.Enqueue(cacheKey, hm); err != nil {
		return err
	}
	///

	return nil
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

	// Get hatchery public key from cache
	cacheData, ok := logCache.Get(fmt.Sprintf("hatchery-key-%d", hatcheryID))
	if !ok {
		// Refresh hatcheries cache
		if err := s.refreshHatcheriesPK(ctx); err != nil {
			return err
		}
		cacheData, ok = logCache.Get(fmt.Sprintf("hatchery-key-%d", hatcheryID))
		if !ok {
			return sdk.WrapError(sdk.ErrForbidden, "unable to find hatchery %d/%s", hatcheryID, hatcheryName)
		}

	}
	pk = cacheData.(*rsa.PublicKey)

	// Verify signature
	if err := jws.Verify(pk, sig.(string), &signature); err != nil {
		return err
	}

	// Get worker + check hatchery ID
	w, err := s.getClearWorker(ctx, workerName)

	if err != nil {
		return err
	}
	if w.HatcheryID == nil {
		return sdk.WrapError(sdk.ErrWrongRequest, "hatchery %d cannot send service log for worker %s started by %s that is no more linked to an hatchery", signature.Service.HatcheryID, w.ID, w.HatcheryName)
	}
	if *w.HatcheryID != signature.Service.HatcheryID {
		return sdk.WrapError(sdk.ErrWrongRequest, "cannot send service log for worker %s from hatchery (expected: %d/actual: %d)", w.ID, *w.HatcheryID, signature.Service.HatcheryID)
	}

	var status string
	statusI := m.Extra["_"+log.ExtraFieldJobStatus]
	if statusI != nil {
		status = statusI.(string)
	}

	hm := handledMessage{
		Signature: signature,
		Msg:       m,
		Status:    status,
	}
	if s.cdnServiceLogsEnabled(ctx) {
		if err := s.Cache.Enqueue(keyServiceLogIncomingQueue, hm); err != nil {
			return err
		}
	}

	// DEPRECATED: call CDS API
	logs := sdk.ServiceLog{
		ServiceRequirementName: signature.Service.RequirementName,
		ServiceRequirementID:   signature.Service.RequirementID,
		WorkflowNodeJobRunID:   signature.JobID,
		WorkflowNodeRunID:      signature.NodeRunID,
		Val:                    m.Full,
	}
	if !strings.HasSuffix(logs.Val, "\n") {
		logs.Val += "\n"
	}
	if err := s.Client.QueueServiceLogs(ctx, []sdk.ServiceLog{logs}); err != nil {
		return err
	}
	///
	return nil
}

func (s *Service) getClearWorker(ctx context.Context, workerName string) (sdk.Worker, error) {
	workerKey := fmt.Sprintf("worker-%s", workerName)

	// Get worker from cache
	cacheData, ok := logCache.Get(workerKey)
	if ok {
		return cacheData.(sdk.Worker), nil
	}

	// Get worker from API
	w, err := s.Client.WorkerGet(ctx, workerName, cdsclient.WithQueryParameter("withKey", "true"))
	if err != nil {
		return sdk.Worker{}, err
	}
	publicKeyDecoded, err := base64.StdEncoding.DecodeString(string(w.PrivateKey))
	if err != nil {
		return sdk.Worker{}, sdk.WithStack(err)
	}
	w.PrivateKey = publicKeyDecoded
	logCache.Set(workerKey, *w, gocache.DefaultExpiration)

	return *w, nil
}

func (s *Service) refreshHatcheriesPK(ctx context.Context) error {
	srvs, err := s.Client.ServiceConfigurationGet(ctx, sdk.TypeHatchery)
	if err != nil {
		return sdk.WrapError(sdk.ErrNotFound, "unable to find hatcheries")
	}
	for _, s := range srvs {
		publicKey, err := base64.StdEncoding.DecodeString(s.PublicKey)
		if err != nil {
			return sdk.WithStack(err)
		}
		pk, err := jws.NewPublicKeyFromPEM(publicKey)
		if err != nil {
			return sdk.WithStack(err)
		}
		logCache.Set(fmt.Sprintf("hatchery-key-%d", s.ID), pk, gocache.DefaultExpiration)
	}
	return nil
}

func (s *Service) cdnServiceLogsEnabled(ctx context.Context) bool {
	cacheKey := "cdn-service-logs-enabled"
	enabled, has := logCache.Get(cacheKey)
	if !has {
		m := make(map[string]string)
		resp, err := s.Client.FeatureEnabled("cdn-service-logs", m)
		if err != nil {
			log.Error(ctx, "unable to get cdn-service features: %v", err)
			return false
		}
		logCache.Set(cacheKey, resp.Enabled, gocache.DefaultExpiration)
		return resp.Enabled
	}
	return enabled.(bool)
}

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
