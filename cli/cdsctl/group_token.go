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
	Args: []cli.Arg{
		{Name: "groupname"},
		{Name: "expiration"},
	},
}

func groupTokenCreateRun(v cli.Values) (interface{}, error) {
	token, err := client.GroupGenerateToken(v["groupname"], v["expiration"])
	if err != nil {
		return nil, err
	}
	return *token, nil
}
