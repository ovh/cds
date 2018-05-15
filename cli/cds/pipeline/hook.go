package pipeline

import (
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func init() {
	pipelineHookCmd.AddCommand(pipelineListHookCmd())
}

var pipelineHookCmd = &cobra.Command{
	Use:   "hook",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var showURLOnly bool

func pipelineListHookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "cds pipeline hook list <projectKey> <applicationName> <pipelineName>",
		Long:  ``,
		Run:   listPipelineHook,
	}

	cmd.Flags().BoolVarP(&showURLOnly, "show-url-only", "", false, "Shows only URL")

	return cmd
}

func listPipelineHook(cmd *cobra.Command, args []string) {

	if len(args) != 3 {
		sdk.Exit("Wrong usage: See %s\n", cmd.Short)
	}

	pipelineProject := args[0]
	appName := args[1]
	pipelineName := args[2]

	hooks, err := sdk.GetHooks(pipelineProject, appName, pipelineName)
	if err != nil {
		sdk.Exit("Cannot retrieve hooks from %s/%s/%s (%s)\n", pipelineProject, appName, pipelineName, err)
	}

	if showURLOnly {
		for _, h := range hooks {
			fmt.Printf("%s\n", h.Link)
		}
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Repository", "URL"})
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")

	for _, h := range hooks {
		table.Append([]string{
			fmt.Sprintf("%d", h.ID),
			fmt.Sprintf("%s/%s/%s", h.Host, h.Project, h.Repository),
			h.Link,
		})
	}
	table.Render()

}
