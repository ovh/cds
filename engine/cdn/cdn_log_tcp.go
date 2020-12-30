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
	"github.com/spf13/cast"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/log/hook"
	"github.com/ovh/cds/sdk/telemetry"
)

var (
	logCache = gocache.New(20*time.Minute, 20*time.Minute)
)

var globalRateLimit *rateLimiter

type GetWorkerOptions struct {
	NeedPrivateKey bool
}

// Start TCP Server
func (s *Service) runTCPLogServer(ctx context.Context) error {
	globalRateLimit = NewRateLimiter(ctx, float64(s.Cfg.TCP.GlobalTCPRateLimit), 1024)

	// Start TCP server
	log.Info(ctx, "Starting tcp server %s:%d", s.Cfg.TCP.Addr, s.Cfg.TCP.Port)
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.Cfg.TCP.Addr, s.Cfg.TCP.Port))
	if err != nil {
		return sdk.WrapError(err, "unable to start tcp log server")
	}

	//Gracefully shutdown the tcp server
	s.GoRoutines.Run(ctx, "service.runTCPLogServer.shutdown", func(ctx context.Context) {
		<-ctx.Done()
		log.Info(ctx, "CDN> Shutdown tcp log Server")
		_ = listener.Close()
	})

	// Looking for something to dequeue
	s.GoRoutines.Run(ctx, "cdn-waiting-job", func(ctx context.Context) {
		s.waitingJobs(ctx)
	})

	s.GoRoutines.Run(ctx, "service.runTCPLogServer.accept", func(ctx context.Context) {
		for {
			conn, err := listener.Accept()
			if err != nil {
				telemetry.Record(ctx, s.Metrics.tcpServerErrorsCount, 1)
				log.Error(ctx, "unable to accept connection: %v", err)
				return
			}
			s.GoRoutines.Exec(ctx, "cdn-logServer", func(ctx context.Context) {
				telemetry.Record(ctx, s.Metrics.tcpServerHitsCount, 1)
				s.handleConnection(ctx, conn)
			})
		}
	})

	return nil
}

// Handle TCP Connection: Global Rate Limit + Line Rate Limit
func (s *Service) handleConnection(ctx context.Context, conn net.Conn) {
	defer func() {
		_ = conn.Close()
	}()

	lineRateLimiter := NewRateLimiter(ctx, float64(s.Cfg.Log.StepLinesRateLimit), 1)

	bufReader := bufio.NewReader(conn)

	b := make([]byte, 1024)
	currentBuffer := make([]byte, 0)
	for {
		// Can i try to read the next 1024B
		if err := globalRateLimit.WaitN(1024); err != nil {
			fields := log.Fields{}
			fields["stack_trace"] = fmt.Sprintf("%+v", err)
			log.ErrorWithFields(ctx, fields, "cdn.log> %v", err)
			continue
		}

		n, err := bufReader.Read(b)
		if err != nil {
			log.Debug("client left: (%v) %v", conn.RemoteAddr(), err)
			return
		}

		// Search for end of line separator
		for i := 0; i < n; i++ {
			if b[i] != byte(0) {
				currentBuffer = append(currentBuffer, b[i])
				continue
			}

			// Check if we can send line
			if err := lineRateLimiter.WaitN(1); err != nil {
				fields := log.Fields{}
				fields["stack_trace"] = fmt.Sprintf("%+v", err)
				log.ErrorWithFields(ctx, fields, "cdn.log> %v", err)
				continue
			}
			if err := s.handleLogMessage(ctx, currentBuffer); err != nil {
				telemetry.Record(ctx, s.Metrics.tcpServerErrorsCount, 1)
				isErrWithStack := sdk.IsErrorWithStack(err)
				fields := log.Fields{}
				if isErrWithStack {
					fields["stack_trace"] = fmt.Sprintf("%+v", err)
				}
				log.ErrorWithFields(ctx, fields, "cdn.log> %v", err)
			}
			currentBuffer = make([]byte, 0)
		}
	}
}

// Handle Message: Worker/Hatchery
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
		telemetry.Record(ctx, s.Metrics.tcpServerStepLogCount, 1)
		return s.handleWorkerLog(ctx, signature.Worker.WorkerName, signature.Worker.WorkerID, sig, m)
	case signature.Service != nil:
		telemetry.Record(ctx, s.Metrics.tcpServerServiceLogCount, 1)
		return s.handleServiceLog(ctx, signature.Service.HatcheryID, signature.Service.HatcheryName, signature.Service.WorkerName, sig, m)
	default:
		return sdk.WithStack(sdk.ErrWrongRequest)
	}
}

// Handle Message from worker (job logs). Enqueue in Redis
func (s *Service) handleWorkerLog(ctx context.Context, workerName string, workerID string, sig interface{}, m hook.Message) error {
	var signature log.Signature

	// Get worker data from cache
	workerData, err := s.getWorker(ctx, workerName, GetWorkerOptions{NeedPrivateKey: true})
	if err != nil {
		return err
	}

	// Verify Signature
	if err := jws.Verify(workerData.PrivateKey, sig.(string), &signature); err != nil {
		return sdk.WrapError(err, "worker key: %d", len(workerData.PrivateKey))
	}
	if workerData.JobRunID == nil || *workerData.JobRunID != signature.JobID || workerData.ID != workerID {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	terminatedI := m.Extra["_"+log.ExtraFieldTerminated]
	terminated := cast.ToBool(terminatedI)

	hm := handledMessage{
		Signature:    signature,
		Msg:          m,
		IsTerminated: terminated,
	}

	sizeQueueKey := cache.Key(keyJobLogSize, strconv.Itoa(int(hm.Signature.JobID)))
	jobQueue := cache.Key(keyJobLogQueue, strconv.Itoa(int(hm.Signature.JobID)))

	if err := s.sendIntoIncomingQueue(hm, jobQueue, sizeQueueKey); err != nil {
		return err
	}
	return nil
}

func (s *Service) sendIntoIncomingQueue(hm handledMessage, incomingQueue string, sizeKey string) error {
	var currentSize int64
	if _, err := s.Cache.Get(sizeKey, &currentSize); err != nil {
		return err
	}
	if currentSize >= s.Cfg.Log.StepMaxSize && !hm.IsTerminated {
		return nil
	}
	if currentSize >= s.Cfg.Log.StepMaxSize && hm.IsTerminated {
		hm.Msg.Full = "...truncated\n"
		hm.Msg.Level = int32(hook.LOG_WARNING)
	}

	if err := s.Cache.Enqueue(incomingQueue, hm); err != nil {
		return err
	}

	if hm.IsTerminated {
		_ = s.Cache.Delete(sizeKey)
	} else {
		// Update size for the job
		newSize := currentSize + int64(len(hm.Msg.Full))
		if err := s.Cache.SetWithTTL(sizeKey, newSize, 3600*24); err != nil {
			return err
		}
	}
	return nil
}

func buildMessage(hm handledMessage) string {
	val := hm.Msg.Full
	if !strings.HasSuffix(val, "\n") {
		val += "\n"
	}
	return fmt.Sprintf("[%s] %s", getLevelString(hm.Msg.Level), val)
}

func getLevelString(level int32) string {
	var lvl string
	switch level {
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
	return lvl
}

//
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
	w, err := s.getWorker(ctx, workerName, GetWorkerOptions{NeedPrivateKey: false})
	if err != nil {
		return err
	}
	if w.HatcheryID == nil {
		return sdk.WrapError(sdk.ErrWrongRequest, "hatchery %d cannot send service log for worker %s started by %s that is no more linked to an hatchery", signature.Service.HatcheryID, w.ID, w.HatcheryName)
	}
	if *w.HatcheryID != signature.Service.HatcheryID {
		return sdk.WrapError(sdk.ErrWrongRequest, "cannot send service log for worker %s from hatchery (expected: %d/actual: %d)", w.ID, *w.HatcheryID, signature.Service.HatcheryID)
	}

	terminatedI := m.Extra["_"+log.ExtraFieldTerminated]
	terminated := cast.ToBool(terminatedI)

	hm := handledMessage{
		Signature:    signature,
		Msg:          m,
		IsTerminated: terminated,
	}

	reqKey := fmt.Sprintf("%d-%d", signature.JobID, signature.Service.RequirementID)
	sizeQueueKey := cache.Key(keyJobLogSize, reqKey)
	jobQueue := cache.Key(keyJobLogQueue, reqKey)

	if err := s.sendIntoIncomingQueue(hm, jobQueue, sizeQueueKey); err != nil {
		return err
	}
	return nil
}

func (s *Service) getWorker(ctx context.Context, workerName string, opts GetWorkerOptions) (sdk.Worker, error) {
	workerKey := fmt.Sprintf("worker-%s", workerName)

	// Get worker from cache
	cacheData, ok := logCache.Get(workerKey)
	if ok {
		w, ok := cacheData.(sdk.Worker)
		if ok && (!opts.NeedPrivateKey || len(w.PrivateKey) > 0) {
			return w, nil
		}
	}

	// Get worker from API
	w, err := s.Client.WorkerGet(ctx, workerName, cdsclient.WithQueryParameter("withKey", "true"))
	if err != nil {
		return sdk.Worker{}, sdk.WrapError(err, "unable to get worker %s", workerName)
	}
	privateKeyDecoded, err := base64.StdEncoding.DecodeString(string(w.PrivateKey))
	if err != nil {
		return sdk.Worker{}, sdk.WithStack(err)
	}
	w.PrivateKey = privateKeyDecoded
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
