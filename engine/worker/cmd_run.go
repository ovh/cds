package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rockbears/log"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/engine/worker/internal"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
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

		if bookedWJobID == 0 {
			sdk.Exit("flag --booked-workflow-job-id is mandatory")
		}

		ctx, cancel := context.WithCancel(ctx)
		// Gracefully shutdown connections
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		defer func() {
			log.Info(ctx, "Run signal.Stop")
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
			ctx := sdk.ContextWithStacktrace(ctx, err)
			ctx = context.WithValue(ctx, cdslog.RequestID, sdk.ExtractHTTPError(err).RequestID)
			log.Error(ctx, err.Error())
			time.Sleep(2 * time.Second)
			sdk.Exit("error: %v", err)
		}
	}
}
