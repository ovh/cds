package application

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

var (
	applicationPipelineCmd = &cobra.Command{
		Use:     "pipeline",
		Short:   "",
		Long:    ``,
		Aliases: []string{"p"},
	}

	cmdApplicationShowPipeline = &cobra.Command{
		Use:   "show",
		Short: "cds application pipeline show <projectKey> <applicationName>",
		Long:  ``,
		Run:   showPipelineInApplication,
	}
	cmdApplicationShowPipelineDetails bool

	cmdApplicationAddPipeline = &cobra.Command{
		Use:     "add",
		Short:   "cds application pipeline add <projectKey> <applicationName> <pipelineName> [-p <pipelineParam>=<value>]",
		Long:    ``,
		Aliases: []string{"attach"},
		Run:     addApplicationPipeline,
	}
	cmdApplicationAddPipelineParams []string

	cmdApplicationRemovePipeline = &cobra.Command{
		Use:   "remove",
		Short: "cds application pipeline remove <projectKey> <applicationName> <pipelineName>",
		Long:  ``,
		Run:   removeApplicationPipeline,
	}
)

func init() {
	applicationPipelineCmd.AddCommand(cmdApplicationShowPipeline)
	applicationPipelineCmd.AddCommand(cmdApplicationAddPipeline)
	applicationPipelineCmd.AddCommand(cmdApplicationRemovePipeline)

	cmdApplicationAddPipeline.Flags().StringSliceVarP(&cmdApplicationAddPipelineParams, "parameter", "p", nil, "Pipeline parameters")
	cmdApplicationShowPipeline.Flags().BoolVarP(&cmdApplicationShowPipelineDetails, "details", "", false, "Show pipeline details")

}

func showPipelineInApplication(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	appName := args[1]

	pipelines, err := sdk.ListApplicationPipeline(projectKey, appName)
	if err != nil {
		sdk.Exit("Error: cannot show pipelines for application %s (%s)\n", appName, err)
	}

	if cmdApplicationShowPipelineDetails {
		data, err := yaml.Marshal(pipelines)
		if err != nil {
			sdk.Exit("Error: cannot format output (%s)\n", err)
		}
		fmt.Println(string(data))
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Pipeline", "Parameters"})
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")

	for i := range pipelines {
		param, err := yaml.Marshal(pipelines[i].Parameter)
		if err != nil {
			sdk.Exit("Error: cannot format output (%s)\n", err)
		}
		table.Append([]string{pipelines[i].Name, string(param)})
	}
	table.Render()

}

func addApplicationPipeline(cmd *cobra.Command, args []string) {
	if len(args) != 3 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	appName := args[1]
	pipelineName := args[2]

	var params []sdk.Parameter
	// Parameters
	for i := range cmdApplicationAddPipelineParams {
		p, err := sdk.NewStringParameter(cmdApplicationAddPipelineParams[i])
		if err != nil {
			sdk.Exit("Error: cannot parse parmeter '%s' (%s)\n", cmdApplicationAddPipelineParams[i])
		}
		params = append(params, p)
	}

	err := sdk.AddApplicationPipeline(projectKey, appName, pipelineName)
	if err != nil {
		sdk.Exit("Error: cannot add pipeline %s in application %s (%s)\n", pipelineName, appName, err)
	}

	err = sdk.UpdateApplicationPipeline(projectKey, appName, pipelineName, params)
	if err != nil {
		sdk.Exit("Error: cannot add pipeline %s in application %s (%s)\n", pipelineName, appName, err)
	}

	fmt.Println("OK")
}

func removeApplicationPipeline(cmd *cobra.Command, args []string) {
	if len(args) != 3 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	appName := args[1]
	pipelineName := args[2]

	err := sdk.RemoveApplicationPipeline(projectKey, appName, pipelineName)
	if err != nil {
		sdk.Exit("Error: cannot remove pipeline %s from project %s (%s)\n", pipelineName, projectKey, err)
	}
	fmt.Println("OK")
}
