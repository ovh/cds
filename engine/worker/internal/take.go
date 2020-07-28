package internal

import (
	"context"
	"encoding/base64"
	"strings"
	"time"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/log/hook"
)

func (w *CurrentWorker) Take(ctx context.Context, job sdk.WorkflowNodeJobRun) error {
	ctxQueueTakeJob, cancelQueueTakeJob := context.WithTimeout(ctx, 20*time.Second)
	defer cancelQueueTakeJob()
	info, err := w.client.QueueTakeJob(ctxQueueTakeJob, job)
	if err != nil {
		return sdk.WrapError(err, "Unable to take job %d", job.ID)
	}
	t := ""
	log.Info(ctx, "takeWorkflowJob> Job %d taken%s", job.ID, t)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	w.currentJob.context = workerruntime.SetJobID(ctx, job.ID)
	w.currentJob.context = ctx

	// Set build variables
	w.currentJob.wJob = &info.NodeJobRun
	w.currentJob.secrets = info.Secrets
	w.currentJob.projectKey = info.ProjectKey
	w.currentJob.workflowName = info.WorkflowName
	w.currentJob.workflowID = info.WorkflowID
	w.currentJob.runID = info.RunID
	w.currentJob.nodeRunName = info.NodeRunName

	// Reset build variables
	w.currentJob.newVariables = nil

	secretKey := make([]byte, 32)
	if _, err := base64.StdEncoding.Decode(secretKey, []byte(info.SigningKey)); err != nil {
		return sdk.WithStack(err)
	}
	signer, err := jws.NewHMacSigner(secretKey)
	if err != nil {
		return sdk.WithStack(err)
	}
	w.currentJob.signer = signer

	log.Info(ctx, "Setup step logger %s", info.GelfServiceAddr)
	throttlePolicy := hook.NewDefaultThrottlePolicy()

	var graylogCfg = &hook.Config{
		Addr:     info.GelfServiceAddr,
		Protocol: "tcp",
		ThrottlePolicy: &hook.ThrottlePolicyConfig{
			Amount: 100,
			Period: 10 * time.Millisecond,
			Policy: throttlePolicy,
		},
	}

	l, h, err := log.New(ctx, graylogCfg)
	if err != nil {
		return sdk.WithStack(err)
	}
	w.SetGelfLogger(h, l)
	start := time.Now()

	//This goroutine try to get the job every 5 seconds, if it fails, it cancel the build.
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
						log.Info(ctx, "takeWorkflowJob> Unable to load workflow job - Not Found (Request) %d: %v", jobID, err)
						cancel()
						return
					}
					log.Error(ctx, "takeWorkflowJob> Unable to load workflow job (Request) %d: %v", jobID, err)

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

				nbConnrefused = 0
				if j == nil || j.Status != sdk.StatusBuilding {
					log.Info(ctx, "takeWorkflowJob> The job is not more in Building Status. Current Status: %s - Cancelling context - err: %v", j.Status, err)
					cancel()
					return
				}

			}
		}
	}(cancel, job.ID, tick)

	//Run !
	res := w.ProcessJob(*info)
	tick.Stop()

	res.RemoteTime = time.Now()
	res.Duration = sdk.Round(time.Since(start), time.Second).String()

	//Wait until the logchannel is empty
	res.BuildID = job.ID

	// Send the reason as a spawninfo
	if res.Status != sdk.StatusSuccess && res.Reason != "" {
		sp := sdk.SpawnMsg{ID: sdk.MsgWorkflowError.ID, Args: []interface{}{res.Reason}}
		infos := []sdk.SpawnInfo{{
			RemoteTime:  time.Now(),
			Message:     sp,
			UserMessage: sp.DefaultUserMessage(),
		}}
		if err := w.Client().QueueJobSendSpawnInfo(ctx, job.ID, infos); err != nil {
			log.Error(ctx, "processJob> Unable to send spawn info: %v", err)
		}
	}

	var lasterr error
	for try := 1; try <= 10; try++ {
		log.Info(ctx, "takeWorkflowJob> Sending build result...")
		ctxSendResult, cancelSendResult := context.WithTimeout(ctx, 120*time.Second)
		lasterr = w.client.QueueSendResult(ctxSendResult, job.ID, res)
		if lasterr == nil {
			log.Info(ctx, "takeWorkflowJob> Send build result OK")
			cancelSendResult()
			return nil
		}
		cancelSendResult()
		if ctx.Err() != nil {
			log.Info(ctx, "takeWorkflowJob> Cannot send build result: HTTP %v - worker cancelled - giving up", lasterr)
			return nil
		}
		log.Warning(ctx, "takeWorkflowJob> Cannot send build result for job id %d: HTTP %v - try: %d - new try in 15s", job.ID, lasterr, try)
		time.Sleep(15 * time.Second)
	}
	log.Error(ctx, "takeWorkflowJob> Could not send built result 10 times, giving up. job: %d", job.ID)
	if lasterr == nil {
		lasterr = err
	}
	return lasterr
}
