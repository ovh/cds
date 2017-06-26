package application

import (
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

var pretty *bool

func applicationListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "cds application list <projectkey>",
		Long:    ``,
		Aliases: []string{"ls", "ps"},
		Run:     listApplications,
	}

	pretty = cmd.Flags().BoolP("pretty", "", false, "display with a pretty table")

	return cmd
}

func listApplications(cmd *cobra.Command, args []string) {
	var apps []sdk.Application
	if len(args) != 1 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	apps, err := sdk.ListApplications(args[0])
	if err != nil {
		sdk.Exit("%s\n", err)
	}

	if *pretty {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Name"})
		table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
		table.SetCenterSeparator("|")

		for _, a := range apps {
			table.Append([]string{a.Name})
		}
		table.Render()
		return
	}

	fmt.Println("NAME")
	for _, app := range apps {
		fmt.Println(app.Name)
	}
}
