package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

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

		w.alive = true
		initViper(w)
		log.Info("What a good time to be alive")
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
			log.Debug("Suicide")
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
		if w.bookedJobID != 0 {
			w.processBookedJob(pbjobs)
		}

		go func(ctx context.Context) {
			if err := w.client.QueuePolling(ctx, wjobs, pbjobs, errs, 2*time.Second); err != nil {
				log.Error("Queues polling stopped: %v", err)
			}
		}(ctx)

		// main loop
		for {
			if ctx.Err() != nil {
				w.drainLogsAndCloseLogger(ctx)
				w.unregister()
				log.Info("Exiting worker on error: %v", ctx.Err())
				if viper.GetBool("force_exit") {
					return
				}
				if w.hatchery.id > 0 {
					log.Info("Waiting 30min to be killed by hatchery, if not killed, worker will exit")
					time.Sleep(30 * time.Minute)
				}
				return
			}

			if !w.alive && viper.GetBool("single_use") {
				registerTick.Stop()
				defer cancel()
				w.drainLogsAndCloseLogger(ctx)
				if viper.GetBool("force_exit") {
					return
				}
				if w.hatchery.id > 0 {
					log.Info("Waiting 30min to be killed by hatchery, if not killed, worker will exit")
					time.Sleep(30 * time.Minute)
				}
				log.Info("Exiting single-use worker")
				return
			}

			select {
			case <-ctx.Done():
				if err := ctx.Err(); err != nil {
					log.Error("Exiting worker: %v", err)
				} else {
					log.Info("Exiting worker")
				}
				w.drainLogsAndCloseLogger(ctx)
				registerTick.Stop()
				w.unregister()
				return

			case j := <-pbjobs:
				if j.ID == 0 {
					continue
				}

				requirementsOK, _ := checkRequirements(w, &j.Job.Action)

				t := ""
				if j.ID == w.bookedJobID {
					t = ", this was my booked job"
				}

				//Take the job
				if requirementsOK {
					log.Info("checkQueue> Taking job %d%s", j.ID, t)
					w.takePipelineBuildJob(ctx, j.ID, j.ID == w.bookedJobID)
				} else {
					log.Debug("Unable to run this job, let's continue %d%s", j.ID, t)
					continue
				}

				if !viper.GetBool("single_use") {
					//Continue
					w.client.WorkerSetStatus(sdk.StatusWaiting)
					continue
				}

				// Unregister from engine and stop the register goroutine
				registerTick.Stop()
				log.Debug("Job is done. Unregistering...")
				if err := w.unregister(); err != nil {
					log.Warning("takeJob> could not unregister: %s", err)
				}

			case j := <-wjobs:
				if j.ID == 0 {
					continue
				}

				requirementsOK, _ := checkRequirements(w, &j.Job.Action)
				t := ""
				if j.ID == w.bookedJobID {
					t = ", this was my booked job"
				}

				//Take the job
				if requirementsOK {
					log.Info("checkQueue> Taking job %d%s", j.ID, t)
					if err := w.takeWorkflowJob(ctx, j); err != nil {
						errs <- err
					}
				} else {
					log.Debug("Unable to run this job, let's continue %d%s", j.ID, t)
					continue
				}

				if !viper.GetBool("single_use") {
					//Continue
					w.client.WorkerSetStatus(sdk.StatusWaiting)
					continue
				}

				// Unregister from engine
				log.Debug("Job is done. Unregistering...")
				if err := w.unregister(); err != nil {
					log.Warning("takeJob> could not unregister: %s", err)
				}

			case err := <-errs:
				log.Error("%v", err)

			case <-registerTick.C:
				w.doRegister()
			}
		}
	}
}

func (w *currentWorker) processBookedJob(pbjobs chan<- sdk.PipelineBuildJob) {
	log.Debug("Try to take the job %d", w.bookedJobID)
	b, _, err := sdk.Request("GET", fmt.Sprintf("/queue/%d/infos", w.bookedJobID), nil)
	if err != nil {
		log.Error("Unable to load pipeline build job %d: %v", w.bookedJobID, err)
		return
	}

	j := &sdk.PipelineBuildJob{}
	if err := json.Unmarshal(b, j); err != nil {
		log.Error("Unable to load pipeline build job %d: %v", w.bookedJobID, err)
		return
	}

	requirementsOK, errRequirements := checkRequirements(w, &j.Job.Action)
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

func (w *currentWorker) doRegister() error {
	if w.id == "" {
		var info string
		if w.bookedJobID > 0 {
			info = fmt.Sprintf(", I was born to work on job %d", w.bookedJobID)
		}
		log.Info("Registering on CDS engine%s", info)
		form := worker.RegistrationForm{
			Name:         w.status.Name,
			Token:        w.token,
			Hatchery:     w.hatchery.id,
			HatcheryName: w.hatchery.name,
			Model:        w.modelID,
		}
		if err := w.register(form); err != nil {
			log.Info("Cannot register: %s", err)
			return err
		}
		w.alive = true
	}
	return nil
}
