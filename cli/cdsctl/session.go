package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

func session() *cobra.Command {
	cmd := cli.Command{
		Name:  "session",
		Short: "Manage CDS auth sessions",
	}

	return cli.NewCommand(cmd, nil,
		cli.SubCommands{
			cli.NewListCommand(authSessionListCmd, authSessionListRun, nil),
			cli.NewCommand(authSessionDeleteCmd, authSessionDeleteRun, nil),
		},
	)
}

var authSessionListCmd = cli.Command{
	Name:  "list",
	Short: "List your auth sessions for given user",
	OptionalArgs: []cli.Arg{
		{
			Name: "username",
		},
	},
}

func authSessionListRun(v cli.Values) (cli.ListResult, error) {
	username := v.GetString("username")
	if username == "" {
		username = "me"
	}

	consumers, err := client.AuthSessionListByUser(username)
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(consumers), nil
}

var authSessionDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete an auth session",
	OptionalArgs: []cli.Arg{
		{
			Name: "username",
		},
	},
	Args: []cli.Arg{
		{
			Name: "session-id",
		},
	},
}

func authSessionDeleteRun(v cli.Values) error {
	username := v.GetString("username")
	if username == "" {
		username = "me"
	}

	sessionID := v.GetString("session-id")
	if err := client.AuthSessionDelete(username, sessionID); err != nil {
		return err
	}
	fmt.Printf("Session '%s' successfully deleted.\n", sessionID)

	return nil
}
