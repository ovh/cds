package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/engine/worker/internal"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func cmdRun() *cobra.Command {
	c := &cobra.Command{
		Use:    "run",
		Hidden: true, // user should not use this command directly
		Run:    runCmd(),
	}

	initFlagsRun(c)
	return c
}

func runCmd() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		var w = new(internal.CurrentWorker)

		//Initialize  context
		ctx := context.Background()

		// Setup workerfrom commandline flags or env variables
		initFromFlags(cmd, w)

		// Get the booked job ID
		bookedWJobID := FlagInt64(cmd, flagBookedWorkflowJobID)

		ctx, cancel := context.WithCancel(ctx)
		// Gracefully shutdown connections
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		defer func() {
			log.Info("Run signal.Stop")
			signal.Stop(c)
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
		// Start the worker
		if err := internal.StartWorker(ctx, w, bookedWJobID); err != nil {
			sdk.Exit("error: %v", err)
		}
	}
}
