package repositoriesmanager

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func listReposManagerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "cds reposmanager list",
		Long:  ``,
		Run:   listReposManager,
	}

	return cmd
}

func listReposManager(cmd *cobra.Command, args []string) {
	rms, err := sdk.GetReposManager()
	if err != nil {
		sdk.Exit("Error: %s\n", err)
	}

	for _, rm := range rms {
		fmt.Printf("%s %s %s\n", rm.Type, rm.Name, rm.URL)
	}
}
