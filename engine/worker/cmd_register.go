package main

import (
	"context"

	"github.com/ovh/cds/engine/worker/internal"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk/log"
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
		initFromFlags(cmd, w)

		if err := w.Register(context.Background()); err != nil {
			log.Error("Unable to register worker %v", err)
		}
		if err := w.Unregister(); err != nil {
			log.Error("Unable to unregister worker %v", err)
		}
	}
}
