package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func statusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Retrieve CDS api status",
		Long:  ``,
		Run:   status,
	}

	return cmd
}

func status(cdm *cobra.Command, args []string) {
	output, err := sdk.GetStatus()
	if err != nil {
		sdk.Exit("Cannot get status (%s)\n", err)
	}

	for _, l := range output {
		fmt.Printf("%s\n", l)
	}
}
