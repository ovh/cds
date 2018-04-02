package environment

import (
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

var environmentVariableCmd = &cobra.Command{
	Use:     "variable",
	Short:   "",
	Long:    ``,
	Aliases: []string{"v"},
}

var force *bool

func init() {
	environmentVariableCmd.AddCommand(cmdEnvironmentShowVariable())
	environmentVariableCmd.AddCommand(cmdEnvironmentUpdateVariable())
	environmentVariableCmd.AddCommand(cmdEnvironmentRemoveVariable())
}

func cmdEnvironmentShowVariable() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "cds environment variable show <projectKey> <environmentName>",
		Long:  ``,
		Run:   showVarInEnvironment,
	}
	return cmd
}

func showVarInEnvironment(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	envName := args[1]

	variables, err := sdk.ShowEnvironmentVariable(projectKey, envName)
	if err != nil {
		sdk.Exit("Error: cannot show variables for environment %s (%s)\n", envName, err)
	}

	data, err := yaml.Marshal(variables)
	if err != nil {
		sdk.Exit("Error: cannot format output (%s)\n", err)
	}

	fmt.Println(string(data))
}

func cmdEnvironmentUpdateVariable() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "cds environment variable update <projectKey> <environmentName> <oldVariableName> <variableName> <variableValue> <variableType>",
		Long:  ``,
		Run:   updateEnvironmentVariable,
	}
	return cmd
}

func updateEnvironmentVariable(cmd *cobra.Command, args []string) {
	if len(args) != 6 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	envName := args[1]
	varOldName := args[2]
	varName := args[3]
	varValue := args[4]
	varType := args[5]

	if err := sdk.UpdateEnvironmentVariable(projectKey, envName, varOldName, varName, varValue, varType); err != nil {
		sdk.Exit("Error: cannot update variable %s in environment %s (%s)\n", varName, envName, err)
	}
	fmt.Printf("OK\n")
}

func cmdEnvironmentRemoveVariable() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "cds environment variable remove <projectKey> <environmentName> <variableName>",
		Long:  ``,
		Run:   removeEnvironmentVariable,
	}
	return cmd
}

func removeEnvironmentVariable(cmd *cobra.Command, args []string) {
	if len(args) != 3 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	envName := args[1]
	varName := args[2]

	err := sdk.RemoveEnvironmentVariable(projectKey, envName, varName)
	if err != nil {
		sdk.Exit("Error: cannot remove variable %s from project %s (%s)\n", varName, projectKey, err)
	}
	fmt.Printf("OK\n")
}
