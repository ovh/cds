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
		Run:   listReposManager,
	}

	return cmd
}

func listReposManager(cmd *cobra.Command, args []string) {
	if ok, err := sdk.IsAdmin(); !ok {
		if err != nil {
			fmt.Printf("Error : %v\n", err)
		}
		sdk.Exit("You are not allowed to run this command")
	}

	rms, err := sdk.GetReposManager()
	if err != nil {
		sdk.Exit("Error: %s\n", err)
	}

	for _, rm := range rms {
		fmt.Println(rm)
	}
}
