package main

import (
	"bufio"
	"fmt"
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

	pflags := mainCmd.PersistentFlags()

	pflags.String("log-level", "notice", "Log Level: debug, info, notice, warning, critical")
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

		if cmd.Flag("auto-update").Value.String() == "true" {
			updateCmd(w)(cmd, args)
		}
		toRun := true
		for toRun {
			execWorker()
			if viper.GetBool("single_use") {
				toRun = false
			} else {
				log.Info("Restarting worker...")
			}
		}
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
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		sdk.Exit("Error on starting worker (StdoutPipe)", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		sdk.Exit("Error on starting worker (StderrPipe)", err)
	}

	stdoutreader := bufio.NewReader(stdout)
	stderrreader := bufio.NewReader(stderr)

	outchan := make(chan bool)
	go func() {
		for {
			line, errs := stdoutreader.ReadString('\n')
			if errs != nil {
				stdout.Close()
				close(outchan)
				return
			}
			if line != "" {
				fmt.Printf(line)
			}
		}
	}()

	errchan := make(chan bool)
	go func() {
		for {
			line, errs := stderrreader.ReadString('\n')
			if errs != nil {
				stderr.Close()
				close(errchan)
				return
			}
			if line != "" {
				fmt.Printf(line)
			}

		}
	}()

	if err := cmd.Start(); err != nil {
		log.Error("start err:%s", err)
	}

	<-outchan
	<-errchan
	if err := cmd.Wait(); err != nil {
		log.Error("wait err:%s", err)
	}
}
