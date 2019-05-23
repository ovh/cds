package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"

	"github.com/ovh/cds/engine/api/grpc"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

// takeWorkflowJob try to take a job.
// If Take is not possible (as Job already booked for example)
// it will return true (-> can work on another job), false, otherwise
func (w *currentWorker) takeWorkflowJob(ctx context.Context, job sdk.WorkflowNodeJobRun) (bool, error) {
	ctxQueueTakeJob, cancelQueueTakeJob := context.WithTimeout(ctx, 20*time.Second)
	defer cancelQueueTakeJob()
	info, err := w.client.QueueTakeJob(ctxQueueTakeJob, job, w.bookedWJobID == job.ID)
	if err != nil {
		if w.bookedWJobID == job.ID {
			return false, sdk.WrapError(err, "Unable to take workflow node run job. This worker can't work on another job.")
		}
		return true, sdk.WrapError(err, "Unable to take workflow node run job. This worker can work on another job.")
	}
	t := ""
	if w.bookedWJobID == job.ID {
		t = ", this was my booked job"
	}
	log.Info("takeWorkflowJob> Job %d taken%s", job.ID, t)

	w.nbActionsDone++
	// Set build variables
	w.currentJob.wJob = &info.NodeJobRun
	w.currentJob.secrets = info.Secrets
	// Reset build variables
	w.currentJob.gitsshPath = ""
	w.currentJob.pkey = ""
	w.currentJob.buildVariables = nil

	start := time.Now()

	//This goroutine try to get the job every 5 seconds, if it fails, it cancel the build.
	ctxWithCancel, cancel := context.WithCancel(ctx)
	tick := time.NewTicker(5 * time.Second)
	workflowNodeJobRun := &sdk.WorkflowNodeJobRun{}
	go func(cancel context.CancelFunc, jobID int64, tick *time.Ticker, workflowNodeJobRun *sdk.WorkflowNodeJobRun) {
		var nbConnrefused int
		for {
			select {
			case <-ctxWithCancel.Done():
				return
			case _, ok := <-tick.C:
				if !ok {
					return
				}

				ctxGetJSON, cancelGetJSON := context.WithTimeout(ctxWithCancel, 5*time.Second)
				defer cancelGetJSON()
				code, err := w.client.(cdsclient.Raw).GetJSON(ctxGetJSON, fmt.Sprintf("/queue/workflows/%d/infos", jobID), workflowNodeJobRun)
				if workflowNodeJobRun.Status == sdk.StatusStopping.String() {
					log.Info("Job status is stopping on API")
					cancel()
					return
				}
				if err != nil {
					if code == http.StatusNotFound {
						log.Info("takeWorkflowJob> Unable to load workflow job - Not Found (Request) %d: %v", jobID, err)
						cancel()
						return
					}
					log.Error("takeWorkflowJob> Unable to load workflow job (Request) %d: %v", jobID, err)

					// If we got a "connection refused", retry 5 times
					if strings.Contains(err.Error(), "connection refused") {
						nbConnrefused++
					}
					if nbConnrefused >= 5 {
						cancel()
						return
					}

					continue // do not kill the worker here, could be a timeout
				}

				if workflowNodeJobRun.Status != sdk.StatusBuilding.String() {
					log.Info("takeWorkflowJob> The job is not more in Building Status. Current Status: %s - Cancelling context - err: %v", workflowNodeJobRun.Status, err)
					cancel()
					return
				}

			}
		}
	}(cancel, job.ID, tick, workflowNodeJobRun)

	// Reset build variables
	w.currentJob.buildVariables = nil
	//Run !
	res := w.processJob(ctx, ctxWithCancel, info)
	tick.Stop()

	if ctxWithCancel.Err() != nil && workflowNodeJobRun != nil && workflowNodeJobRun.Status == sdk.StatusStopping.String() {
		res.Status = sdk.StatusStopped.String()
		log.Debug("takeWorkflowJob> job is stopped")
	}

	now, _ := ptypes.TimestampProto(time.Now())
	res.RemoteTime = now
	res.Duration = sdk.Round(time.Since(start), time.Second).String()

	//Wait until the logchannel is empty
	w.drainLogsAndCloseLogger(ctxWithCancel)
	res.BuildID = job.ID
	// Try to send result through grpc
	if w.grpc.conn != nil {
		client := grpc.NewWorkflowQueueClient(w.grpc.conn)
		ctxSendResult, cancelSendResult := context.WithTimeout(ctx, 5*time.Second)
		_, err := client.SendResult(ctxSendResult, &res)
		if err == nil {
			cancelSendResult()
			return false, nil
		}
		cancelSendResult()
		log.Error("Unable to send result through grpc: %v", err)
	}

	var lasterr error
	for try := 1; try <= 10; try++ {
		log.Info("takeWorkflowJob> Sending build result with status %s...", res.Status)
		ctxSendResult, cancelSendResult := context.WithTimeout(ctx, 5*time.Second)
		lasterr = w.client.QueueSendResult(ctxSendResult, job.ID, res)
		if lasterr == nil {
			log.Info("takeWorkflowJob> Send build result OK")
			cancelSendResult()
			return false, nil
		}
		cancelSendResult()
		if ctxWithCancel.Err() != nil {
			log.Info("takeWorkflowJob> Cannot send build result: HTTP %v - worker cancelled - giving up", lasterr)
			return false, nil
		}
		log.Warning("takeWorkflowJob> Cannot send build result: HTTP %v - try: %d - new try in 15s", lasterr, try)
		time.Sleep(15 * time.Second)
	}
	log.Error("takeWorkflowJob> Could not send built result 10 times, giving up. job: %d", job.ID)
	return false, lasterr
}
