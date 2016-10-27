package trigger

import (
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"
)

func updateParamTriggerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "cds trigger update <srcproject>/<srcapp>/<srcpip>[/<env>] <destproject>/<destapp>/<desstpip>[/<destenv>] <paramName>=<paramValue> [<paramName>=<paramValue>...]",
		Long:  `Updates param values`,
		Run:   updateParamTrigger,
	}

	return cmd
}

func updateParamTrigger(cmd *cobra.Command, args []string) {

	if len(args) <= 2 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	for _, arg := range args[2:] {

		splitted := strings.Split(arg, "=")
		if len(splitted) != 2 {
			sdk.Exit("Invalid argument : format should be <paramName>=<paramValue> : %s\n", arg)
		}

		exists := false
		for i := range cmdParamPipelineTrigger.Parameters {
			if cmdParamPipelineTrigger.Parameters[i].Name == splitted[0] {
				cmdParamPipelineTrigger.Parameters[i].Value = splitted[1]
				exists = true
				break
			}
		}

		if !exists {
			sdk.Exit("Error: unknown param name %s\n", splitted[0])
		}
	}

	err := sdk.UpdateTrigger(cmdParamPipelineTrigger)
	if err != nil {
		sdk.Exit("Error: failed to update trigger (%s)\n", err)
	}

}
