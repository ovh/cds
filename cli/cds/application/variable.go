package application

import (
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

var applicationVariableCmd = &cobra.Command{
	Use:     "variable",
	Short:   "",
	Long:    ``,
	Aliases: []string{"v"},
}

var force *bool

func init() {
	applicationVariableCmd.AddCommand(cmdApplicationShowVariable())
	applicationVariableCmd.AddCommand(cmdApplicationAddVariable())
	applicationVariableCmd.AddCommand(cmdApplicationUpdateVariable())
	applicationVariableCmd.AddCommand(cmdApplicationRemoveVariable())
}

func cmdApplicationShowVariable() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "cds application variable show <projectKey> <applicationName>",
		Long:  ``,
		Run:   showVarInApplication,
	}
	return cmd
}

func showVarInApplication(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	appName := args[1]

	variables, err := sdk.ShowApplicationVariable(projectKey, appName)
	if err != nil {
		sdk.Exit("Error: cannot show variables for application %s (%s)\n", appName, err)
	}

	data, err := yaml.Marshal(variables)
	if err != nil {
		sdk.Exit("Error: cannot format output (%s)\n", err)
	}

	fmt.Println(string(data))
}

func cmdApplicationAddVariable() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "cds application variable add <projectKey> <applicationName> <variableName> <variableValue> <variableType>",
		Long:  ``,
		Run:   addApplicationVariable,
	}
	force = cmd.Flags().BoolP("force", "", false, "force update if variable already exist")

	return cmd
}

func addApplicationVariable(cmd *cobra.Command, args []string) {
	var err error
	if len(args) != 5 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	appName := args[1]
	varName := args[2]
	varValue := args[3]
	varType := args[4]

	if *force {
		variables, errSh := sdk.ShowApplicationVariable(projectKey, appName)
		if errSh != nil {
			sdk.Exit("Error: cannot get existing variables for application %s (%s)\n", appName, err)
		}

		varExist := false
		for _, v := range variables {
			if v.Name == varName {
				varExist = true
				break
			}
		}

		if !varExist {
			err = sdk.AddApplicationVariable(projectKey, appName, varName, varValue, varType)
		} else {
			err = sdk.UpdateApplicationVariable(projectKey, appName, varName, varName, varValue, varType)
		}
	} else {
		err = sdk.AddApplicationVariable(projectKey, appName, varName, varValue, varType)
	}

	if err != nil {
		sdk.Exit("Error: cannot add variable %s in application %s (%s)\n", varName, appName, err)
	}

	fmt.Printf("OK\n")
}

func cmdApplicationUpdateVariable() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "cds application variable update <projectKey> <applicationName> <oldVariableName> <variableName> <variableValue> <variableType>",
		Long:  ``,
		Run:   updateApplicationVariable,
	}
	return cmd
}

func updateApplicationVariable(cmd *cobra.Command, args []string) {
	if len(args) != 6 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	appName := args[1]
	oldName := args[2]
	varName := args[3]
	varValue := args[4]
	varType := args[5]

	err := sdk.UpdateApplicationVariable(projectKey, appName, oldName, varName, varValue, varType)
	if err != nil {
		sdk.Exit("Error: cannot update variable %s in application %s (%s)\n", varName, appName, err)
	}
	fmt.Printf("OK\n")
}

func cmdApplicationRemoveVariable() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "cds application variable remove <projectKey> <applicationName> <variableName>",
		Long:  ``,
		Run:   removeApplicationVariable,
	}
	return cmd
}

func removeApplicationVariable(cmd *cobra.Command, args []string) {
	if len(args) != 3 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	projectKey := args[0]
	appName := args[1]
	varName := args[2]

	err := sdk.RemoveApplicationVariable(projectKey, appName, varName)
	if err != nil {
		sdk.Exit("Error: cannot remove variable %s from project %s (%s)\n", varName, projectKey, err)
	}
	fmt.Printf("OK\n")
}
