package main

import (
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func cmdMain(w *currentWorker) *cobra.Command {
	var mainCmd = &cobra.Command{
		Use:   "worker",
		Short: "CDS Worker",
		Run:   mainCommandRun(w),
	}

	initFlagsRun(mainCmd)

	return mainCmd
}

func mainCommandRun(w *currentWorker) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		var autoUpdate = FlagBool(cmd, flagAutoUpdate)
		var singleUse = FlagBool(cmd, flagSingleUse)

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
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Error("start err:%s", err)
	}

	if err := cmd.Wait(); err != nil {
		log.Error("wait err:%s", err)
	}
}
