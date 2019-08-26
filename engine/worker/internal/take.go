package internal

import (
	"context"
	"strings"
	"time"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (w *CurrentWorker) Take(ctx context.Context, job sdk.WorkflowNodeJobRun) error {
	ctxQueueTakeJob, cancelQueueTakeJob := context.WithTimeout(ctx, 20*time.Second)
	defer cancelQueueTakeJob()
	info, err := w.client.QueueTakeJob(ctxQueueTakeJob, job)
	if err != nil {
		return sdk.WrapError(err, "Unable to take job %d", job.ID)
	}
	t := ""
	log.Info("takeWorkflowJob> Job %d taken%s", job.ID, t)

	w.currentJob.context = workerruntime.SetJobID(ctx, job.ID)
	w.currentJob.context = ctx

	// Set build variables
	w.currentJob.wJob = &info.NodeJobRun
	w.currentJob.secrets = info.Secrets
	// Reset build variables
	w.currentJob.newVariables = nil

	start := time.Now()

	//This goroutine try to get the job every 5 seconds, if it fails, it cancel the build.
	ctx, cancel := context.WithCancel(ctx)
	tick := time.NewTicker(5 * time.Second)
	go func(cancel context.CancelFunc, jobID int64, tick *time.Ticker) {
		var nbConnrefused int
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-tick.C:
				if !ok {
					return
				}
				var j *sdk.WorkflowNodeJobRun
				var err error
				ctxGetJSON, cancelGetJSON := context.WithTimeout(ctx, 5*time.Second)
				defer cancelGetJSON()

				if j, err = w.Client().QueueJobInfo(ctxGetJSON, jobID); err != nil {
					if sdk.ErrorIs(err, sdk.ErrWorkflowNodeRunJobNotFound) {
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

				if j == nil || j.Status != sdk.StatusBuilding {
					log.Info("takeWorkflowJob> The job is not more in Building Status. Current Status: %s - Cancelling context - err: %v", j.Status, err)
					cancel()
					return
				}

			}
		}
	}(cancel, job.ID, tick)

	//Run !
	res, err := w.ProcessJob(*info)
	// We keep the err for later usage
	tick.Stop()

	res.RemoteTime = time.Now()
	res.Duration = sdk.Round(time.Since(start), time.Second).String()

	//Wait until the logchannel is empty
	res.BuildID = job.ID

	var lasterr error
	for try := 1; try <= 10; try++ {
		log.Info("takeWorkflowJob> Sending build result...")
		ctxSendResult, cancelSendResult := context.WithTimeout(ctx, 120*time.Second)
		lasterr = w.client.QueueSendResult(ctxSendResult, job.ID, res)
		if lasterr == nil {
			log.Info("takeWorkflowJob> Send build result OK")
			cancelSendResult()
			return nil
		}
		cancelSendResult()
		if ctx.Err() != nil {
			log.Info("takeWorkflowJob> Cannot send build result: HTTP %v - worker cancelled - giving up", lasterr)
			return nil
		}
		log.Warning("takeWorkflowJob> Cannot send build result: HTTP %v - try: %d - new try in 15s", lasterr, try)
		time.Sleep(15 * time.Second)
	}
	log.Error("takeWorkflowJob> Could not send built result 10 times, giving up. job: %d", job.ID)
	if lasterr == nil {
		lasterr = err
	}
	return lasterr
}
