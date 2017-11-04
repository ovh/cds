package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes"

	"github.com/ovh/cds/engine/api/grpc"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// takeWorkflowJob try to take a job.
// If Take is not possible (as Job already booked for example)
// it will return true (-> can work on another job), false, otherwise
func (w *currentWorker) takeWorkflowJob(ctx context.Context, job sdk.WorkflowNodeJobRun) (bool, error) {
	info, err := w.client.QueueTakeJob(job, w.bookedJobID == job.ID)
	if err != nil {
		return true, sdk.WrapError(err, "takeWorkflowJob> Unable to take workflow node run job. This worker can work on another job.")
	}
	t := ""
	if w.bookedJobID == job.ID {
		t = ", this was my booked job"
	}
	log.Info("takeWorkflowJob> Job %d taken%s", job.ID, t)

	w.nbActionsDone++
	// Set build variables
	w.currentJob.wJob = &info.NodeJobRun
	// Reset build variables
	w.currentJob.gitsshPath = ""
	w.currentJob.pkey = ""
	w.currentJob.buildVariables = nil

	start := time.Now()

	//This goroutine try to get the pipeline build job every 5 seconds, if it fails, it cancel the build.
	ctx, cancel := context.WithCancel(ctx)
	tick := time.NewTicker(5 * time.Second)
	go func(cancel context.CancelFunc, jobID int64, tick *time.Ticker) {
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-tick.C:
				if !ok {
					return
				}
				b, _, err := sdk.Request("GET", fmt.Sprintf("/queue/workflows/%d/infos", jobID), nil)
				if err != nil {
					log.Error("Unable to load pipeline build job %d", jobID)
					cancel()
					return
				}

				j := &sdk.WorkflowNodeJobRun{}
				if err := json.Unmarshal(b, j); err != nil {
					log.Error("Unable to load job run %d: %v", jobID, err)
					cancel()
					return
				}
				if j.Status != sdk.StatusBuilding.String() {
					cancel()
					return
				}

			}
		}
	}(cancel, job.ID, tick)

	// Reset build variables
	w.currentJob.buildVariables = nil
	//Run !
	res := w.processJob(ctx, info)
	tick.Stop()

	now, _ := ptypes.TimestampProto(time.Now())
	res.RemoteTime = now
	res.Duration = sdk.Round(time.Since(start), time.Second).String()

	//Wait until the logchannel is empty
	w.drainLogsAndCloseLogger(ctx)

	// Try to send result through grpc
	if w.grpc.conn != nil {
		client := grpc.NewWorkflowQueueClient(w.grpc.conn)
		res.BuildID = job.ID
		_, err := client.SendResult(ctx, &res)
		if err == nil {
			return false, nil
		}
		log.Error("Unable to send result through grpc: %v", err)
	}

	return false, w.client.QueueSendResult(job.ID, res)
}
