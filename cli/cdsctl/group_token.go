package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var (
	groupTokenCmd = cli.Command{
		Name:  "token",
		Short: "Manage CDS group token",
	}

	groupToken = cli.NewCommand(groupTokenCmd, nil,
		[]*cobra.Command{
			cli.NewGetCommand(groupTokenCreateCmd, groupTokenCreateRun, nil),
		})
)

var groupTokenCreateCmd = cli.Command{
	Name:  "generate",
	Short: "Generate a new token",
	Long: `
		Useful to generate a new token when you use the cli or the api in scripts or for your worker, hatchery, uservices.
		The expiration must be [daily|persistent|session]
		Daily expirate after one day
		Persistent doesn't expirate until you revoke them
	`,
	Args: []cli.Arg{
		{Name: "groupname"},
		{Name: "expiration"},
	},
	OptionalArgs: []cli.Arg{
		{Name: "description"},
	},
}

func groupTokenCreateRun(v cli.Values) (interface{}, error) {
	token, err := client.GroupGenerateToken(v["groupname"], v["expiration"], v.GetString("description"))
	if err != nil {
		return nil, err
	}
	return *token, nil
}
