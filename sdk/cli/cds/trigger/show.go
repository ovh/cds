package trigger

import (
	"fmt"

	yaml "gopkg.in/yaml.v2"

	"github.com/spf13/cobra"
	"github.com/ovh/cds/sdk"
)

func showTriggerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "cds trigger show <srcproject>/<srcapp>/<srcpip>[/<env>] <destproject>/<destapp>/<desstpip>[/<destenv>]",
		Long:  ``,
		Run:   showTrigger,
	}

	return cmd
}

func showTrigger(cmd *cobra.Command, args []string) {

	if len(args) != 2 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	t, err := triggerFromString(args[0], args[1])
	if err != nil {
		sdk.Exit("Error: %s", err)
	}

	triggers, err := sdk.GetTriggers(t.SrcProject.Key, t.SrcApplication.Name, t.SrcPipeline.Name, t.SrcEnvironment.Name)
	if err != nil {
		sdk.Exit("Error: cannot retrieve triggers (%s)\n", err)
	}

	var trigger *sdk.PipelineTrigger
	for _, tr := range triggers {
		if triggersEqual(t, &tr) {
			trigger, err = sdk.GetTrigger(t.DestProject.Key, t.DestApplication.Name, t.DestPipeline.Name, tr.ID)
			if err != nil {
				sdk.Exit("Error: cannot get trigger: %s", err)
			}
			break
		}
	}
	if trigger == nil {
		sdk.Exit("Error: trigger not found")
	}

	data, err := yaml.Marshal(trigger)
	if err != nil {
		sdk.Exit("Error: cannot format output (%s)\n", err)
	}

	fmt.Println(string(data))
}
