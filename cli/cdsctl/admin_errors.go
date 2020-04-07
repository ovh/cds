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
	Short: "Get CDS errors for given request id",
	Args: []cli.Arg{
		{Name: "request-id"},
	},
}

func adminErrorsGetFunc(v cli.Values) error {
	res, err := client.MonErrorsGet(v.GetString("request-id"))
	if err != nil {
		return err
	}

	for i := range res {
		fmt.Printf("Message[%d]: %s\n", i, res[i].Message)
		if res[i].StackTrace != "" {
			fmt.Printf("Stack trace[%d]:\n%s", i, res[i].StackTrace)
		}
	}

	return nil
}
