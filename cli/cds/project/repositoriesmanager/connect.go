package repositoriesmanager

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func connectReposManagerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connect",
		Short: "cds project reposmanager connect <project key> <repositories manager>",
		Long:  ``,
		Run:   connectReposManager,
	}

	return cmd
}

func connectReposManager(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	rmName := args[1]
	token, url, err := sdk.ConnectReposManager(projectKey, rmName)
	if err != nil {
		sdk.Exit("✘ Error: %s\n", err)
	}
	fmt.Printf("Go to the following link in your browser\n - %s\n", url)

	if strings.HasPrefix(url, "https://github.com") {
		fmt.Println("And follow instructions.")
		os.Exit(0)
	}

	// scan for user input of response
	var verifier string
	fmt.Println("Enter verification code ?")
	fmt.Scan(&verifier)

	access, secret, err := sdk.ConnectReposManagerCallback(projectKey, rmName, token, verifier)
	if err != nil {
		sdk.Exit("✘ Error: %s\n", err)
	}
	fmt.Printf("✔ Connection successful to %s \n - access token: %s\n - secret: %s\n", rmName, access, secret)
}
