package main

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var userCmd = cli.Command{
	Name:    "user",
	Aliases: []string{"users"},
	Short:   "Manage CDS user",
}

func usr() *cobra.Command {
	return cli.NewCommand(userCmd, nil, []*cobra.Command{
		cli.NewGetCommand(userMeCmd, userMeRun, nil),
		cli.NewListCommand(userListCmd, userListRun, nil),
		cli.NewGetCommand(userShowCmd, userShowRun, nil),
		userGpg(),
	})
}

var userListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS users",
}

func userListRun(v cli.Values) (cli.ListResult, error) {
	users, err := client.UserList(context.Background())
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(users), nil
}

var userMeCmd = cli.Command{
	Name:  "me",
	Short: "Show Current CDS user details",
}

func userMeRun(v cli.Values) (interface{}, error) {
	u, err := client.UserGetMe(context.Background())
	if err != nil {
		return nil, err
	}
	return u, nil
}

var userShowCmd = cli.Command{
	Name:  "show",
	Short: "Show CDS user details",
	Args: []cli.Arg{
		{
			Name: "username",
		},
	},
}

func userShowRun(v cli.Values) (interface{}, error) {
	u, err := client.UserGet(context.Background(), v.GetString("username"))
	if err != nil {
		return nil, err
	}
	return *u, nil
}
