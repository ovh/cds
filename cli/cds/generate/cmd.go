package generate

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

var (
	groupname  string
	expiration string
)

// Cmd returns the root cobra command for token generation
func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "token generation for worker and hatchery",
	}

	cmd.AddCommand(workercmd())
	return cmd
}

func workercmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token",
		Short: "cds generate token -g <group> -e <expiration>",
		Long:  "generate a token for worker linked to given group permissions",
		Run:   worker,
	}

	cmd.Flags().StringVarP(&groupname, "group", "g", "", "Group permissions for new token")
	cmd.Flags().StringVarP(&expiration, "expiration", "e", "", "Expiration value for newly created token [session|daily|persistent]")
	return cmd
}

func worker(cmd *cobra.Command, args []string) {
	if groupname == "" {
		sdk.Exit("Error: group name not provided (%s)\n", cmd.Short)
	}
	if expiration == "" {
		sdk.Exit("Error: expiration value not provided (%s)\n", cmd.Short)
	}

	e, err := sdk.ExpirationFromString(expiration)
	if err != nil {
		sdk.Exit("Error: %s\n", err)
	}

	token, err := sdk.GenerateWorkerToken(groupname, e)
	if err != nil {
		sdk.Exit("Error: cannot generate token (%s)\n", err)
	}

	fmt.Printf("%s\n", token.Token)
}
