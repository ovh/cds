package trigger

import (
	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"
)

func deleteParamTriggerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "cds trigger params delete <srcproject>/<srcapp>/<srcpip>[/<env>] <destproject>/<destapp>/<desstpip>[/<destenv>] <paramName> [<paramName...]",
		Long:  `Empties param values`,
		Run:   deleteParamTrigger,
	}

	return cmd
}

func deleteParamTrigger(cmd *cobra.Command, args []string) {

	if len(args) <= 2 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	for _, arg := range args[2:] {

		exists := false
		for i := range cmdParamPipelineTrigger.Parameters {
			if cmdParamPipelineTrigger.Parameters[i].Name == arg {
				cmdParamPipelineTrigger.Parameters[i].Value = ""
				exists = true
				break
			}
		}

		if !exists {
			sdk.Exit("Error: unknown param name %s\n", arg)
		}
	}

	err := sdk.UpdateTrigger(cmdParamPipelineTrigger)
	if err != nil {
		sdk.Exit("Error: failed to update trigger (%s)\n", err)
	}

}
