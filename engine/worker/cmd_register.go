package main

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
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
		form := sdk.WorkerRegistrationForm{
			Name:         w.status.Name,
			Token:        w.token,
			Hatchery:     w.hatchery.id,
			HatcheryName: w.hatchery.name,
			ModelID:      w.model.ID,
		}

		if err := w.register(form, false); err != nil {
			log.Error("Unable to register worker %s: %v", w.status.Name, err)
		}
		if err := w.unregister(); err != nil {
			log.Error("Unable to unregister worker %s: %v", w.status.Name, err)
		}

		if viper.GetBool("force_exit") {
			log.Info("Exiting worker with force_exit true")
			return
		}

		if w.hatchery.id > 0 {
			log.Info("Waiting 30min to be killed by hatchery, if not killed, worker will exit")
			time.Sleep(30 * time.Minute)
		}
	}
}
