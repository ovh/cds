package main

import (
	"fmt"
	"os"

	"github.com/ovh/cds/cli"
	"github.com/spf13/cobra"
)

var (
	adminPluginsActionCmd = cli.Command{
		Name:  "plugins-action",
		Short: "Manage CDS Plugins",
	}

	adminPluginsAction = cli.NewCommand(adminPluginsActionCmd, nil,
		[]*cobra.Command{
			cli.NewCommand(adminPluginsActionAddBinaryCmd, adminPluginsActionAddBinaryFunc, nil),
		},
	)
)

var adminPluginsActionAddBinaryCmd = cli.Command{
	Name:  "binary-add",
	Short: "Add a binary",
	Args: []cli.Arg{
		{
			Name: "filename",
		},
	},
}

func adminPluginsActionAddBinaryFunc(v cli.Values) error {

	f, err := os.Open(v.GetString("filename"))
	if err != nil {
		return fmt.Errorf("unable to open file %s: %v", v.GetString("filename"), err)
	}

	if err := client.ActionAddPlugin(f, v.GetString("filename")); err != nil {
		return err
	}

	return nil
}
