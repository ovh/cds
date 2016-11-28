package group

import (
	"fmt"

	"github.com/ovh/cds/sdk"

	"github.com/spf13/cobra"
)

func cmdGroupInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "cds group info <groupName>",
		Long:  ``,
		Run:   infoGroup,
	}

	return cmd
}

func infoGroup(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	groupName := args[0]

	group, err := sdk.GetGroup(groupName)
	if err != nil {
		sdk.Exit("%s\n", err)
	}

	fmt.Printf("Groupname: %s\n", group.Name)

	if group.Users != nil || group.Admins != nil {
		fmt.Printf("Users:\n")
		for _, u := range group.Admins {
			fmt.Printf(" - %s [Admin]\n", u.Username)
		}
		for _, u := range group.Users {
			fmt.Printf(" - %s\n", u.Username)
		}
	}

	if group.ProjectGroups != nil {
		fmt.Printf("Projects:\n")
		for _, prj := range group.ProjectGroups {
			fmt.Printf(" - %s : %d\n", prj.Project.Name, prj.Permission)
		}
	}

	if group.PipelineGroups != nil {
		fmt.Printf("Pipelines:\n")
		for _, pip := range group.PipelineGroups {
			fmt.Printf(" - %s : %d\n", pip.Pipeline.Name, pip.Permission)
		}
	}

}
