package trigger

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

var cmdTriggerAddParams []string
var cmdTriggerAddPrerequisites []string
var cmdTriggerManual bool

func addTriggerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "cds trigger add <srcproject>/<srcapp>/<srcpip>[/<srcenv>] <destproject>/<destapp>/<desstpip>[/<destenv>] [-p <paramName>=<paramValue>] [--prerequisite <pipelineParamName>=<expectedValue>] [--manual]",
		Long:  ``,
		Run:   addTrigger,
	}

	cmd.Flags().BoolVarP(&cmdTriggerManual, "manual", "", false, "Manual Trigger or not")
	cmd.Flags().StringSliceVarP(&cmdTriggerAddParams, "parameter", "p", nil, "Trigger parameter")
	cmd.Flags().StringSliceVarP(&cmdTriggerAddPrerequisites, "prerequisite", "", nil, "Trigger prerequisite")
	return cmd
}

func addTrigger(cmd *cobra.Command, args []string) {

	if len(args) != 2 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	t, err := triggerFromString(args[0], args[1])
	if err != nil {
		sdk.Exit("Error: %s\n", err)
	}

	// Parameters
	for i := range cmdTriggerAddParams {
		p, err := sdk.NewStringParameter(cmdTriggerAddParams[i])
		if err != nil {
			sdk.Exit("Error: cannot parse parameter '%s' (%s)\n", cmdTriggerAddParams[i])
		}
		t.Parameters = append(t.Parameters, p)
	}

	// Prerequisites
	for i := range cmdTriggerAddPrerequisites {
		p, err := sdk.NewPrerequisite(cmdTriggerAddPrerequisites[i])
		if err != nil {
			sdk.Exit("Error: cannot parse parameter '%s' (%s)\n", cmdTriggerAddPrerequisites[i])
		}
		t.Prerequisites = append(t.Prerequisites, p)
	}

	t.Manual = cmdTriggerManual

	err = sdk.AddTrigger(t)
	if err != nil {
		sdk.Exit("Error: %s\n", err)
	}

	fmt.Printf("Trigger created.\n")
}
