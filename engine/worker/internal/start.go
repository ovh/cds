package internal

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func StartWorker(ctx context.Context, w *CurrentWorker, bookedJobID int64) (mainError error) {
	log.Info("Starting worker %s", w.Name())
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	httpServerCtx, stopHTTPServer := context.WithCancel(ctx)
	defer stopHTTPServer()
	w.Serve(httpServerCtx)

	//Register
	if err := w.Register(ctx); err != nil {
		return sdk.WrapError(err, "unable to register to CDS")
	}

	//Register every 10 seconds
	refreshTick := time.NewTicker(30 * time.Second)

	// start queue polling
	jobsChan := make(chan sdk.WorkflowNodeJobRun, 50)
	errsChan := make(chan error, 1)

	//Definition of the function which must be called to stop the worker
	var endFunc = func() {
		log.Info("Stopping worker %s", w.Name())
		if err := w.Unregister(); err != nil {
			log.Error("Unable to unregister: %v", err)
			mainError = err
		}
		refreshTick.Stop()
		cancel()
		stopHTTPServer()

		if err := ctx.Err(); err != nil {
			log.Warning("Exiting worker: %v", err)
		} else {
			log.Warning("Exiting worker")
		}
	}

	if bookedJobID != 0 {
		if err := processBookedWJob(ctx, w, jobsChan, bookedJobID); err != nil {
			// Unbook job
			if errR := w.Client().QueueJobRelease(ctx, bookedJobID); errR != nil {
				log.Error("runCmd> QueueJobRelease> Cannot release job")
			}
			bookedJobID = 0
			// this worker was spawned for a job
			// this job can't be process (errP != nil)
			// so, call endFunc() now, this worker don't have to work
			// on another job
			endFunc()
			return sdk.WrapError(err, "unable to process booked job")
		}
	} else {
		sdk.GoRoutine(ctx, "worker.QueuePolling", func(ctx context.Context) {
			if err := w.Client().QueuePolling(ctx, jobsChan, errsChan, 2*time.Second, "", nil); err != nil {
				log.Info("Queues polling stopped: %v", err)
			}
		})
	}

	if err := w.Client().WorkerSetStatus(ctx, sdk.StatusWaiting); err != nil {
		log.Error("WorkerSetStatus> error on WorkerSetStatus(ctx, sdk.StatusWaiting): %s", err)
	}

	// Errors check loops
	go func(errs chan error) {
		for {
			select {
			case err := <-errs:
				log.Error("An error has occured: %v", err)
				if strings.Contains(err.Error(), "not authenticated") {
					endFunc()
					return
				}
			}
		}
	}(errsChan)

	// Register (heartbeat loop)
	go func() {
		var nbErrors int
		for {
			select {
			case <-ctx.Done():
				return
			case <-refreshTick.C:
				if err := w.Client().WorkerRefresh(ctx); err != nil {
					log.Error("Heartbeat failed: %v", err)
					nbErrors++
					if nbErrors == 5 {
						errsChan <- err
					}
				}
				nbErrors = 0
			}
		}
	}()

	// main loop
	for {
		if ctx.Err() != nil {
			endFunc()
			return ctx.Err()
		}

		select {
		case <-ctx.Done():
			endFunc()
			return ctx.Err()
		case j := <-jobsChan:
			if j.ID == 0 {
				continue
			}
			log.Debug("checkQueue> Receive workflow job %d", j.ID)

			var requirementsOK, pluginsOK bool
			var t string
			if bookedJobID == 0 { // If we already check the requirements before and it was OK
				requirementsOK, _ = checkRequirements(w, &j.Job.Action)
				if j.ID == bookedJobID {
					t = ", this was my booked job"
				}

				var errPlugins error
				pluginsOK, errPlugins = checkPluginDeployment(w, j)
				if !pluginsOK {
					log.Error("Plugins doesn't match: %v", errPlugins)
				}
			} else { // Because already checked previously
				requirementsOK = true
				pluginsOK = true
			}

			//Take the job
			if requirementsOK && pluginsOK {
				log.Debug("checkQueue> Try take the job %d%s", j.ID, t)
				if err := w.Take(ctx, j); err != nil {
					log.Info("Unable to run this job  %d. Take info:%s: %v", j.ID, t, err)
					bookedJobID = 0
					errsChan <- err
				}
			}

			if err := w.Client().WorkerSetStatus(ctx, sdk.StatusWaiting); err != nil {
				log.Error("WorkerSetStatus> error on WorkerSetStatus(ctx, sdk.StatusWaiting): %s", err)
			}

			// Unregister from engine
			log.Info("Job is done. Unregistering...")
			defer endFunc()
			return nil
		}
	}
}

func processBookedWJob(ctx context.Context, w *CurrentWorker, wjobs chan<- sdk.WorkflowNodeJobRun, bookedWJobID int64) error {
	log.Debug("Try to take the workflow node job %d", bookedWJobID)
	wjob, err := w.Client().QueueJobInfo(ctx, bookedWJobID)
	if err != nil {
		return sdk.WrapError(err, "Unable to load workflow node job %d", bookedWJobID)
	}

	requirementsOK, errRequirements := checkRequirements(w, &wjob.Job.Action)
	if !requirementsOK {
		var details string
		for _, r := range errRequirements {
			details += fmt.Sprintf(" %s(%s)", r.Value, r.Type)
		}
		infos := []sdk.SpawnInfo{{
			RemoteTime: time.Now(),
			Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoWorkerForJobError.ID, Args: []interface{}{w.Name(), details}},
		}}
		if err := w.Client().QueueJobSendSpawnInfo(ctx, wjob.ID, infos); err != nil {
			return sdk.WrapError(err, "Cannot record QueueJobSendSpawnInfo for job (err spawn): %d", wjob.ID)
		}
		return fmt.Errorf("processBookedWJob> the worker have no all requirements")
	}

	pluginsOK, errPlugins := checkPluginDeployment(w, *wjob)
	if !pluginsOK {
		var details = errPlugins.Error()

		infos := []sdk.SpawnInfo{{
			RemoteTime: time.Now(),
			Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoWorkerForJobError.ID, Args: []interface{}{w.Name(), details}},
		}}
		if err := w.Client().QueueJobSendSpawnInfo(ctx, wjob.ID, infos); err != nil {
			return sdk.WrapError(err, "Cannot record QueueJobSendSpawnInfo for job (err spawn): %d", wjob.ID)
		}
		return fmt.Errorf("processBookedWJob> the worker have no all plugins")
	}

	// requirementsOK is ok
	wjobs <- *wjob

	return nil
}
