package application

import (
	"fmt"
	"gopkg.in/yaml.v2"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

var applicationPipelineCmd = &cobra.Command{
	Use:     "pipeline",
	Short:   "",
	Long:    ``,
	Aliases: []string{"p"},
}

func init() {
	applicationPipelineCmd.AddCommand(cmdApplicationShowPipeline())
	applicationPipelineCmd.AddCommand(cmdApplicationAddPipeline())
	applicationPipelineCmd.AddCommand(cmdApplicationRemovePipeline())
}

func cmdApplicationShowPipeline() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "cds application pipeline show <projectKey> <applicationName>",
		Long:  ``,
		Run:   showPipelineInApplication,
	}
	return cmd
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

	data, err := yaml.Marshal(pipelines)
	if err != nil {
		sdk.Exit("Error: cannot format output (%s)\n", err)
	}

	fmt.Println(string(data))
}

var cmdApplicationAddPipelineParams []string

func cmdApplicationAddPipeline() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add",
		Short:   "cds application pipeline add <projectKey> <applicationName> <pipelineName> [-p <pipelineParam>=<value>]",
		Long:    ``,
		Aliases: []string{"attach"},
		Run:     addApplicationPipeline,
	}
	cmd.Flags().StringSliceVarP(&cmdApplicationAddPipelineParams, "parameter", "p", nil, "Pipeline parameters")
	return cmd
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

func cmdApplicationRemovePipeline() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "cds application pipeline remove <projectKey> <applicationName> <pipelineName>",
		Long:  ``,
		Run:   removeApplicationPipeline,
	}
	return cmd
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
