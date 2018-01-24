package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var (
	environmentCmd = cli.Command{
		Name:  "environment",
		Short: "Manage CDS environment",
		Aliases: []string{
			"env",
		},
	}

	environment = cli.NewCommand(environmentCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(environmentListCmd, environmentListRun, nil, withAllCommandModifiers()...),
			cli.NewCommand(environmentCreateCmd, environmentCreateRun, nil, withAllCommandModifiers()...),
			cli.NewDeleteCommand(environmentDeleteCmd, environmentDeleteRun, nil, withAllCommandModifiers()...),
			environmentKey,
			environmentVariable,
			environmentGroup,
		})
)

var environmentListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS environments",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
}

func environmentListRun(v cli.Values) (cli.ListResult, error) {
	apps, err := client.EnvironmentList(v[_ProjectKey])
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(apps), nil
}

var environmentCreateCmd = cli.Command{
	Name:  "create",
	Short: "Create a CDS environment",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "environment-name"},
	},
	Aliases: []string{"add"},
}

func environmentCreateRun(v cli.Values) error {
	env := &sdk.Environment{Name: v["environment-name"]}
	return client.EnvironmentCreate(v[_ProjectKey], env)
}

var environmentDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a CDS environment",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "environment-name"},
	},
}

func environmentDeleteRun(v cli.Values) error {
	err := client.EnvironmentDelete(v[_ProjectKey], v["environment-name"])
	if err != nil && v.GetBool("force") && sdk.ErrorIs(err, sdk.ErrNoEnvironment) {
		fmt.Println(err.Error())
		os.Exit(0)
	}

	return err
}
