package project

import (
	"fmt"

	"github.com/ovh/cds/sdk"

	"github.com/spf13/cobra"
)

func cmdProjectInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "cds project info <projectUniqueKey>",
		Long:  ``,
		Run:   getProject,
	}

	return cmd
}

func getProject(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	key := args[0]

	project, err := sdk.GetProject(key)
	if err != nil {
		sdk.Exit("Error: cannot get project info for project %s (%s)\n", key, err)
	}

	fmt.Printf("Project Key: %s\n", project.Key)
	fmt.Printf("Project Name: %s\n", project.Name)

	if project.Applications != nil {
		fmt.Printf("Applications:\n")
		for _, elt := range project.Applications {
			fmt.Printf(" - %s\n", elt.Name)
		}
	}

	if project.Environments != nil {
		fmt.Printf("Environments:\n")
		for _, elt := range project.Environments {
			fmt.Printf(" - %s\n", elt.Name)
		}
	}

	if project.Pipelines != nil {
		fmt.Printf("Pipelines:\n")
		for _, elt := range project.Pipelines {
			fmt.Printf(" - %s\n", elt.Name)
		}
	}

	if project.ProjectGroups != nil {
		fmt.Printf("Groups:\n")
		for _, elt := range project.ProjectGroups {
			fmt.Printf(" - %s %d\n", elt.Group.Name, elt.Permission)
		}
	}
}
