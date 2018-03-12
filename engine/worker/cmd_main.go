package main

import (
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func cmdMain(w *currentWorker) *cobra.Command {
	var mainCmd = &cobra.Command{
		Use:   "worker",
		Short: "CDS Worker",
		Run:   mainCommandRun(w),
	}

	flags := mainCmd.Flags()

	flags.String("log-level", "notice", "Log Level: debug, info, notice, warning, critical")
	viper.BindPFlag("log_level", flags.Lookup("log-level"))

	flags.String("api", "", "URL of CDS API")
	viper.BindPFlag("api", flags.Lookup("api"))

	flags.Bool("insecure", false, `(SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.`)
	viper.BindPFlag("insecure", flags.Lookup("insecure"))

	flags.String("token", "", "CDS Token")
	viper.BindPFlag("token", flags.Lookup("token"))

	flags.String("name", "", "Name of worker")
	viper.BindPFlag("name", flags.Lookup("name"))

	flags.Int("model", 0, "Model of worker")
	viper.BindPFlag("model", flags.Lookup("model"))

	flags.Int("hatchery", 0, "Hatchery ID spawing worker")
	viper.BindPFlag("hatchery", flags.Lookup("hatchery"))

	flags.String("hatchery-name", "", "Hatchery Name spawing worker")
	viper.BindPFlag("hatchery_name", flags.Lookup("hatchery-name"))

	initFlagsRun(mainCmd)

	return mainCmd
}

func initFlagsRun(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.Bool("single-use", false, "Exit after executing an action")
	viper.BindPFlag("single_use", flags.Lookup("single-use"))

	flags.Bool("auto-update", false, "Auto update worker binary from CDS API")
	viper.BindPFlag("auto_update", flags.Lookup("auto-update"))

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
}

func mainCommandRun(w *currentWorker) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		viper.SetEnvPrefix("cds")
		viper.AutomaticEnv()

		// Do not use viper.Get... in this func
		// this will not work, as flags are defined on a sub-command too

		var autoUpdate, singleUse bool
		if os.Getenv("CDS_AUTO_UPDATE") != "" {
			if os.Getenv("CDS_AUTO_UPDATE") == "true" || os.Getenv("CDS_AUTO_UPDATE") == "1" {
				autoUpdate = true
			}
		} else if cmd.Flag("auto-update").Value.String() == "true" {
			autoUpdate = true
		}

		// default false for single use
		singleUse = true
		if os.Getenv("CDS_SINGLE_USE") != "" {
			if os.Getenv("CDS_SINGLE_USE") == "false" || os.Getenv("CDS_SINGLE_USE") == "0" {
				singleUse = false
			}
		} else if cmd.Flag("single-use").Value.String() == "false" {
			singleUse = false
		}

		log.Initialize(&log.Conf{})

		if autoUpdate {
			updateCmd(w)(cmd, args)
		}

		for {
			execWorker()
			if singleUse {
				log.Info("single-use true, worker will be shutdown...")
				break
			} else {
				log.Info("Restarting worker...")
			}
		}
		log.Info("Stopping worker...")
	}
}

func execWorker() {
	current, errExec := os.Executable()
	if errExec != nil {
		sdk.Exit("Error on getting current binary worker", errExec)
	}

	log.Info("Current binary: %s", current)
	args := []string{"run"}
	args = append(args, os.Args[1:]...)
	cmd := exec.Command(current, args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Error("start err:%s", err)
	}

	if err := cmd.Wait(); err != nil {
		log.Error("wait err:%s", err)
	}
}
