package pipeline

import (
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

// pipelineParameterCmd Command to manage parameter on pipeline
var pipelineParameterCmd = &cobra.Command{
	Use:     "parameter",
	Short:   "",
	Long:    ``,
	Aliases: []string{"par"},
}

func init() {
	pipelineParameterCmd.AddCommand(cmdPipelineShowParameter())
	pipelineParameterCmd.AddCommand(cmdPipelineAddParameter())
	pipelineParameterCmd.AddCommand(cmdPipelineUpdateParameter())
	pipelineParameterCmd.AddCommand(cmdPipelineRemoveParameter())
}

func cmdPipelineShowParameter() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "cds pipeline parameter show <projectKey> <pipelineName>",
		Long:  ``,
		Run:   showParameterInPipeline,
	}
	return cmd
}

func showParameterInPipeline(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	pipelineName := args[1]

	parameters, err := sdk.ShowParameterInPipeline(projectKey, pipelineName)
	if err != nil {
		sdk.Exit("Error: cannot show parameters for pipeline %s (%s)\n", pipelineName, err)
	}

	data, err := yaml.Marshal(parameters)
	if err != nil {
		sdk.Exit("Error: cannot format output (%s)\n", err)
	}

	fmt.Println(string(data))
}

func cmdPipelineAddParameter() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "cds pipeline parameter add <projectKey> <pipelineName> <paramName> <paramValue> <paramType> <paramDescription>",
		Long:  ``,
		Run:   addParameterInPipeline,
	}
	return cmd
}

func addParameterInPipeline(cmd *cobra.Command, args []string) {
	if len(args) != 6 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	pipelineName := args[1]
	paramName := args[2]
	paramValue := args[3]
	paramType := args[4]
	paramDescription := args[5]

	err := sdk.AddParameterInPipeline(projectKey, pipelineName, paramName, paramValue, paramType, paramDescription)
	if err != nil {
		sdk.Exit("Error: cannot add parameter %s in pipeline %s (%s)\n", paramName, pipelineName, err)
	}
	fmt.Printf("OK\n")
}

func cmdPipelineUpdateParameter() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "cds pipeline parameter update <projectKey> <pipelineName> <paramName> <newParamName> <paramValue> <paramType> <paramDescription>",
		Long:  ``,
		Run:   updateParameterInPipeline,
	}
	return cmd
}

func updateParameterInPipeline(cmd *cobra.Command, args []string) {
	if len(args) != 7 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	pipelineName := args[1]
	paramName := args[2]
	newParamName := args[3]
	paramValue := args[4]
	paramType := args[5]
	paramDescription := args[6]

	err := sdk.UpdateParameterInPipeline(projectKey, pipelineName, paramName, newParamName, paramValue, paramType, paramDescription)
	if err != nil {
		sdk.Exit("Error: cannot update parameter %s in pipeline %s (%s)\n", paramName, pipelineName, err)
	}
	fmt.Printf("OK\n")
}

func cmdPipelineRemoveParameter() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "cds pipeline parameter remove <projectKey> <pipelineName> <paramName>",
		Long:  ``,
		Run:   removeParameterFromPipeline,
	}
	return cmd
}

func removeParameterFromPipeline(cmd *cobra.Command, args []string) {
	if len(args) != 3 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	pipelineName := args[1]
	paramName := args[2]

	err := sdk.RemoveParameterFromPipeline(projectKey, pipelineName, paramName)
	if err != nil {
		sdk.Exit("Error: cannot remove parameter %s from pipeline %s (%s)\n", paramName, pipelineName, err)
	}
	fmt.Printf("OK\n")
}
