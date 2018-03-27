package action

import (
	"fmt"
	"strings"

	"github.com/ovh/cds/sdk"

	"github.com/spf13/cobra"
)

var cmdActionAddParams = struct {
	Params       []string
	Requirements []string
}{}

func cmdActionAdd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "cds action add [<actionName>]",
		Long: `cds action add [<actionName>]

<actionName> is mandatory

		`,
		Run: addAction,
	}

	cmd.Flags().StringSliceVarP(&cmdActionAddParams.Params, "parameter", "p", nil, "Action parameters")
	cmd.Flags().StringSliceVarP(&cmdActionAddParams.Requirements, "requirement", "r", nil, "Action requirements")

	cmd.AddCommand(cmdActionAddRequirement())
	cmd.AddCommand(cmdActionAddStep())
	return cmd
}

func addAction(cmd *cobra.Command, args []string) {
	name := ""
	if len(args) == 0 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	} else if len(args) > 0 {
		name = args[0]
	}

	var req []sdk.Requirement
	for _, r := range cmdActionAddParams.Requirements {
		req = append(req, sdk.Requirement{
			Name:  r,
			Type:  sdk.BinaryRequirement,
			Value: r,
		})
	}

	if err := sdk.AddAction(name, getCmdParameters(), req); err != nil {
		sdk.Exit("%s\n", err)
	}

	fmt.Printf("OK\n")
}

func getCmdParameters() []sdk.Parameter {
	parameters := []sdk.Parameter{}
	for _, p := range cmdActionAddParams.Params {
		p := sdk.Parameter{
			Name: p,
			Type: sdk.StringParameter,
		}
		parameters = append(parameters, p)
	}
	return parameters
}

func cmdActionAddRequirement() *cobra.Command {
	cmd := &cobra.Command{
		Use: "requirement",
		Run: addActionRequirement,
	}
	return cmd
}

func addActionRequirement(cmd *cobra.Command, args []string) {
}

var cmdActionAddStepParams []string

func cmdActionAddStep() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "step",
		Short: "cds action add step <actionName> <childaction> [-p <paramName>=<paramValue>]",
		Run:   addActionStep,
	}

	cmd.Flags().StringSliceVarP(&cmdActionAddStepParams, "parameter", "p", nil, "Action parameters")
	return cmd
}

func addActionStep(cmd *cobra.Command, args []string) {

	if len(args) != 2 {
		sdk.Exit("Wrong usage. See '%s'\n", cmd.Short)
	}

	actionName := args[0]
	childAction := args[1]

	child, errGet := sdk.GetAction(childAction)
	if errGet != nil {
		sdk.Exit("Error: Cannot retrieve action %s (%s)\n", childAction, errGet)
	}

	for _, p := range cmdActionAddStepParams {
		t := strings.SplitN(p, "=", 2)
		if len(t) != 2 {
			sdk.Exit("Error: invalid parameter format (%s)", p)
		}
		found := false
		for i := range child.Parameters {
			if t[0] == child.Parameters[i].Name {
				found = true
				child.Parameters[i].Value = t[1]
				break
			}
		}
		if !found {
			sdk.Exit("Error: Argument %s does not exists in action %s\n", t[0], child.Name)
		}
	}

	if err := sdk.AddActionStep(actionName, child); err != nil {
		sdk.Exit("Error: Cannot add step %s in action %s (%s)\n", childAction, actionName, err)
	}

	return
}
