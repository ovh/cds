package trigger

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"
)

func copyTriggerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "copy",
		Short:   "cds trigger copy <project>/<app>/<pip>/<triggerid> <srcproject>/<srcapp>/<srcpip>[/<srcenv>] <destproject>/<destapp>/<desstpip>[/<destenv>]",
		Long:    ``,
		Aliases: []string{"remove", "rm"},
		Run:     copyTrigger,
	}

	return cmd
}

func copyTrigger(cmd *cobra.Command, args []string) {

	if len(args) != 3 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	triggerT := strings.Split(args[0], "/")
	if len(triggerT) != 4 {
		sdk.Exit("Wrong usage: trigger to copy should be in the format <project>/<app>/<pip>/<triggerid>\n")
	}

	triggerID, err := strconv.ParseInt(triggerT[3], 10, 64)
	if err != nil {
		sdk.Exit("Wrong usage: invalid trigger id %s: %s\n", triggerT[3], err)
	}

	trigger, err := sdk.GetTrigger(triggerT[0], triggerT[1], triggerT[2], triggerID)
	if err != nil {
		sdk.Exit("Error: cannot get trigger %s: %s\n", args[0], err)
	}

	dstTrigger, err := triggerFromString(args[1], args[2])
	if err != nil {
		sdk.Exit("Error: %s", err)
	}

	dstTrigger.Parameters = append(dstTrigger.Parameters, trigger.Parameters...)
	dstTrigger.Prerequisites = append(dstTrigger.Prerequisites, trigger.Prerequisites...)
	dstTrigger.Manual = trigger.Manual

	if err := sdk.AddTrigger(dstTrigger); err != nil {
		sdk.Exit("Error: cannot create trigger: %s\n", err)
	}

	fmt.Println("Trigger copied.")
}
