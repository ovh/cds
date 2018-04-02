package application

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

var (
	applicationPipelineCmd = &cobra.Command{
		Use:     "pipeline",
		Short:   "cds application pipeline",
		Aliases: []string{"p"},
	}

	cmdApplicationAddPipeline = &cobra.Command{
		Use:     "add",
		Short:   "cds application pipeline add <projectKey> <applicationName> <pipelineName> [-p <pipelineParam>=<value>]",
		Aliases: []string{"attach"},
		Run:     addApplicationPipeline,
	}
	cmdApplicationAddPipelineParams []string

	cmdApplicationRemovePipeline = &cobra.Command{
		Use:   "remove",
		Short: "cds application pipeline remove <projectKey> <applicationName> <pipelineName>",
		Run:   removeApplicationPipeline,
	}
)

func init() {
	applicationPipelineCmd.AddCommand(cmdApplicationAddPipeline)
	applicationPipelineCmd.AddCommand(cmdApplicationRemovePipeline)
	applicationPipelineCmd.AddCommand(cmdApplicationPipelineScheduler)

	cmdApplicationPipelineScheduler.AddCommand(cmdApplicationPipelineSchedulerList)
	cmdApplicationPipelineScheduler.AddCommand(cmdApplicationPipelineSchedulerAdd)
	cmdApplicationPipelineScheduler.AddCommand(cmdApplicationPipelineSchedulerUpdate)
	cmdApplicationPipelineScheduler.AddCommand(cmdApplicationPipelineSchedulerDelete)

	cmdApplicationAddPipeline.Flags().StringSliceVarP(&cmdApplicationAddPipelineParams, "parameter", "p", nil, "Pipeline parameters")

	cmdApplicationPipelineSchedulerAdd.Flags().StringSliceVarP(&cmdApplicationAddPipelineParams, "parameter", "p", nil, "Pipeline parameters")
	cmdApplicationPipelineSchedulerAdd.Flags().StringVarP(&cmdApplicationPipelineSchedulerAddEnv, "environment", "e", "", "Set environment")

	cmdApplicationPipelineSchedulerUpdate.Flags().StringVarP(&cmdApplicationPipelineSchedulerAddEnv, "environment", "e", "", "Set environment")
	cmdApplicationPipelineSchedulerUpdate.Flags().StringSliceVarP(&cmdApplicationAddPipelineParams, "parameter", "p", nil, "Pipeline parameters")
	cmdApplicationPipelineSchedulerUpdate.Flags().StringVarP(&cmdApplicationPipelineSchedulerUpdateCronExpr, "cron", "c", "", "Set cron expr")
	cmdApplicationPipelineSchedulerUpdate.Flags().StringVarP(&cmdApplicationPipelineSchedulerUpdateDisable, "disable", "", "", "Disable scheduler")
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

	pip, errPip := sdk.GetPipeline(projectKey, pipelineName)
	if errPip != nil {
		sdk.Exit("Error: cannot add pipeline %s in application %s (%s)\n", pipelineName, appName, errPip)
	}

	//Check that all pipeline parameter are set
	var checkParams = true
	for _, p := range pip.Parameter {
		var found bool
		for _, sp := range params {
			if p.Name == sp.Name {
				found = true
				continue
			}
		}
		if !found {
			checkParams = false
			fmt.Printf(" - Missing Pipeline Parameters : %s : %s\n", p.Name, p.Description)
		}
	}
	if !checkParams {
		sdk.Exit("Error: cannot add pipeline %s in application %s: missing parameter(s)\n", pipelineName, appName)
	}

	if err := sdk.AddApplicationPipeline(projectKey, appName, pipelineName); err != nil {
		sdk.Exit("Error: cannot add pipeline %s in application %s (%s)\n", pipelineName, appName, err)
	}

	if err := sdk.UpdateApplicationPipeline(projectKey, appName, pipelineName, params); err != nil {
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

	if err := sdk.RemoveApplicationPipeline(projectKey, appName, pipelineName); err != nil {
		sdk.Exit("Error: cannot remove pipeline %s from project %s (%s)\n", pipelineName, projectKey, err)
	}
	fmt.Println("OK")
}
