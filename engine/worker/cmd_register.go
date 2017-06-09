package main

import (
	"os"

	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk/log"
	"github.com/spf13/cobra"
)

func cmdRegister(w *currentWorker) *cobra.Command {
	var cmdRegister = &cobra.Command{
		Use:   "register",
		Short: "worker register",
		Run:   cmdRegisterRun(w),
	}
	return cmdRegister
}

func cmdRegisterRun(w *currentWorker) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		initViper(w)
		form := worker.RegistrationForm{
			Name:         w.status.Name,
			Token:        w.token,
			Hatchery:     w.hatchery.id,
			HatcheryName: w.hatchery.name,
			Model:        w.modelID,
		}
		if err := w.register(form); err != nil {
			log.Error("Unable to register worker: %s", err)
			os.Exit(1)
		}
		if err := w.unregister(); err != nil {
			log.Error("Unable to unregister worker: %s", err)
			os.Exit(1)
		}
	}
}
