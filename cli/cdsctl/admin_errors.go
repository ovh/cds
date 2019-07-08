package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var adminErrorsCmd = cli.Command{
	Name:    "errors",
	Aliases: []string{"error"},
	Short:   "Manage CDS errors",
}

func adminErrors() *cobra.Command {
	return cli.NewCommand(adminErrorsCmd, nil, []*cobra.Command{
		cli.NewCommand(adminErrorsGetCmd, adminErrorsGetFunc, nil),
	})
}

var adminErrorsGetCmd = cli.Command{
	Name:  "get",
	Short: "Get CDS error",
	Args: []cli.Arg{
		{Name: "uuid"},
	},
}

func adminErrorsGetFunc(v cli.Values) error {
	res, err := client.MonErrorsGet(v.GetString("uuid"))
	if err != nil {
		return err
	}

	fmt.Printf("Message: %s\n", res.Message)
	if res.StackTrace != "" {
		fmt.Printf("Stack trace:\n%s", res.StackTrace)
	}

	return nil
}
