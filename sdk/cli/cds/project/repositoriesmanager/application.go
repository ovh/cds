package repositoriesmanager

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func addApplicationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-application",
		Short: "cds project reposmanager add-application <project key> <repositories manager> <repository fullname>",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				sdk.Exit("Wrong usage: %s\n", cmd.Short)
			}

			if err := sdk.AddApplicationFromReposManager(args[0], args[1], args[2]); err != nil {
				sdk.Exit("✘ Error: %s\n", err)
			}
			fmt.Println("✔ Success")

		},
	}

	return cmd
}
