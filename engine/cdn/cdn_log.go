package cdn

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"time"

	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/log/hook"
)

var (
	workers = make(map[string]sdk.Worker)
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
			go s.handleConnection(ctx, conn)
		}
	}()
}

func (s *Service) handleConnection(ctx context.Context, conn net.Conn) {
	defer func() {
		_ = conn.Close()
	}()

	timeoutDuration := 5 * time.Second
	bufReader := bufio.NewReader(conn)

	for {
		// Set a deadline for reading. Read operation will fail if no data is received after deadline.
		if err := conn.SetReadDeadline(time.Now().Add(timeoutDuration)); err != nil {
			log.Error(ctx, "unable to set read deadline on connection")
			return
		}
		bytes, err := bufReader.ReadBytes(byte(0))
		if err != nil {
			log.Info(ctx, "client left")
			return
		}
		log.Warning(ctx, "Message IN CDNNNNNNNNNN")
		var m *hook.Message
		if err := m.UnmarshalJSON(bytes); err != nil {
			log.Error(ctx, "cdn.log > unable to unmarshal log message: %v", err)
			continue
		}

		sig, ok := m.Extra[log.ExtraFieldSignature]
		if !ok || sig == "" {
			log.Error(ctx, "cdn.log > signature not found on log message %+v", m)
			continue
		}

		stepOrderI, ok := m.Extra[log.ExtraFieldStepOrder]
		if !ok {
			log.Error(ctx, "cdn.log > missing step order extra field")
			continue
		}
		stepOrder, ok := stepOrderI.(int64)
		if !ok {
			log.Error(ctx, "cdn.log > unable to read step order extra field")
			continue
		}

		// Get worker datas
		var workerSign sdk.WorkerSignature
		if err := jws.UnsafeParse(sig.(string), &workerSign); err != nil {
			log.Error(ctx, "cdn.log > unable to unsafe parse log signature: %v", err)
			continue
		}
		workerData, ok := workers[workerSign.WorkerID]
		if !ok {
			var err error
			workerData, err = s.getWorker(ctx, workerSign.WorkerID)
			if err != nil {
				log.Error(ctx, "cdn.log > unable to retrieve worker data from api: %v", err)
				continue
			}
		}
		if err := jws.Verify(workerData.PrivateKey, sig.(string), &workerSign); err != nil {
			log.Error(ctx, "cdn.log > unable to verify signature")
			continue
		}

		pbJob, err := workflow.LoadNodeJobRun(ctx, s.Db, s.Cache, workerSign.JobID)
		if err != nil {
			log.Error(ctx, "cdn.log > unable to verify signature")
			continue
		}

		logDate := time.Unix(0, int64(m.Time*1e9))
		logs := sdk.Log{
			JobID:        pbJob.ID,
			LastModified: &logDate,
			NodeRunID:    pbJob.WorkflowNodeRunID,
			Start:        &logDate,
			StepOrder:    stepOrder,
			Val:          m.Full,
		}
		if err := workflow.AddLog(s.Db, pbJob, &logs, s.Cfg.Log.StepMaxSize); err != nil {
			log.Error(ctx, "cdn.log > unable to insert log")
			continue
		}
	}
}

func (s *Service) getWorker(ctx context.Context, workerID string) (sdk.Worker, error) {
	w, err := worker.LoadWorkerWithDecryptKey(ctx, s.Db, workerID)
	if err != nil {
		return sdk.Worker{}, err
	}
	workers[w.ID] = w
	return w, nil
}
