package cdn

import (
	"bufio"
	"context"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"time"

	gocache "github.com/patrickmn/go-cache"

	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/log/hook"
)

var (
	logCache = gocache.New(20*time.Minute, 30*time.Minute)
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
		select {
		case <-ctx.Done():
			log.Info(ctx, "CDN> Shutdown tcp log Server")
			_ = listener.Close()
		}
	}()

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
	// Get Log Message
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
		observability.Record(ctx, WorkerLogReceived, 1)
		return s.handleWorkerLog(ctx, signature.Worker.WorkerID, signature.Worker.WorkerName, sig, m)
	case signature.Service != nil:
		observability.Record(ctx, ServiceLogReceived, 1)
		return s.handleServiceLog(ctx, signature.Service.HatcheryID, signature.Service.HatcheryName, signature.Service.WorkerName, sig, m)
	default:
		return sdk.WithStack(sdk.ErrWrongRequest)
	}
}

func (s *Service) handleWorkerLog(ctx context.Context, workerID string, workerName string, sig interface{}, m hook.Message) error {
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

	// Send log to API
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
	if err := s.Client.QueueSendLogs(ctx, signature.JobID, logs); err != nil {
		return err
	}
	return nil
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
	if w.HatcheryID != signature.Service.HatcheryID {
		return sdk.WrapError(sdk.ErrWrongRequest, "hatchery and worker does not match")
	}

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
	w, err := s.Client.WorkerGet(ctx, workerName)
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
	//s.Client.
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
