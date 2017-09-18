package main

import (
	"fmt"
	"io/ioutil"
	"path"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var (
	actionCmd = cli.Command{
		Name:  "action",
		Short: "Manage CDS action",
	}

	action = cli.NewCommand(actionCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(actionListCmd, actionListRun, nil),
			cli.NewGetCommand(actionShowCmd, actionShowRun, nil),
			cli.NewCommand(actionDeleteCmd, actionDeleteRun, nil),
			cli.NewCommand(actionDocCmd, actionDocRun, nil),
		})
)

var actionListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS actions",
}

func actionListRun(v cli.Values) (cli.ListResult, error) {
	actions, err := client.ActionList()
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(actions), nil
}

var actionShowCmd = cli.Command{
	Name:  "show",
	Short: "Show a CDS action",
	Args: []cli.Arg{
		{Name: "action-name"},
	},
}

func actionShowRun(v cli.Values) (interface{}, error) {
	action, err := client.ActionGet(v["action-name"])
	if err != nil {
		return nil, err
	}
	return *action, nil
}

var actionDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a CDS action",
	Args: []cli.Arg{
		{Name: "action-name"},
	},
}

func actionDeleteRun(v cli.Values) error {
	return client.ActionDelete(v["action-name"])
}

var actionDocCmd = cli.Command{
	Name:  "doc",
	Short: "Generate Action Documentation: cdsctl action doc <path-to-hclFile>",
	Args: []cli.Arg{
		{Name: "path"},
	},
}

func actionDocRun(v cli.Values) error {
	btes, errRead := ioutil.ReadFile(v["path"])
	if errRead != nil {
		return fmt.Errorf("Error while reading file: %s", errRead)
	}

	action, errFrom := sdk.NewActionFromScript(btes)
	if errFrom != nil {
		return fmt.Errorf("Error loading file: %s", errFrom)
	}

	fmt.Println(sdk.ActionInfoMarkdown(action, path.Base(v["path"])))
	return nil
}
