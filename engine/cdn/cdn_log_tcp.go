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

	gocache "github.com/patrickmn/go-cache"
	"github.com/rockbears/log"
	"github.com/spf13/cast"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdn"
	"github.com/ovh/cds/sdk/jws"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/log/hook/graylog"
	"github.com/ovh/cds/sdk/telemetry"
)

var globalRateLimit *rateLimiter

// Reasons dimensioning the rejected log lines metric. A log line is either accepted (enqueued
// in the incoming queue) or rejected with one of those reasons: the sum must match the number
// of received lines.
const (
	rejectReasonUnmarshalError   = "unmarshal_error"
	rejectReasonSignatureInvalid = "signature_invalid"
	rejectReasonWorkerNotFound   = "worker_not_found"
	rejectReasonServiceNotFound  = "service_not_found"
	rejectReasonMismatch         = "mismatch"
	rejectReasonStepMaxSize      = "step_max_size"
	rejectReasonQueueError       = "queue_error"
	rejectReasonOther            = "other"
)

type GetWorkerOptions struct {
	NeedPrivateKey bool
}

// recordLogRejected counts a log line that will not be enqueued. The reason is recorded as a
// tag: cdn/tcp/errors alone cannot tell a bad signature from an unknown worker, which are two
// different incidents with two different fixes.
func (s *Service) recordLogRejected(ctx context.Context, reason string) {
	ctx = telemetry.ContextWithTag(ctx, telemetry.TagReason, reason)
	telemetry.Record(ctx, s.Metrics.tcpServerLogRejectedCount, 1)
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
			log.Error(sdk.ContextWithStacktrace(ctx, err), err.Error())
			continue
		}

		n, err := bufReader.Read(b)
		if err != nil {
			log.Debug(ctx, "client left: (%v) %v", conn.RemoteAddr(), err)
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
				log.Error(sdk.ContextWithStacktrace(ctx, err), err.Error())
				continue
			}
			if err := s.handleLogMessage(ctx, currentBuffer); err != nil {
				telemetry.Record(ctx, s.Metrics.tcpServerErrorsCount, 1)
				log.Error(sdk.ContextWithStacktrace(ctx, err), err.Error())
			}
			currentBuffer = make([]byte, 0)
		}
	}
}

// Handle Message: Worker/Hatchery
func (s *Service) handleLogMessage(ctx context.Context, messageReceived []byte) error {
	msg := graylog.Message{}
	if err := msg.UnmarshalJSON(messageReceived); err != nil {
		s.recordLogRejected(ctx, rejectReasonUnmarshalError)
		return sdk.WrapError(err, "unable to unmarshall gelf message: %s", string(messageReceived))
	}

	// Extract Signature
	sig, ok := msg.Extra["_"+cdslog.ExtraFieldSignature]
	if !ok || sig == "" {
		s.recordLogRejected(ctx, rejectReasonSignatureInvalid)
		return sdk.WithStack(fmt.Errorf("signature not found on log message: %+v", msg))
	}

	// Unsafe parse of signature to get datas
	var signature cdn.Signature
	if err := jws.UnsafeParse(sig.(string), &signature); err != nil {
		s.recordLogRejected(ctx, rejectReasonSignatureInvalid)
		return err
	}

	switch {
	case signature.Worker != nil:
		telemetry.Record(ctx, s.Metrics.tcpServerStepLogCount, 1)
		return s.handleWorkerLog(ctx, signature, sig, msg)
	case signature.Service != nil:
		telemetry.Record(ctx, s.Metrics.tcpServerServiceLogCount, 1)
		return s.handleServiceLog(ctx, signature, sig, msg)
	case signature.HatcheryService != nil:
		telemetry.Record(ctx, s.Metrics.tcpServerServiceLogCount, 1)
		return s.handleServiceLog(ctx, signature, sig, msg)
	default:
		s.recordLogRejected(ctx, rejectReasonOther)
		return sdk.WithStack(sdk.ErrWrongRequest)
	}
}

// Handle Message from worker (job logs). Enqueue in Redis
func (s *Service) handleWorkerLog(ctx context.Context, unsafeSign cdn.Signature, sig interface{}, msg graylog.Message) error {
	var signature cdn.Signature

	var jobID string
	switch {
	case unsafeSign.JobID != 0:
		workerData, err := s.verifyWorkerLog(ctx, unsafeSign, sig.(string), &signature)
		if err != nil {
			return err
		}
		if workerData.JobRunID == nil || *workerData.JobRunID != signature.JobID || workerData.ID != unsafeSign.Worker.WorkerID {
			s.recordLogRejected(ctx, rejectReasonMismatch)
			return sdk.WithStack(sdk.ErrForbidden)
		}
		jobID = strconv.Itoa(int(signature.JobID))
	case unsafeSign.RunJobID != "":
		workerData, err := s.verifyWorkerV2Log(ctx, unsafeSign, sig.(string), &signature)
		if err != nil {
			return err
		}
		if workerData.JobRunID == "" || workerData.JobRunID != signature.RunJobID || workerData.ID != unsafeSign.Worker.WorkerID {
			s.recordLogRejected(ctx, rejectReasonMismatch)
			return sdk.WithStack(sdk.ErrForbidden)
		}
		jobID = unsafeSign.RunJobID
	}

	terminatedI := msg.Extra["_"+cdslog.ExtraFieldTerminated]
	terminated := cast.ToBool(terminatedI)

	hm := handledMessage{
		Signature:    signature,
		Msg:          msg,
		IsTerminated: terminated,
	}

	sizeQueueKey := cache.Key(keyJobLogSize, jobID)
	jobQueue := cache.Key(keyJobLogQueue, jobID)

	if err := s.sendIntoIncomingQueue(ctx, hm, jobQueue, sizeQueueKey); err != nil {
		return err
	}
	return nil
}

// verifyWorkerLog is the v1 twin of verifyWorkerV2Log: the key of a v1 worker is cached under its
// name too, so a stale entry rejects every line of the job until it expires. Same strictly
// bounded recovery: evict, fetch once, verify once more.
func (s *Service) verifyWorkerLog(ctx context.Context, unsafeSign cdn.Signature, sig string, signature *cdn.Signature) (sdk.Worker, error) {
	workerData, err := s.getWorker(ctx, unsafeSign.Worker.WorkerName, GetWorkerOptions{NeedPrivateKey: true})
	if err != nil {
		s.recordLogRejected(ctx, rejectReasonWorkerNotFound)
		return sdk.Worker{}, err
	}

	verifyErr := jws.Verify(workerData.PrivateKey, sig, signature)
	if verifyErr == nil {
		return workerData, nil
	}

	refreshedWorkerData, refreshed, err := s.refreshWorkerKey(ctx, unsafeSign.Worker.WorkerName)
	if err != nil {
		s.recordLogRejected(ctx, rejectReasonWorkerNotFound)
		return sdk.Worker{}, err
	}
	if refreshed {
		workerData = refreshedWorkerData
		if verifyErr = jws.Verify(workerData.PrivateKey, sig, signature); verifyErr == nil {
			return workerData, nil
		}
	}

	s.recordLogRejected(ctx, rejectReasonSignatureInvalid)
	if workerData.ID != unsafeSign.Worker.WorkerID {
		return sdk.Worker{}, sdk.WrapError(verifyErr, "worker key: %d: signature worker id %s does not match worker %s id %s (name reused?)", len(workerData.PrivateKey), unsafeSign.Worker.WorkerID, workerData.Name, workerData.ID)
	}
	return sdk.Worker{}, sdk.WrapError(verifyErr, "worker key: %d", len(workerData.PrivateKey))
}

// verifyWorkerV2Log verifies the signature of a v2 worker log line against the key of the worker.
// That key is cached by worker NAME only: when a name is reused, the cached entry still holds the
// key of the previous worker and every line of the new job is rejected until the entry expires
// (20 minutes). So on a verification failure the entry is evicted, the key fetched again and the
// verification retried once. The refresh itself is rate limited per worker, otherwise a job whose
// every line fails verification would call the API once per line.
func (s *Service) verifyWorkerV2Log(ctx context.Context, unsafeSign cdn.Signature, sig string, signature *cdn.Signature) (sdk.V2Worker, error) {
	workerData, err := s.getWorkerV2(ctx, unsafeSign.Worker.WorkerName, GetWorkerOptions{NeedPrivateKey: true})
	if err != nil {
		s.recordLogRejected(ctx, rejectReasonWorkerNotFound)
		return sdk.V2Worker{}, err
	}

	verifyErr := jws.Verify(workerData.PrivateKey, sig, signature)
	if verifyErr == nil {
		return workerData, nil
	}

	refreshedWorkerData, refreshed, err := s.refreshWorkerV2Key(ctx, unsafeSign.Worker.WorkerName)
	if err != nil {
		s.recordLogRejected(ctx, rejectReasonWorkerNotFound)
		return sdk.V2Worker{}, err
	}
	if refreshed {
		workerData = refreshedWorkerData
		if verifyErr = jws.Verify(workerData.PrivateKey, sig, signature); verifyErr == nil {
			return workerData, nil
		}
	}

	s.recordLogRejected(ctx, rejectReasonSignatureInvalid)
	if workerData.ID != unsafeSign.Worker.WorkerID {
		return sdk.V2Worker{}, sdk.WrapError(verifyErr, "worker key: %d: signature worker id %s does not match worker %s id %s (name reused?)", len(workerData.PrivateKey), unsafeSign.Worker.WorkerID, workerData.Name, workerData.ID)
	}
	return sdk.V2Worker{}, sdk.WrapError(verifyErr, "worker key: %d", len(workerData.PrivateKey))
}

func (s *Service) sendIntoIncomingQueue(ctx context.Context, hm handledMessage, incomingQueue string, sizeKey string) error {
	var currentSize int64
	if _, err := s.Cache.Get(sizeKey, &currentSize); err != nil {
		s.recordLogRejected(ctx, rejectReasonQueueError)
		return err
	}
	if currentSize >= s.Cfg.Log.StepMaxSize && !hm.IsTerminated {
		s.recordLogRejected(ctx, rejectReasonStepMaxSize)
		return nil
	}
	if currentSize >= s.Cfg.Log.StepMaxSize && hm.IsTerminated {
		hm.Msg.Full = "...truncated\n"
		hm.Msg.Level = int32(graylog.LOG_WARNING)
	}

	if err := s.Cache.Enqueue(incomingQueue, hm); err != nil {
		s.recordLogRejected(ctx, rejectReasonQueueError)
		return err
	}
	telemetry.Record(ctx, s.Metrics.tcpServerLogAcceptedCount, 1)

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
	return val
}

func (s *Service) handleServiceLog(ctx context.Context, unsafeSign cdn.Signature, sig interface{}, msg graylog.Message) error {
	var signature cdn.Signature
	var pk *rsa.PublicKey

	var hatcheryID, hatcheryName, serviceName string
	if unsafeSign.JobID != 0 {
		hatcheryID = strconv.FormatInt(unsafeSign.Service.HatcheryID, 10)
		hatcheryName = unsafeSign.Service.HatcheryName
		serviceName = unsafeSign.Service.RequirementName
	} else if unsafeSign.RunJobID != "" {
		hatcheryID = unsafeSign.HatcheryService.HatcheryID
		hatcheryName = unsafeSign.HatcheryService.HatcheryName
		serviceName = unsafeSign.HatcheryService.ServiceName
	} else {
		s.recordLogRejected(ctx, rejectReasonSignatureInvalid)
		return sdk.WrapError(sdk.ErrForbidden, "invalid signature %v", unsafeSign)
	}

	// Get hatchery public key from cache
	cacheData, ok := runCache.Get(fmt.Sprintf("hatchery-key-%s", hatcheryID))
	if !ok {
		// Refresh hatcheries cache
		if err := s.refreshHatcheriesPK(ctx); err != nil {
			s.recordLogRejected(ctx, rejectReasonServiceNotFound)
			return err
		}
		cacheData, ok = runCache.Get(fmt.Sprintf("hatchery-key-%s", hatcheryID))
		if !ok {
			s.recordLogRejected(ctx, rejectReasonServiceNotFound)
			return sdk.WrapError(sdk.ErrForbidden, "unable to find hatchery %s/%s", hatcheryID, hatcheryName)
		}
	}
	pk = cacheData.(*rsa.PublicKey)

	// Verify signature
	if err := jws.Verify(pk, sig.(string), &signature); err != nil {
		s.recordLogRejected(ctx, rejectReasonSignatureInvalid)
		return err
	}

	var key string
	switch {
	case signature.JobID != 0:
		// Get worker + check hatchery ID
		w, err := s.getWorker(ctx, signature.Service.WorkerName, GetWorkerOptions{NeedPrivateKey: false})
		if err != nil {
			s.recordLogRejected(ctx, rejectReasonWorkerNotFound)
			return err
		}
		if w.HatcheryID == nil {
			s.recordLogRejected(ctx, rejectReasonMismatch)
			return sdk.WrapError(sdk.ErrWrongRequest, "hatchery %d cannot send service log for worker %s started by %s that is no more linked to an hatchery", signature.Service.HatcheryID, w.ID, w.HatcheryName)
		}
		if *w.HatcheryID != signature.Service.HatcheryID {
			s.recordLogRejected(ctx, rejectReasonMismatch)
			return sdk.WrapError(sdk.ErrWrongRequest, "cannot send service log (%s) for worker %s from hatchery (expected: %d / actual: %d)", serviceName, w.ID, *w.HatcheryID, signature.Service.HatcheryID)
		}

		key = fmt.Sprintf("%d-%d", signature.JobID, signature.Service.RequirementID)

	case signature.RunJobID != "":
		key = fmt.Sprintf("%s-%s", signature.RunJobID, signature.HatcheryService.ServiceName)
	}

	terminatedI := msg.Extra["_"+cdslog.ExtraFieldTerminated]
	terminated := cast.ToBool(terminatedI)

	hm := handledMessage{
		Signature:    signature,
		Msg:          msg,
		IsTerminated: terminated,
	}

	sizeQueueKey := cache.Key(keyJobLogSize, key)
	jobQueue := cache.Key(keyJobLogQueue, key)

	if err := s.sendIntoIncomingQueue(ctx, hm, jobQueue, sizeQueueKey); err != nil {
		return err
	}
	return nil
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
		runCache.Set(fmt.Sprintf("hatchery-key-%d", s.ID), pk, gocache.DefaultExpiration)
	}

	// Load hatchery
	hatcheries, err := s.Client.HatcheryList(ctx)
	if err != nil {
		return sdk.WrapError(sdk.ErrNotFound, "unable to find hatcheries")
	}
	for _, h := range hatcheries {
		pk, err := jws.NewPublicKeyFromPEM(h.PublicKey)
		if err != nil {
			return sdk.WithStack(err)
		}
		runCache.Set(fmt.Sprintf("hatchery-key-%s", h.ID), pk, gocache.DefaultExpiration)
	}
	return nil
}
