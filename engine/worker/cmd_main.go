package main

import (
	"context"
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

	pflags.String("key", "", "CDS KEY")
	viper.BindPFlag("key", pflags.Lookup("key"))

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

	flags.String("basedir", "", "Worker working directory")
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
		ctx, cancel := context.WithCancel(context.Background())
		w.alive = true
		initViper(w)
		log.Info("What a good time to be alive")
		w.initServer(ctx)

		// Gracefully shutdown connections
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		signal.Notify(c, syscall.SIGTERM)
		signal.Notify(c, syscall.SIGKILL)
		go func() {
			<-c
			if w.grpc.conn != nil {
				log.Warning("Closing GRPC connections")
				w.grpc.conn.Close()
			}
			cancel()
			os.Exit(0)
		}()

		// start logger routine with a large buffer
		w.logChan = make(chan sdk.Log, 10000)
		go w.logger()

		suicideTick := time.NewTicker(time.Duration(viper.GetInt("ttl")) * time.Minute).C
		queuePollingTick := time.NewTicker(4 * time.Second).C
		registerTick := time.NewTicker(10 * time.Second).C

		pbjobs := make(chan sdk.PipelineBuildJob)
		wjobs := make(chan sdk.WorkflowNodeJobRun)
		errs := make(chan error)

		go w.client.QueuePolling(ctx, wjobs, pbjobs, errs, 2*time.Second)

		for {
			if !w.alive && viper.GetBool("single_use") {
				return
			}

			select {
			case j := <-pbjobs:

			case j := <-wjobs:

			case err := <-errs:
				log.Error("%v", err)

			case <-registerTick:
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
						continue
					}
					w.alive = true
				}

			case <-suicideTick:
				if w.nbActionsDone == 0 {
					log.Info("Time to exit.")
					w.unregister()
				}
			case <-queuePollingTick:
				w.queuePolling()
				firstViewQueue = false
			}
		}
	}
}
