package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rockbears/log"
	"github.com/sirupsen/logrus"
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

		// Initialize context
		ctx, cancel := context.WithCancel(context.Background())

		// Setup workerfrom commandline flags or env variables
		cfg, err := initFromFlags(cmd)
		if err != nil {
			log.Fatal(ctx, "%v", err)
		}
		if err := initFromConfig(ctx, cfg, w); err != nil {
			log.Fatal(ctx, "%v", err)
		}

		defer cdslog.Flush(ctx, logrus.StandardLogger())

		// TODO: remove this code with all the flags replaces by config
		// Get the booked job ID
		if cfg.RunJobID == "" && cfg.BookedJobID == 0 {
			bookedWJobID := FlagInt64(cmd, flagBookedWorkflowJobID)
			runJobID := FlagString(cmd, flagRunJobID)
			if bookedWJobID == 0 && runJobID == "" {
				sdk.Exit("flag --booked-workflow-job-id or run-job-id are mandatory")
			}
			cfg.BookedJobID = bookedWJobID
			cfg.RunJobID = runJobID
		}

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
				log.Info(ctx, "Received syscall.SIGTERM")
				cancel()
				return
			case <-ctx.Done():
				return
			}
		}()
		// Start the worker
		if sdk.IsValidUUID(cfg.RunJobID) {
			if err := internal.V2StartWorker(ctx, w, cfg.RunJobID, cfg.Region); err != nil {
				ctx := sdk.ContextWithStacktrace(ctx, err)
				ctx = context.WithValue(ctx, cdslog.RequestID, sdk.ExtractHTTPError(err).RequestID)
				log.Error(ctx, err.Error())
				time.Sleep(2 * time.Second)
				sdk.Exit("error: %v", err)
			}
			return
		}

		if err := internal.StartWorker(ctx, w, cfg.BookedJobID); err != nil {
			ctx := sdk.ContextWithStacktrace(ctx, err)
			ctx = context.WithValue(ctx, cdslog.RequestID, sdk.ExtractHTTPError(err).RequestID)
			log.Error(ctx, err.Error())
			time.Sleep(2 * time.Second)
			sdk.Exit("error: %v", err)
		}
	}
}
