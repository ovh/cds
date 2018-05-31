package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
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
		Short:  "worker run.",
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
		log.Info("auto-update: %t", w.autoUpdate)
		log.Info("single-use: %t", w.singleUse)

		w.initServer(ctx)

		// Gracefully shutdown connections
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
		defer func() {
			log.Info("Run signal.Stop. My hostname is %s", hostname)
			signal.Stop(c)
			cancel()
		}()

		go func() {
			select {
			case <-c:
				defer cancel()
				return
			case <-ctx.Done():
				return
			}
		}()

		time.AfterFunc(time.Duration(FlagInt(cmd, flagTTL))*time.Minute, func() {
			if w.nbActionsDone == 0 {
				log.Debug("Suicide")
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

		//Register every 10 seconds if we have nothing to do
		registerTick := time.NewTicker(10 * time.Second)

		updateTick := time.NewTicker(5 * time.Minute)

		// start logger routine with a large buffer
		w.logger.logChan = make(chan sdk.Log, 100000)
		go w.logProcessor(ctx)

		// start queue polling
		var pbjobs chan sdk.PipelineBuildJob
		if !w.disableOldWorkflows {
			pbjobs = make(chan sdk.PipelineBuildJob, 1)
		}
		wjobs := make(chan sdk.WorkflowNodeJobRun, 1)
		errs := make(chan error, 1)

		//Before start the loop, take the bookJobID
		if !w.disableOldWorkflows && w.bookedPBJobID != 0 {
			w.processBookedPBJob(pbjobs)
		}
		if w.bookedWJobID != 0 {
			w.processBookedWJob(wjobs)
		}
		if err := w.client.WorkerSetStatus(sdk.StatusWaiting); err != nil {
			log.Error("WorkerSetStatus> error on WorkerSetStatus(sdk.StatusWaiting): %s", err)
		}

		go func(ctx context.Context) {
			if err := w.client.QueuePolling(ctx, wjobs, pbjobs, errs, 2*time.Second, 0); err != nil {
				log.Error("Queues polling stopped: %v", err)
			}
		}(ctx)

		//Definition of the function which must be called to stop the worker
		var endFunc = func() {
			log.Info("Enter endFunc")
			w.drainLogsAndCloseLogger(ctx)
			registerTick.Stop()
			updateTick.Stop()
			w.unregister()
			cancel()

			if FlagBool(cmd, flagForceExit) {
				log.Info("Exiting worker with force_exit true")
				return
			}

			if w.hatchery.id > 0 {
				log.Info("Waiting 30min to be killed by hatchery, if not killed, worker will exit")
				time.Sleep(30 * time.Minute)
			}

			if err := ctx.Err(); err != nil {
				log.Error("Exiting worker: %v", err)
			} else {
				log.Info("Exiting worker")
			}
		}

		go func(errs chan error) {
			for {
				select {
				case err := <-errs:
					log.Error("An error has occured: %v", err)
				}
			}
		}(errs)

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

			case j := <-pbjobs:
				if w.disableOldWorkflows || j.ID == 0 {
					continue
				}

				requirementsOK, _ := checkRequirements(w, &j.Job.Action, j.ExecGroups, j.ID)

				t := ""
				if j.ID == w.bookedPBJobID {
					t = ", this was my booked job"
				}

				//Take the job
				if requirementsOK {
					log.Debug("checkQueue> Try take the PipelineBuildJob %d%s", j.ID, t)
					canWorkOnAnotherJob := w.takePipelineBuildJob(ctx, j.ID, j.ID == w.bookedPBJobID)
					if canWorkOnAnotherJob {
						continue
					}
				} else {
					if err := w.client.WorkerSetStatus(sdk.StatusWaiting); err != nil {
						log.Error("WorkerSetStatus> error on WorkerSetStatus(sdk.StatusWaiting): %s", err)
					}
					log.Debug("Unable to run this pipeline build job, requirements not OK, let's continue %d%s", j.ID, t)
					continue
				}

				if !w.singleUse {
					//Continue
					log.Debug("PipelineBuildJob is done. single_use to false, keep worker alive")
					if err := w.client.WorkerSetStatus(sdk.StatusWaiting); err != nil {
						log.Error("WorkerSetStatus> error on WorkerSetStatus(sdk.StatusWaiting): %s", err)
					}
					continue
				}

				// Unregister from engine and stop the register goroutine
				log.Info("PipelineBuildJob is done. Unregistering...")
				cancel()
			case j := <-wjobs:
				if j.ID == 0 {
					continue
				}

				requirementsOK, _ := checkRequirements(w, &j.Job.Action, nil, j.ID)
				t := ""
				if j.ID == w.bookedWJobID {
					t = ", this was my booked job"
				}

				pluginsOK, errPlugins := checkPlugins(w, j)
				if !pluginsOK {
					log.Error("Plugins doesn't match: %v", errPlugins)
				}

				//Take the job
				if requirementsOK && pluginsOK {
					log.Debug("checkQueue> Try take the job %d%s", j.ID, t)
					if canWorkOnAnotherJob, err := w.takeWorkflowJob(ctx, j); err != nil {
						log.Info("Unable to run this job  %d%s. Take info:%s, continue:%t", j.ID, t, err, canWorkOnAnotherJob)
						if !canWorkOnAnotherJob {
							errs <- err
						} else {
							continue
						}
					}
				} else {
					if err := w.client.WorkerSetStatus(sdk.StatusWaiting); err != nil {
						log.Error("WorkerSetStatus> error on WorkerSetStatus(sdk.StatusWaiting): %s", err)
					}
					log.Debug("Unable to run this workflow job, requirements not ok, let's continue %d%s", j.ID, t)
					continue
				}

				if !w.singleUse {
					//Continue
					log.Debug("Job is done. single_use to false, keep worker alive")
					if err := w.client.WorkerSetStatus(sdk.StatusWaiting); err != nil {
						log.Error("WorkerSetStatus> error on WorkerSetStatus(sdk.StatusWaiting): %s", err)
					}
					continue
				}

				// Unregister from engine
				log.Info("Job is done. Unregistering...")
				cancel()
			case <-registerTick.C:
				w.doRegister()
			case <-updateTick.C:
				w.doUpdate()
			}
		}
	}
}

func (w *currentWorker) processBookedPBJob(pbjobs chan<- sdk.PipelineBuildJob) {
	log.Debug("Try to take the pipeline build job %d", w.bookedPBJobID)
	b, _, err := sdk.Request("GET", fmt.Sprintf("/queue/%d/infos", w.bookedPBJobID), nil)
	if err != nil {
		log.Error("Unable to load pipeline build job %d: %v on Request", w.bookedPBJobID, err)
		return
	}

	j := &sdk.PipelineBuildJob{}
	if err := json.Unmarshal(b, j); err != nil {
		log.Error("Unable to load pipeline build job %d: %v on Unmarshal", w.bookedPBJobID, err)
		return
	}

	requirementsOK, errRequirements := checkRequirements(w, &j.Job.Action, j.ExecGroups, w.bookedPBJobID)
	if !requirementsOK {
		var details string
		for _, r := range errRequirements {
			details += fmt.Sprintf(" %s(%s)", r.Value, r.Type)
		}
		infos := []sdk.SpawnInfo{{
			RemoteTime: time.Now(),
			Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoWorkerForJobError.ID, Args: []interface{}{w.status.Name, details}},
		}}
		if err := sdk.AddSpawnInfosPipelineBuildJob(j.ID, infos); err != nil {
			log.Warning("Cannot record AddSpawnInfosPipelineBuildJob for job (err spawn): %d %s", j.ID, err)
		}
		return
	}

	// requirementsOK is ok
	pbjobs <- *j
}

func (w *currentWorker) processBookedWJob(wjobs chan<- sdk.WorkflowNodeJobRun) {
	log.Debug("Try to take the workflow node job %d", w.bookedWJobID)
	wjob, err := w.client.QueueJobInfo(w.bookedWJobID)
	if err != nil {
		log.Error("Unable to load workflow node job %d: %v", w.bookedWJobID, err)
		return
	}

	requirementsOK, errRequirements := checkRequirements(w, &wjob.Job.Action, nil, wjob.ID)
	if !requirementsOK {
		var details string
		for _, r := range errRequirements {
			details += fmt.Sprintf(" %s(%s)", r.Value, r.Type)
		}
		infos := []sdk.SpawnInfo{{
			RemoteTime: time.Now(),
			Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoWorkerForJobError.ID, Args: []interface{}{w.status.Name, details}},
		}}
		if err := w.client.QueueJobSendSpawnInfo(true, wjob.ID, infos); err != nil {
			log.Warning("Cannot record QueueJobSendSpawnInfo for job (err spawn): %d %s", wjob.ID, err)
		}
		return
	}

	pluginsOK, errPlugins := checkPlugins(w, *wjob)
	if !pluginsOK {
		var details = errPlugins.Error()

		infos := []sdk.SpawnInfo{{
			RemoteTime: time.Now(),
			Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoWorkerForJobError.ID, Args: []interface{}{w.status.Name, details}},
		}}
		if err := w.client.QueueJobSendSpawnInfo(true, wjob.ID, infos); err != nil {
			log.Warning("Cannot record QueueJobSendSpawnInfo for job (err spawn): %d %s", wjob.ID, err)
		}
		return
	}

	// requirementsOK is ok
	wjobs <- *wjob
}

func (w *currentWorker) doUpdate() {
	if w.autoUpdate {
		version, err := w.client.Version()
		if err != nil {
			log.Error("Error while getting version from CDS API: %s", err)
		}
		if version.Version != sdk.VERSION {
			sdk.Exit("Exiting this CDS Worker process - auto updating worker")
		}
	}
}

func (w *currentWorker) doRegister() error {
	if w.id == "" {
		var info string
		if w.bookedPBJobID > 0 {
			info = fmt.Sprintf(", I was born to work on pipeline build job %d", w.bookedPBJobID)
		}
		if w.bookedWJobID > 0 {
			info = fmt.Sprintf(", I was born to work on workflow node job %d", w.bookedWJobID)
		}
		log.Info("Registering on CDS engine%s Version:%s", info, sdk.VERSION)
		form := sdk.WorkerRegistrationForm{
			Name:         w.status.Name,
			Token:        w.token,
			Hatchery:     w.hatchery.id,
			HatcheryName: w.hatchery.name,
			ModelID:      w.model.ID,
		}
		if err := w.register(form); err != nil {
			log.Info("Cannot register: %s", err)
			return err
		}
		log.Debug("I am registered, with groupID:%d and model:%v", w.groupID, w.model)
	}
	return nil
}
