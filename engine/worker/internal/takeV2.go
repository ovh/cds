package internal

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/jws"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/log/hook"
	"github.com/rockbears/log"
)

func (w *CurrentWorker) V2Take(ctx context.Context, region, jobRunID string) error {
	info, err := w.clientV2.V2QueueWorkerTakeJob(ctx, region, jobRunID)
	if err != nil {
		return sdk.WrapError(err, "Unable to take job %s", jobRunID)
	}

	ctx = context.WithValue(ctx, cdslog.Workflow, info.RunJob.WorkflowName)
	ctx = context.WithValue(ctx, cdslog.Project, info.RunJob.ProjectKey)

	log.Info(ctx, "takeWorkflowJob> Job %s taken", jobRunID)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	w.currentJobV2.context = ctx
	w.currentJobV2.runJob = &info.RunJob
	w.currentJobV2.sensitiveDatas = info.SensitiveDatas
	w.currentJobV2.integrations = make(map[string]sdk.ProjectIntegration)
	w.actions = info.AsCodeActions
	w.currentJobV2.runJobContext = info.Contexts
	w.actionPlugin = make(map[string]*sdk.GRPCPlugin)

	// setup blur
	w.blur, err = sdk.NewBlur(w.currentJobV2.sensitiveDatas)
	if err != nil {
		return err
	}

	secretKey := make([]byte, 32)
	if _, err := base64.StdEncoding.Decode(secretKey, []byte(info.SigningKey)); err != nil {
		return sdk.WithStack(err)
	}
	signer, err := jws.NewHMacSigner(secretKey)
	if err != nil {
		return sdk.WithStack(err)
	}
	w.signer = signer

	log.Info(ctx, "Setup step logger %s", w.cfg.GelfServiceAddr)
	throttlePolicy := hook.NewDefaultThrottlePolicy()

	var graylogCfg = &hook.Config{
		Addr:     w.cfg.GelfServiceAddr,
		Protocol: "tcp",
		ThrottlePolicy: &hook.ThrottlePolicyConfig{
			Amount: 100,
			Period: 10 * time.Millisecond,
			Policy: throttlePolicy,
		},
	}

	if w.cfg.GelfServiceAddrEnableTLS {
		tcpCDNUrl := w.cfg.GelfServiceAddr
		// Check if the url has a scheme
		// We have to remove if to retrieve the hostname
		if i := strings.Index(tcpCDNUrl, "://"); i > -1 {
			tcpCDNUrl = tcpCDNUrl[i+3:]
		}
		tcpCDNHostname, _, err := net.SplitHostPort(tcpCDNUrl)
		if err != nil {
			return sdk.WithStack(err)
		}

		graylogCfg.TLSConfig = &tls.Config{ServerName: tcpCDNHostname}
	}

	l, h, err := cdslog.New(ctx, graylogCfg)
	if err != nil {
		return sdk.WithStack(err)
	}
	w.SetGelfLogger(h, l)

	// This goroutine try to get the job every 5 seconds, if it fails, it cancel the build.
	tick := time.NewTicker(5 * time.Second)
	go func(cancel context.CancelFunc, runJobID string, tick *time.Ticker) {
		defer tick.Stop()
		var nbConnrefused int
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-tick.C:
				if !ok {
					return
				}
				ctxGetJSON, cancelGetJSON := context.WithTimeout(ctx, 5*time.Second)
				j, err := w.ClientV2().V2QueueGetJobRun(ctxGetJSON, region, runJobID)
				if err != nil {
					cancelGetJSON()
					if sdk.ErrorIs(err, sdk.ErrWorkflowNodeRunJobNotFound) {
						log.Info(ctx, "V2Take> Unable to load workflow run job - Not Found (Request) %s: %v", runJobID, err)
						cancel()
						return
					}
					log.Error(ctx, "V2Take> Unable to load workflow run job (Request) %s: %v", runJobID, err)

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
				cancelGetJSON()
				nbConnrefused = 0
				if j == nil || j.RunJob.Status != sdk.V2WorkflowRunJobStatusBuilding {
					log.Warn(ctx, "V2Take> The job is not more in Building Status. Current Status: %s - Cancelling context - err: %v", j.RunJob.Status, err)
					cancel()
					return
				}

			}
		}
	}(cancel, jobRunID, tick)

	// Run !
	res := w.V2ProcessJob()
	res.Time = time.Now()

	// Send the reason as a spawninfo
	if res.Status != sdk.V2WorkflowRunJobStatusSuccess && res.Error != "" {
		info := sdk.V2SendJobRunInfo{
			Level:   sdk.WorkflowRunInfoLevelError,
			Message: fmt.Sprintf("⚠ An error has occurred: %s", res.Error),
			Time:    time.Now(),
		}
		if err := w.ClientV2().V2QueuePushJobInfo(ctx, w.currentJobV2.runJob.Region, w.currentJobV2.runJob.ID, info); err != nil {
			log.Error(ctx, "processJob> Unable to send spawn info: %v", err)
		}
	}

	var lasterr error
	for try := 1; try <= 10; try++ {
		log.Info(ctx, "takeWorkflowJob> Sending build result...")
		lasterr = w.ClientV2().V2QueueJobResult(ctx, w.currentJobV2.runJob.Region, w.currentJobV2.runJob.ID, res)
		if lasterr == nil {
			log.Info(ctx, "takeWorkflowJob> Build result sent")
			return nil
		}
		if ctx.Err() != nil {
			log.Info(ctx, "takeWorkflowJob> Cannot send build result: HTTP %v - worker cancelled - giving up", lasterr)
			return nil
		}
		log.Warn(ctx, "takeWorkflowJob> Cannot send build result for job id %s: HTTP %v - try: %d - new try in 15s", w.currentJobV2.runJob.ID, lasterr, try)
		time.Sleep(15 * time.Second)
	}
	log.Error(ctx, "takeWorkflowJob> Could not send built result 10 times, giving up. job: %s", w.currentJobV2.runJob.ID)
	if lasterr == nil {
		lasterr = err
	}
	return lasterr
}
