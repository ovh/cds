package trigger

import (
	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"
)

var cmdParamPipelineTrigger *sdk.PipelineTrigger

func paramTriggerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "list, update, delete trigger params",
		Long:  ``,
	}

	cmd.PersistentPreRun = paramPreRun

	cmd.AddCommand(listParamTriggerCmd())
	cmd.AddCommand(updateParamTriggerCmd())
	cmd.AddCommand(deleteParamTriggerCmd())

	return cmd
}

func paramPreRun(cmd *cobra.Command, args []string) {

	if len(args) < 2 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	cmdParamPipelineTrigger = retrieveTrigger(args[0], args[1])
}
