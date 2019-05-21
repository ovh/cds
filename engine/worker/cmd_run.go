package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/gops/agent"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func cmdRun(w *currentWorker) *cobra.Command {
	c := &cobra.Command{
		Use:    "run",
		Hidden: true, // user should not use this command directly
		Run:    runCmd(w),
	}

	initFlagsRun(c)
	return c
}

func runCmd(w *currentWorker) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		//Initialize  context
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)

		initFlags(cmd, w)

		if FlagString(cmd, flagLogLevel) == "debug" {
			if err := agent.Listen(nil); err != nil {
				sdk.Exit("Error on starting gops agent", err)
			}
		}

		hostname, errh := os.Hostname() // no check of err here
		if errh != nil {
			sdk.Exit(fmt.Sprintf("error compute hostname: %s", errh))
		}
		log.Info("CDS Worker starting")
		log.Info("version: %s", sdk.VERSION)
		log.Info("hostname: %s", hostname)
		log.Info("single-use: %t", w.singleUse)

		httpServerCtx, stopHTTPServer := context.WithCancel(context.Background())
		w.initServer(httpServerCtx)

		// Gracefully shutdown connections
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
		defer func() {
			log.Info("Run signal.Stop. Hostname: %s", hostname)
			signal.Stop(c)
			stopHTTPServer()
			cancel()
		}()

		go func() {
			select {
			case <-c:
				cancel()
				return
			case <-ctx.Done():
				return
			}
		}()

		ttl := FlagInt(cmd, flagTTL)
		time.AfterFunc(time.Duration(ttl)*time.Minute, func() {
			if w.nbActionsDone == 0 {
				cancel()
			}
		})

		//Register
		t0 := time.Now()
		for w.id == "" && ctx.Err() == nil {
			if t0.Add(6 * time.Minute).Before(time.Now()) {
				log.Error("Unable to register to CDS. Exiting...")
				cancel()
				os.Exit(1)
			}
			if err := w.doRegister(); err != nil {
				log.Error("Unable to register to CDS (%v). Retry", err)
			}
			time.Sleep(2 * time.Second)
		}

		//Register every 10 seconds
		registerTick := time.NewTicker(10 * time.Second)
		refreshTick := time.NewTicker(30 * time.Second)

		// start logger routine with a large buffer
		w.logger.logChan = make(chan sdk.Log, 100000)
		go w.logProcessor(ctx)

		// start queue polling
		wjobs := make(chan sdk.WorkflowNodeJobRun, 50)
		errs := make(chan error, 1)

		//Definition of the function which must be called to stop the worker
		var endFunc = func() {
			log.Info("Stopping the worker")
			w.drainLogsAndCloseLogger(ctx)
			registerTick.Stop()
			refreshTick.Stop()
			w.unregister()
			cancel()
			stopHTTPServer()

			if FlagBool(cmd, flagForceExit) {
				log.Info("Exiting worker with force_exit true")
				return
			}

			if w.hatchery.name != "" {
				log.Info("Waiting 30min to be killed by hatchery, if not killed, worker will exit")
				time.Sleep(30 * time.Minute)
			}

			if err := ctx.Err(); err != nil {
				log.Error("Exiting worker: %v", err)
			} else {
				log.Info("Exiting worker")
			}
		}

		var exceptJobID int64
		if w.bookedWJobID != 0 {
			if errP := w.processBookedWJob(ctx, wjobs); errP != nil {
				// Unbook job
				if errR := w.client.QueueJobRelease(w.bookedWJobID); errR != nil {
					log.Error("runCmd> QueueJobRelease> Cannot release job")
				}
				exceptJobID = w.bookedWJobID
				w.bookedWJobID = 0
				// this worker was spawned for a job
				// this job can't be process (errP != nil)
				// so, call endFunc() now, this worker don't have to work
				// on another job
				endFunc()
				return
			}
			exceptJobID = w.bookedWJobID
		}

		if err := w.client.WorkerSetStatus(ctx, sdk.StatusWaiting); err != nil {
			log.Error("WorkerSetStatus> error on WorkerSetStatus(ctx, sdk.StatusWaiting): %s", err)
		}

		go func(ctx context.Context, exceptID *int64) {
			if err := w.client.QueuePolling(ctx, wjobs, errs, 2*time.Second, 0, "", nil, exceptID); err != nil {
				log.Info("Queues polling stopped: %v", err)
			}
		}(ctx, &exceptJobID)

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
		}(errs)

		// Register (heartbeat loop)
		go func() {
			var nbErrors int
			for {
				select {
				case <-ctx.Done():
					return
				case <-refreshTick.C:
					if err := w.client.WorkerRefresh(ctx); err != nil {
						log.Error("Heartbeat failed: %v", err)
						nbErrors++
						if nbErrors == 5 {
							errs <- err
						}
					}
					nbErrors = 0
				case <-registerTick.C:
					if err := w.doRegister(); err != nil {
						log.Error("Register failed: %v", err)
						nbErrors++
						if nbErrors == 5 {
							errs <- err
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
				return
			}

			select {
			case <-ctx.Done():
				endFunc()
				return
			case j := <-wjobs:
				if j.ID == 0 {
					continue
				}
				log.Debug("checkQueue> Receive workflow job %d", j.ID)

				var requirementsOK, pluginsOK bool
				var t string
				if exceptJobID != j.ID && w.bookedWJobID == 0 { // If we already check the requirements before and it was OK
					requirementsOK, _ = checkRequirements(w, &j.Job.Action, j.ID)
					if j.ID == w.bookedWJobID {
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
					if canWorkOnAnotherJob, err := w.takeWorkflowJob(ctx, j); err != nil {
						log.Info("Unable to run this job  %d%s. Take info:%s, continue:%t", j.ID, t, err, canWorkOnAnotherJob)
						w.bookedWJobID = 0
						if !canWorkOnAnotherJob {
							errs <- err
						} else {
							continue
						}
					}
				} else if ttl > 0 {
					// If requirements are KO and the ttl > 0, keep alive
					if err := w.client.WorkerSetStatus(ctx, sdk.StatusWaiting); err != nil {
						log.Error("WorkerSetStatus> error on WorkerSetStatus(ctx, sdk.StatusWaiting): %s", err)
					}
					w.bookedWJobID = 0
					log.Debug("Unable to run this job %d%s, requirements not ok. let's continue", j.ID, t)
					continue
				}

				var continueTakeJob = true

				// Is the worker is "single use": unregister and exit the worker
				if w.singleUse {
					continueTakeJob = false
				}

				// If the TTL is null: unregister and exit the worker
				if ttl == 0 {
					continueTakeJob = false
				}

				if continueTakeJob {
					//Continue
					if err := w.client.WorkerSetStatus(ctx, sdk.StatusWaiting); err != nil {
						log.Error("WorkerSetStatus> error on WorkerSetStatus(ctx, sdk.StatusWaiting): %s", err)
					}
					w.bookedWJobID = 0
					continue
				}

				// Unregister from engine
				log.Info("Job is done. Unregistering...")
				cancel()
			}
		}
	}
}

func (w *currentWorker) processBookedWJob(ctx context.Context, wjobs chan<- sdk.WorkflowNodeJobRun) error {
	log.Debug("Try to take the workflow node job %d", w.bookedWJobID)
	wjob, err := w.client.QueueJobInfo(w.bookedWJobID)
	if err != nil {
		return sdk.WrapError(err, "Unable to load workflow node job %d", w.bookedWJobID)
	}

	requirementsOK, errRequirements := checkRequirements(w, &wjob.Job.Action, wjob.ID)
	if !requirementsOK {
		var details string
		for _, r := range errRequirements {
			details += fmt.Sprintf(" %s(%s)", r.Value, r.Type)
		}
		infos := []sdk.SpawnInfo{{
			RemoteTime: time.Now(),
			Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoWorkerForJobError.ID, Args: []interface{}{w.status.Name, details}},
		}}
		if err := w.client.QueueJobSendSpawnInfo(ctx, wjob.ID, infos); err != nil {
			return sdk.WrapError(err, "Cannot record QueueJobSendSpawnInfo for job (err spawn): %d", wjob.ID)
		}
		return fmt.Errorf("processBookedWJob> the worker have no all requirements")
	}

	pluginsOK, errPlugins := checkPluginDeployment(w, *wjob)
	if !pluginsOK {
		var details = errPlugins.Error()

		infos := []sdk.SpawnInfo{{
			RemoteTime: time.Now(),
			Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoWorkerForJobError.ID, Args: []interface{}{w.status.Name, details}},
		}}
		if err := w.client.QueueJobSendSpawnInfo(ctx, wjob.ID, infos); err != nil {
			return sdk.WrapError(err, "Cannot record QueueJobSendSpawnInfo for job (err spawn): %d", wjob.ID)
		}
		return fmt.Errorf("processBookedWJob> the worker have no all plugins")
	}

	// requirementsOK is ok
	wjobs <- *wjob

	return nil
}

func (w *currentWorker) doRegister() error {
	if w.id == "" {
		var info string
		if w.bookedWJobID > 0 {
			info = fmt.Sprintf(", I was born to work on workflow node job %d", w.bookedWJobID)
		}
		log.Info("Registering on CDS engine%s Version:%s", info, sdk.VERSION)

		form := sdk.WorkerRegistrationForm{
			Name:         w.status.Name,
			Token:        w.token,
			HatcheryName: w.hatchery.name,
			ModelID:      w.model.ID,
		}
		if err := w.register(form); err != nil {
			log.Error("Cannot register: %s", err)
			return err
		}
	}
	return nil
}
