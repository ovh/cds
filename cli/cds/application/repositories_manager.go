package application

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

var applicationRepositoriesManagerCmd = &cobra.Command{
	Use:   "reposmanager",
	Short: "",
	Long:  ``,
}

func init() {
	applicationRepositoriesManagerCmd.AddCommand(cmdApplicationAttachReposManager())
	applicationRepositoriesManagerCmd.AddCommand(cmdApplicationDetachReposManager())
}

func cmdApplicationAttachReposManager() *cobra.Command {
	return &cobra.Command{
		Use:   "attach",
		Short: "cds application reposmanager attach <projectKey> <applicationName> <repositories manager> <repository fullname>",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
				sdk.Exit("Wrong usage: %s\n", cmd.Short)
			}
			projectKey := args[0]
			appName := args[1]
			rmName := args[2]
			fullname := args[3]
			if err := sdk.AttachApplicationToReposistoriesManager(projectKey, appName, rmName, fullname); err != nil {
				sdk.Exit("✘ Error: unable to attach application %s (%s) to %s : %s", appName, fullname, rmName, err)
			}
			fmt.Println("✔ Success")
		},
	}
}

func cmdApplicationDetachReposManager() *cobra.Command {
	return &cobra.Command{
		Use:   "detach",
		Short: "cds application reposmanager detach <projectKey> <applicationName> <repositories manager>",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				sdk.Exit("Wrong usage: %s\n", cmd.Short)
			}
			projectKey := args[0]
			appName := args[1]
			rmName := args[2]
			if err := sdk.DetachApplicationToReposistoriesManager(projectKey, appName, rmName); err != nil {
				sdk.Exit("✘ Error: unable to detach application %s to %s : %s", appName, rmName, err)
			}
			fmt.Println("✔ Success")
		},
	}
}
