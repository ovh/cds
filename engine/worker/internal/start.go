package internal

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

func StartWorker(ctx context.Context, w *CurrentWorker, bookedJobID string) (mainError error) {
	ctx = context.WithValue(ctx, log.Field("permJobID"), bookedJobID)

	log.Info(ctx, "Starting worker %s on job %d", w.Name(), bookedJobID)

	if bookedJobID == "0" {
		return errors.Errorf("startWorker: bookedJobID is mandatory. val: %s", bookedJobID)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	httpServerCtx, stopHTTPServer := context.WithCancel(ctx)
	defer stopHTTPServer()
	if err := w.Serve(httpServerCtx); err != nil {
		return err
	}

	//Register
	if err := w.Register(ctx); err != nil {
		return sdk.WrapError(err, "unable to register to CDS")
	}

	//Register every 10 seconds
	refreshTick := time.NewTicker(30 * time.Second)

	// start queue polling
	errsChan := make(chan error, 1)

	//Definition of the function which must be called to stop the worker
	var endFunc = func() {
		log.Info(ctx, "Stopping worker %s", w.Name())
		if err := w.Unregister(ctx); err != nil {
			log.Error(ctx, "Unable to unregister: %v", err)
			mainError = err
		}
		refreshTick.Stop()
		cancel()
		stopHTTPServer()

		if err := ctx.Err(); err != nil {
			log.Warn(ctx, "Exiting worker: %v", err)
		} else {
			log.Warn(ctx, "Exiting worker")
		}
	}

	runJob, err := processBookedWJob(ctx, w, bookedJobID)
	if err != nil {
		// Unbook job
		if errR := w.Client().QueueJobRelease(ctx, bookedJobID); errR != nil {
			log.Error(ctx, "runCmd> QueueJobRelease> Cannot release job")
		}
		// this worker was spawned for a job
		// this job can't be process (err != nil)
		// so, call endFunc() now, this worker don't have to work
		// on another job
		endFunc()
		return sdk.WrapError(err, "unable to process booked job")
	}

	if err := w.Client().WorkerSetStatus(ctx, sdk.StatusWaiting); err != nil {
		log.Error(ctx, "WorkerSetStatus> error on WorkerSetStatus(ctx, sdk.StatusWaiting): %s", err)
	}

	// Errors check loops
	go func() {
		for err := range errsChan {
			log.Error(ctx, "An error has occured: %v", err)
			if strings.Contains(err.Error(), "not authenticated") {
				endFunc()
				return
			}
		}
	}()

	// Register (heartbeat loop)
	go func() {
		var nbErrors int
		for {
			select {
			case <-ctx.Done():
				return
			case <-refreshTick.C:
				if err := w.Client().WorkerRefresh(ctx); err != nil {
					log.Error(ctx, "Heartbeat failed: %v", err)
					nbErrors++
					if nbErrors == 5 {
						errsChan <- err
					}
				}
				nbErrors = 0
			}
		}
	}()

	//Take the job
	log.Debug(ctx, "checkQueue> Try take the job %d", runJob.ID)
	if err := w.Take(ctx, *runJob); err != nil {
		log.Info(ctx, "Unable to run this job  %d. Take info: %v", runJob.ID, err)
		errsChan <- err
	}

	// Unregister from engine
	log.Info(ctx, "Job is done. Unregistering...")
	endFunc()
	return nil

}

func processBookedWJob(ctx context.Context, w *CurrentWorker, bookedWJobID string) (*sdk.WorkflowNodeJobRun, error) {
	log.Debug(ctx, "Try to take the workflow node job %s", bookedWJobID)
	wjob, err := w.Client().QueueJobInfo(ctx, bookedWJobID)
	if err != nil {
		return nil, sdk.WrapError(err, "Unable to load workflow node job %s", bookedWJobID)
	}

	requirementsOK, errRequirements := checkRequirements(ctx, w, &wjob.Job.Action)
	if !requirementsOK {
		var details string
		for _, r := range errRequirements {
			details += fmt.Sprintf(" %s(%s)", r.Value, r.Type)
		}
		infos := []sdk.SpawnInfo{{
			RemoteTime: time.Now(),
			Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoWorkerForJobError.ID, Args: []interface{}{w.Name(), details}},
		}}
		if err := w.Client().QueueJobSendSpawnInfo(ctx, strconv.FormatInt(wjob.ID, 10), infos); err != nil {
			return nil, sdk.WrapError(err, "Cannot record QueueJobSendSpawnInfo for job (err spawn): %d", wjob.ID)
		}
		return nil, fmt.Errorf("processBookedWJob> the worker have no all requirements")
	}

	pluginsOK, errPlugins := checkPlugins(ctx, w, *wjob)
	if !pluginsOK {
		var details = errPlugins.Error()
		infos := []sdk.SpawnInfo{{
			RemoteTime: time.Now(),
			Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoWorkerForJobError.ID, Args: []interface{}{w.Name(), details}},
		}}
		if err := w.Client().QueueJobSendSpawnInfo(ctx, strconv.FormatInt(wjob.ID, 10), infos); err != nil {
			return nil, sdk.WrapError(err, "Cannot record QueueJobSendSpawnInfo for job (err spawn): %d", wjob.ID)
		}
		return nil, fmt.Errorf("processBookedWJob> the worker doesn't have the required plugins")
	}
	return wjob, nil
}
