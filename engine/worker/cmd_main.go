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
	"github.com/spf13/viper"

	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func cmdMain(w *currentWorker) *cobra.Command {
	var mainCmd = &cobra.Command{
		Use:   "worker",
		Short: "CDS Worker",
		Run:   mainCommandRun(w),
	}

	pflags := mainCmd.PersistentFlags()

	pflags.String("log-level", "notice", "Log Level : debug, info, notice, warning, critical")
	viper.BindPFlag("log_level", pflags.Lookup("log-level"))

	pflags.String("api", "", "URL of CDS API")
	viper.BindPFlag("api", pflags.Lookup("api"))

	pflags.String("token", "", "CDS Token")
	viper.BindPFlag("token", pflags.Lookup("token"))

	pflags.String("name", "", "Name of worker")
	viper.BindPFlag("name", pflags.Lookup("name"))

	pflags.Int("model", 0, "Model of worker")
	viper.BindPFlag("model", pflags.Lookup("model"))

	pflags.Int("hatchery", 0, "Hatchery ID spawing worker")
	viper.BindPFlag("hatchery", pflags.Lookup("hatchery"))

	pflags.String("hatchery-name", "", "Hatchery Name spawing worker")
	viper.BindPFlag("hatchery_name", pflags.Lookup("hatchery-name"))

	flags := mainCmd.Flags()

	flags.Bool("single-use", false, "Exit after executing an action")
	viper.BindPFlag("single_use", flags.Lookup("single-use"))

	flags.Bool("force-exit", false, "If single_use=true, force exit. This is useful if it's spawned by an Hatchery (default: worker wait 30min for being killed by hatchery)")
	viper.BindPFlag("force_exit", flags.Lookup("force-exit"))

	flags.String("basedir", "", "This directory (default TMPDIR os environment var) will contains worker working directory and temporary files")
	viper.BindPFlag("basedir", flags.Lookup("basedir"))

	flags.Int("ttl", 30, "Worker time to live (minutes)")
	viper.BindPFlag("ttl", flags.Lookup("ttl"))

	flags.Int64("booked-pb-job-id", 0, "Booked Pipeline Build job id")
	viper.BindPFlag("booked_pb_job_id", flags.Lookup("booked-pb-job-id"))

	flags.Int64("booked-workflow-job-id", 0, "Booked Workflow job id")
	viper.BindPFlag("booked_workflow_job_id", flags.Lookup("booked-workflow-job-id"))

	flags.Int64("booked-job-id", 0, "Booked job id")
	viper.BindPFlag("booked_job_id", flags.Lookup("booked-job-id"))

	flags.String("grpc-api", "", "CDS GRPC tcp address")
	viper.BindPFlag("grpc_api", flags.Lookup("grpc-api"))

	flags.Bool("grpc-insecure", false, "Disable GRPC TLS encryption")
	viper.BindPFlag("grpc_insecure", flags.Lookup("grpc-insecure"))

	flags.String("graylog-protocol", "", "Ex: --graylog-protocol=xxxx-yyyy")
	viper.BindPFlag("graylog_protocol", flags.Lookup("graylog-protocol"))

	flags.String("graylog-host", "", "Ex: --graylog-host=xxxx-yyyy")
	viper.BindPFlag("graylog_host", flags.Lookup("graylog-host"))

	flags.String("graylog-port", "", "Ex: --graylog-port=12202")
	viper.BindPFlag("graylog_port", flags.Lookup("graylog-port"))

	flags.String("graylog-extra-key", "", "Ex: --graylog-extra-key=xxxx-yyyy")
	viper.BindPFlag("graylog_extra_key", flags.Lookup("graylog-extra-key"))

	flags.String("graylog-extra-value", "", "Ex: --graylog-extra-value=xxxx-yyyy")
	viper.BindPFlag("graylog_extra_value", flags.Lookup("graylog-extra-value"))

	return mainCmd
}

func mainCommandRun(w *currentWorker) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		//Initliaze context
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)

		initViper(w)

		if viper.GetString("log_level") == "debug" {
			if err := agent.Listen(nil); err != nil {
				sdk.Exit("Error on starting gops agent", err)
			}
		}

		hostname, errh := os.Hostname() // no check of err here
		if errh != nil {
			hostname = fmt.Sprintf("error compute hostname: %s", errh)
		}
		log.Info("What a good time to be alive, I'm in version %s, my hostname is %s", sdk.VERSION, hostname)
		w.initServer(ctx)

		// Gracefully shutdown connections
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
		defer func() {
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

		time.AfterFunc(time.Duration(viper.GetInt("ttl"))*time.Minute, func() {
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

		// start logger routine with a large buffer
		w.logger.logChan = make(chan sdk.Log, 100000)
		go w.logProcessor(ctx)

		// start queue polling
		pbjobs := make(chan sdk.PipelineBuildJob, 1)
		wjobs := make(chan sdk.WorkflowNodeJobRun, 1)
		errs := make(chan error, 1)

		//Before start the loop, take the bookJobID
		if w.bookedPBJobID != 0 {
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
			w.drainLogsAndCloseLogger(ctx)
			registerTick.Stop()
			w.unregister()
			cancel()

			if viper.GetBool("force_exit") {
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
				if j.ID == 0 {
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
					log.Debug("Unable to run this job, let's continue %d%s", j.ID, t)
					continue
				}

				if !viper.GetBool("single_use") {
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

				//Take the job
				if requirementsOK {
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
					log.Debug("Unable to run this job, let's continue %d%s", j.ID, t)
					continue
				}

				if !viper.GetBool("single_use") {
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
			}
		}
	}
}

func (w *currentWorker) processBookedPBJob(pbjobs chan<- sdk.PipelineBuildJob) {
	log.Debug("Try to take the pipeline build job %d", w.bookedPBJobID)
	b, _, err := sdk.Request("GET", fmt.Sprintf("/queue/%d/infos", w.bookedPBJobID), nil)
	if err != nil {
		log.Error("Unable to load pipeline build job %d: %v", w.bookedPBJobID, err)
		return
	}

	j := &sdk.PipelineBuildJob{}
	if err := json.Unmarshal(b, j); err != nil {
		log.Error("Unable to load pipeline build job %d: %v", w.bookedPBJobID, err)
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

	// requirementsOK is ok
	wjobs <- *wjob
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
		form := worker.RegistrationForm{
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
