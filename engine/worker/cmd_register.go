package main

import (
	"context"

	"github.com/ovh/cds/engine/worker/internal"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/rockbears/log"
	"github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

func cmdRegister() *cobra.Command {
	var cmdRegister = &cobra.Command{
		Use:    "register",
		Long:   "worker register is a subcommand used by hatchery. This is not directly useful for end user",
		Hidden: true, // user should not use this command directly
		Run:    cmdRegisterRun(),
	}
	initFlagsRun(cmdRegister)
	return cmdRegister
}

func cmdRegisterRun() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		var w = new(internal.CurrentWorker)

		ctx := context.Background()

		cfg, err := initFromFlags(cmd)
		if err != nil {
			log.Fatal(ctx, "%v", err)
		}

		if err := initFromConfig(ctx, cfg, w); err != nil {
			log.Fatal(ctx, "%v", err)
		}

		defer cdslog.Flush(ctx, logrus.StandardLogger())

		if err := w.Register(ctx); err != nil {
			log.Error(ctx, "Unable to register worker %v", err)
		}
		if err := w.Unregister(ctx); err != nil {
			log.Error(ctx, "Unable to unregister worker %v", err)
		}
	}
}
