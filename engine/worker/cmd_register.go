package main

import (
	"os"

	"github.com/ovh/cds/sdk/log"
	"github.com/spf13/cobra"
)

func cmdRegister() *cobra.Command {
	var cmdRegister = &cobra.Command{
		Use:   "register",
		Short: "worker register",
		Run:   cmdRegisterRun,
	}
	return cmdRegister
}

func cmdRegisterRun(cmd *cobra.Command, args []string) {
	initViper()
	if err := register(api, name, key); err != nil {
		log.Error("Unable to register worker: %s", err)
		os.Exit(1)
	}
	if err := unregister(); err != nil {
		log.Error("Unable to unregister worker: %s", err)
		os.Exit(1)
	}
}
