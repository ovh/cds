package cdn

import (
	"bufio"
	"context"
	"crypto/rsa"
	"fmt"
	"net"
	"strings"
	"time"

	gocache "github.com/patrickmn/go-cache"

	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/log/hook"
)

var (
	logCache = gocache.New(20*time.Minute, 30*time.Minute)
)

func (s *Service) RunTcpLogServer(ctx context.Context) {
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
				log.Error(ctx, "unable to accept connection: %v", err)
				return
			}
			sdk.GoRoutine(ctx, "cdn-logServer", func(ctx context.Context) {
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
		return s.handleWorkerLog(ctx, signature.Worker.WorkerID, sig, m)
	case signature.Service != nil:
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

	pbJob, err := workflow.LoadNodeJobRun(ctx, s.Db, s.Cache, signature.JobID)
	if err != nil {
		return err
	}

	logDate := time.Unix(0, int64(m.Time*1e9))
	logs := sdk.Log{
		JobID:        pbJob.ID,
		LastModified: &logDate,
		NodeRunID:    pbJob.WorkflowNodeRunID,
		Start:        &logDate,
		StepOrder:    signature.Worker.StepOrder,
		Val:          m.Full,
	}
	if !strings.HasSuffix(logs.Val, "\n") {
		logs.Val += "\n"
	}

	tx, err := s.Db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	if err := workflow.AddLog(tx, pbJob, &logs, s.Cfg.Log.StepMaxSize); err != nil {
		return err
	}
	return sdk.WithStack(tx.Commit())
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
		if w.HatcheryID != signature.Service.HatcheryID {
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
	logCache.Set(fmt.Sprintf("worker-%s", w.ID), w, gocache.DefaultExpiration)
	return w, nil
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
