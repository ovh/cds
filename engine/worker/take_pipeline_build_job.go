package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/protobuf/ptypes"

	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// takePipelineBuildJob takes pipeline build job. If failed, this func return true if worker can safely check another job
func (w *currentWorker) takePipelineBuildJob(ctx context.Context, pipelineBuildJobID int64, isBooked bool) bool {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	log.Debug("takePipelineBuildJob> Begin %p", ctx)
	defer func() {
		log.Debug("takePipelineBuildJob> End %p (%s)", ctx, ctx.Err())
	}()

	in := worker.TakeForm{Time: time.Now()}
	if isBooked {
		in.BookedJobID = pipelineBuildJobID
	}

	bodyTake, errm := json.Marshal(in)
	if errm != nil {
		log.Info("takeJob> Cannot marshal body: %s", errm)
	}

	w.nbActionsDone++
	w.currentJob.gitsshPath = ""
	w.currentJob.pkey = ""
	path := fmt.Sprintf("/queue/%d/take", pipelineBuildJobID)
	data, code, errr := sdk.Request("POST", path, bodyTake)
	if errr != nil {
		log.Info("takeJob> Cannot take job %d : %s", pipelineBuildJobID, errr)
		return true
	}
	if code != http.StatusOK {
		return true
	}

	pbji := worker.PipelineBuildJobInfo{}
	if err := json.Unmarshal([]byte(data), &pbji); err != nil {
		log.Info("takeJob> Cannot unmarshal action: %s", err)
		return false
	}

	w.currentJob.pbJob = pbji.PipelineBuildJob

	//This goroutine try to get the pipeline build job every 5 seconds, if it fails, it cancel the build.
	tick := time.NewTicker(5 * time.Second)
	go func(jobID int64, tick *time.Ticker) {
		for {
			select {
			case <-ctx.Done():
				log.Error("Exiting pipeline build job info goroutine: %v", ctx.Err())
				tick.Stop()
				return

			case _, ok := <-tick.C:
				if !ok {
					return
				}
				b, _, err := sdk.Request("GET", fmt.Sprintf("/queue/%d/infos", jobID), nil)
				if err != nil {
					log.Error("Unable to load pipeline build job %d", jobID)
					log.Error("Cancelling context")
					cancel()
					return
				}

				j := &sdk.PipelineBuildJob{}
				if err := json.Unmarshal(b, j); err != nil {
					log.Error("Unable to load pipeline build job %d: %v", jobID, err)
					log.Error("Cancelling context")
					cancel()
					return
				}
				if j.Status != sdk.StatusBuilding.String() {
					log.Error("Cancelling context")
					cancel()
					return
				}

			}
		}
	}(pipelineBuildJobID, tick)

	// Reset build variables
	w.currentJob.buildVariables = nil
	start := time.Now()

	//Run !
	res := w.run(ctx, &pbji)
	tick.Stop()
	now, _ := ptypes.TimestampProto(time.Now())
	res.RemoteTime = now
	res.Duration = sdk.Round(time.Since(start), time.Second).String()

	//Wait until the logchannel is empty
	w.drainLogsAndCloseLogger(ctx)

	path = fmt.Sprintf("/queue/%d/result", pipelineBuildJobID)
	body, errm := json.MarshalIndent(res, " ", " ")
	if errm != nil {
		log.Error("takeJob> Cannot marshal result: %s", errm)
		return false
	}

	code = 300
	var isThereAnyHopeLeft = 10
	for code >= 300 {
		var errre error
		_, code, errre = sdk.Request("POST", path, body)
		if code == http.StatusNotFound {
			log.Info("takeJob> Cannot send build result: PipelineBuildJob does not exists anymore")
			break
		}
		if errre == nil && code < 300 {
			log.Info("BuildResult sent.")
			break
		}

		if errre != nil {
			log.Warning("takeJob> Cannot send build result: %s", errre)
		} else {
			log.Warning("takeJob> Cannot send build result: HTTP %d", code)
		}

		time.Sleep(5 * time.Second)
		isThereAnyHopeLeft--
		if isThereAnyHopeLeft < 0 {
			log.Info("takeJob> Could not send built result 10 times, giving up")
			break
		}
	}
	return false
}
