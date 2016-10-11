package project

import (
	"os"

	"github.com/ovh/cds/sdk"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var cmdProjectList = &cobra.Command{
	Use:     "list",
	Short:   "",
	Long:    ``,
	Aliases: []string{"ls"},
	Run:     listProject,
}

func listProject(cmd *cobra.Command, args []string) {
	projects, err := sdk.ListProject()
	if err != nil {
		sdk.Exit("Error: cannot list project (%s)\n", err)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Key", "Name"})
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")

	for i := range projects {
		table.Append([]string{projects[i].Key, projects[i].Name})
	}
	table.Render()
}
