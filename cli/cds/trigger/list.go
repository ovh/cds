package trigger

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func listTriggerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "cds trigger list <srcproject>/<srcapp>/<srcpip>[/<srcenv>]",
		Long:  ``,
		Run:   listTrigger,
	}

	return cmd
}

func listTrigger(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	src := args[0]
	t := strings.Split(src, "/")
	if len(t) < 3 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}
	p := t[0]
	app := t[1]
	pip := t[2]
	var env string
	if len(t) == 4 {
		env = t[3]
	} else {
		env = sdk.DefaultEnv.Name
	}

	triggers, err := sdk.GetTriggers(p, app, pip, env)
	if err != nil {
		sdk.Exit("Error: %s\n", err)
	}

	var triggering, triggered []sdk.PipelineTrigger
	for _, t := range triggers {
		if t.SrcProject.Key == p && t.SrcApplication.Name == app && t.SrcPipeline.Name == pip {
			triggering = append(triggering, t)
		} else {
			triggered = append(triggered, t)
		}
	}

	if len(triggering) > 0 {
		fmt.Printf("%s/%s/%s[%s] triggers:\n", p, app, pip, env)
		for _, t := range triggering {
			fmt.Printf("- %s/%s/%s[%s]\n", t.DestProject.Key, t.DestApplication.Name, t.DestPipeline.Name, t.DestEnvironment.Name)
		}
	}

	if len(triggered) > 0 {
		fmt.Printf("\n%s/%s/%s[%s] is triggered by:\n", p, app, pip, env)
		for _, t := range triggered {
			fmt.Printf("- %s/%s/%s[%s]\n", t.SrcProject.Key, t.SrcApplication.Name, t.SrcPipeline.Name, t.SrcEnvironment.Name)
		}
	}
}
