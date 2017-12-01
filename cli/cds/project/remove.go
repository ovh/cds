package project

import (
	"fmt"

	"github.com/ovh/cds/sdk"

	"github.com/spf13/cobra"
)

var forceDelete bool

func cmdProjectRemove() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove",
		Short:   "cds project remove <projectUniqueKey>",
		Long:    ``,
		Run:     removeProject,
		Aliases: []string{"delete", "rm", "del"},
	}

	cmd.Flags().BoolVarP(&forceDelete, "force", "", false, "delete project and everything in it")
	return cmd
}

func removeProject(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	key := args[0]

	var err error
	if forceDelete {
		err = forceRemoveProject(key)
	} else {
		err = sdk.RemoveProject(key)
	}

	if err != nil {
		if forceDelete && sdk.ErrorIs(err, sdk.ErrNoProject) {
			fmt.Printf("%s\n", err.Error())
			return
		}
		sdk.Exit("Error: cannot remove project %s (%s)\n", key, err)
	}

	fmt.Printf("OK\n")
}

func forceRemoveProject(key string) error {

	// Delete all workflow
	ws, errW := sdk.WorkflowList(key)
	if errW != nil {
		return errW
	}
	for _, w := range ws {
		if err := sdk.WorkflowDelete(key, w.Name); err != nil {
			return err
		}
	}

	// Delete all apps
	apps, errA := sdk.ListApplications(key)
	if errA != nil {
		return errA
	}
	for _, app := range apps {
		if err := sdk.DeleteApplication(key, app.Name); err != nil {
			return err
		}
	}

	// Delete all pipelines
	pips, errP := sdk.ListPipelines(key)
	if errP != nil {
		return errP
	}
	for _, pip := range pips {
		if err := sdk.DeletePipeline(key, pip.Name); err != nil {
			return err
		}
	}

	// Delete all environments
	envs, errE := sdk.ListEnvironments(key)
	if errE != nil {
		return errE
	}
	for _, env := range envs {
		if err := sdk.DeleteEnvironment(key, env.Name); err != nil {
			return err
		}
	}

	// Delete project
	return sdk.RemoveProject(key)
}
