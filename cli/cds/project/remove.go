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

	// Delete all apps
	apps, err := sdk.ListApplications(key)
	if err != nil {
		return err
	}
	for _, app := range apps {
		err = sdk.DeleteApplication(key, app.Name)
		if err != nil {
			return err
		}
	}

	// Delete all pipelines
	pips, err := sdk.ListPipelines(key)
	if err != nil {
		return err
	}
	for _, pip := range pips {
		err = sdk.DeletePipeline(key, pip.Name)
		if err != nil {
			return err
		}
	}

	// Delete all environments
	envs, err := sdk.ListEnvironments(key)
	if err != nil {
		return err
	}
	for _, env := range envs {
		err = sdk.DeleteEnvironment(key, env.Name)
		if err != nil {
			return err
		}
	}

	// Delete project
	err = sdk.RemoveProject(key)
	if err != nil {
		return err
	}

	return nil
}
