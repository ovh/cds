package trigger

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func deleteTriggerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete",
		Short:   "cds trigger delete <srcproject>/<srcapp>/<srcpip>[/<srcenv>] <destproject>/<destapp>/<desstpip>[/<destenv>]",
		Long:    ``,
		Aliases: []string{"remove", "rm"},
		Run:     deleteTrigger,
	}

	return cmd
}

func deleteTrigger(cmd *cobra.Command, args []string) {

	if len(args) != 2 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	t, err := triggerFromString(args[0], args[1])
	if err != nil {
		sdk.Exit("Error: %s\n", err)
	}

	triggers, err := sdk.GetTriggers(t.SrcProject.Key, t.SrcApplication.Name, t.SrcPipeline.Name, t.SrcEnvironment.Name)
	if err != nil {
		sdk.Exit("Error: cannot retrieve triggers (%s)\n", err)
	}

	var found = false
	for _, tr := range triggers {
		if triggersEqual(t, &tr) {
			err = sdk.DeleteTrigger(t.DestProject.Key, t.DestApplication.Name, t.DestPipeline.Name, tr.ID)
			if err != nil {
				sdk.Exit("Error: cannot delete trigger (%s)\n", err)
			}
			fmt.Printf("Trigger deleted.\n")
			found = true
			break
		}
	}

	if !found {
		sdk.Exit("Error: trigger not found\n")
	}

}
