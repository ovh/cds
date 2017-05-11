package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func cmdMain() *cobra.Command {
	var mainCmd = &cobra.Command{
		Use:   "worker",
		Short: "CDS Worker",
		Run:   mainCommandRun,
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

func mainCommandRun(cmd *cobra.Command, args []string) {
	initViper()
	log.Info("What a good time to be alive")

	alive = true

	basedir = viper.GetString("basedir")
	if basedir == "" {
		basedir = os.TempDir()
	}

	bookedJobID = viper.GetInt64("booked_job_id")

	initServer()

	// Gracefully shutdown connections
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	signal.Notify(c, syscall.SIGKILL)
	go func() {
		<-c
		if grpcConn != nil {
			log.Warning("Closing GRPC connections")
			grpcConn.Close()
		}
		os.Exit(0)
	}()

	// start logger routine with a large buffer
	logChan = make(chan sdk.Log, 10000)
	go logger(logChan)

	suicideTick := time.NewTicker(time.Duration(viper.GetInt("ttl")) * time.Minute).C
	queuePollingTick := time.NewTicker(4 * time.Second).C
	registerTick := time.NewTicker(1 * time.Second).C

	for {
		if !alive && viper.GetBool("single_use") {
			return
		}

		select {
		case <-registerTick:
			if WorkerID == "" {
				var info string
				if bookedJobID > 0 {
					info = fmt.Sprintf(", I was born to work on job %d", bookedJobID)
				}
				log.Info("Registering on CDS engine%s", info)
				if err := register(api, name, key); err != nil {
					log.Info("Cannot register: %s", err)
					continue
				}
				alive = true
			}

		case <-suicideTick:
			if nbActionsDone == 0 {
				log.Info("Time to exit.")
				unregister()
			}

		case <-queuePollingTick:
			queuePolling()
			firstViewQueue = false
		}
	}

}
