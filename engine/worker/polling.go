package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/viper"

	"github.com/golang/protobuf/ptypes"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var firstViewQueue = true

func postCheckRequirementError(r *sdk.Requirement, err error) {
	s := fmt.Sprintf("Error checking requirement Name=%s Type=%s Value=%s :%s", r.Name, r.Type, r.Value, err)
	sdk.Request("POST", "/queue/requirements/errors", []byte(s))
}

func (w *currentWorker) takePipelineBuildJob(ctx context.Context, pipelineBuildJobID int64, isBooked bool) {
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
		return
	}
	if code != http.StatusOK {
		return
	}

	pbji := worker.PipelineBuildJobInfo{}
	if err := json.Unmarshal([]byte(data), &pbji); err != nil {
		log.Info("takeJob> Cannot unmarshal action: %s", err)
		return
	}

	w.currentJob.pbJob = pbji.PipelineBuildJob

	// Reset build variables
	w.currentJob.buildVariables = nil
	start := time.Now()
	//Run !
	res := w.run(ctx, &pbji)
	now, _ := ptypes.TimestampProto(time.Now())
	res.RemoteTime = now
	res.Duration = sdk.Round(time.Since(start), time.Second).String()

	//Wait until the logchannel is empty
	if ctx.Err() == nil {
		w.drainLogsAndCloseLogger(ctx)
	}

	log.Debug("Send result")

	path = fmt.Sprintf("/queue/%d/result", pipelineBuildJobID)
	body, errm := json.MarshalIndent(res, " ", " ")
	if errm != nil {
		log.Error("takeJob> Cannot marshal result: %s", errm)
		w.unregister()
		return
	}

	code = 300
	var isThereAnyHopeLeft = 10
	for code >= 300 {
		var errre error
		_, code, errre = sdk.Request("POST", path, body)
		if code == http.StatusNotFound {
			log.Info("takeJob> Cannot send build result: PipelineBuildJob does not exists anymore")
			w.unregister() // well...
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

	if viper.GetBool("single_use") {
		// Unregister from engine
		if err := w.unregister(); err != nil {
			log.Warning("takeJob> could not unregister: %s", err)
		}
	}

}
