package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var adminVariableSetCmd = cli.Command{
	Name:    "variableset",
	Aliases: []string{"variablesets", "vs"},
	Short:   "Manage CDS variable sets (admin only)",
}

func adminVariableSets() *cobra.Command {
	return cli.NewCommand(adminVariableSetCmd, nil, []*cobra.Command{
		cli.NewCommand(adminVariableSetCopyCmd, adminVariableSetCopyRun, nil, withAllCommandModifiers()...),
	})
}

var adminVariableSetCopyCmd = cli.Command{
	Name:    "copy",
	Aliases: []string{"cp"},
	Short:   "Copy a variable set and all its items into another project",
	Example: "cdsctl admin variableset copy SRC-PROJECT MY-VARIABLESET DST-PROJECT --name NEW-NAME",
	Args: []cli.Arg{
		{Name: "source-project-key"},
		{Name: "variableset-name"},
		{Name: "target-project-key"},
	},
	Flags: []cli.Flag{
		{Name: "name", Usage: "Name of the copied variable set, defaults to the source name", Type: cli.FlagString},
	},
}

func adminVariableSetCopyRun(v cli.Values) error {
	vs, err := client.AdminProjectVariableSetCopy(context.Background(), v.GetString("source-project-key"), v.GetString("variableset-name"), sdk.CopyVariableSetToProject{
		TargetProjectKey: v.GetString("target-project-key"),
		NewName:          v.GetString("name"),
	})
	if err != nil {
		return err
	}
	fmt.Printf("Variable set %s/%s copied to %s/%s with %d items\n", v.GetString("source-project-key"), v.GetString("variableset-name"), vs.ProjectKey, vs.Name, len(vs.Items))
	return nil
}
