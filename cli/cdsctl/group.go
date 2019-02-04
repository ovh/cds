package main

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var groupCmd = cli.Command{
	Name:  "group",
	Short: "Manage CDS group",
}

func group() *cobra.Command {
	return cli.NewCommand(groupCmd, nil, []*cobra.Command{
		cli.NewListCommand(groupListCmd, groupListRun, nil),
		cli.NewGetCommand(groupShowCmd, groupShowRun, nil),
		cli.NewCommand(groupCreateCmd, groupCreateRun, nil),
		cli.NewCommand(groupRenameCmd, groupRenameRun, nil),
		cli.NewDeleteCommand(groupDeleteCmd, groupDeleteRun, nil),
		cli.NewCommand(groupGrantCmd, groupGrantRun, nil),
		cli.NewCommand(groupRevokeCmd, groupRevokeRun, nil),
		groupUser(),
	})
}

var groupListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS groups",
}

func groupListRun(v cli.Values) (cli.ListResult, error) {
	apps, err := client.GroupList()
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(apps), nil
}

var groupShowCmd = cli.Command{
	Name:  "show",
	Short: "Show a CDS group",
	Args: []cli.Arg{
		{Name: "group-name"},
	},
}

func groupShowRun(v cli.Values) (interface{}, error) {
	group, err := client.GroupGet(v.GetString("group-name"))
	if err != nil {
		return nil, err
	}
	return *group, nil
}

var groupCreateCmd = cli.Command{
	Name:  "create",
	Short: "Create a CDS group",
	Args: []cli.Arg{
		{Name: "group-name"},
	},
	Aliases: []string{"add"},
}

func groupCreateRun(v cli.Values) error {
	gr := &sdk.Group{Name: v.GetString("group-name")}
	return client.GroupCreate(gr)
}

var groupRenameCmd = cli.Command{
	Name:  "rename",
	Short: "Rename a CDS group",
	Args: []cli.Arg{
		{Name: "old-group-name"},
		{Name: "new-group-name"},
	},
}

func groupRenameRun(v cli.Values) error {
	return client.GroupRename(v.GetString("old-group-name"), v.GetString("new-group-name"))
}

var groupDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a CDS group",
	Args: []cli.Arg{
		{Name: "group-name"},
	},
}

func groupDeleteRun(v cli.Values) error {
	err := client.GroupDelete(v.GetString("group-name"))
	if err != nil {
		if v.GetBool("force") && sdk.ErrorIs(err, sdk.ErrGroupNotFound) {
			fmt.Println(err.Error())
			return nil
		}
	}

	return err
}

var groupGrantCmd = cli.Command{
	Name:    "grant",
	Short:   "Grant a CDS group in a project or workflow",
	Aliases: []string{"add", "insert"},
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "group-name"},
		{
			Name: "permission",
			IsValid: func(arg string) bool {
				perm, err := strconv.Atoi(arg)
				if err != nil {
					return false
				}
				switch perm {
				case 4, 5, 7:
					return true
				default:
					return false
				}
			},
		},
	},
	Flags: []cli.Flag{
		{
			Name:      "workflow",
			ShortHand: "n",
			Usage:     "Workflow name",
		},
		{
			Name:      "only-project",
			ShortHand: "p",
			Usage:     "Indicate if the group must be added only on project or also on all workflows in project",
			Type:      cli.FlagBool,
		},
	},
}

func groupGrantRun(v cli.Values) error {
	groupName := v.GetString("group-name")
	// Don't check error (already done in isValid function)
	permission, _ := v.GetInt64("permission")
	project := v.GetString(_ProjectKey)
	workflow := v.GetString("workflow")

	if workflow != "" {
		if err := client.WorkflowGroupAdd(project, workflow, groupName, int(permission)); err != nil {
			return sdk.WrapError(err, "cannot add group %s workflow %s/%s", groupName, project, workflow)
		}
		fmt.Printf("Group '%s' added on workflow '%s/%s' with success\n", groupName, project, workflow)
	} else {
		if err := client.ProjectGroupAdd(project, groupName, int(permission), v.GetBool("only-project")); err != nil {
			return sdk.WrapError(err, "cannot add group %s on project %s", groupName, project)
		}
		fmt.Printf("Group '%s' added on project '%s' with success\n", groupName, project)
	}

	return nil
}

var groupRevokeCmd = cli.Command{
	Name:    "revoke",
	Short:   "Revoke a CDS group in a project or workflow",
	Aliases: []string{"remove", "delete", "rm", "del"},
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "group-name"},
	},
	Flags: []cli.Flag{
		{
			Name:      "workflow",
			ShortHand: "n",
			Usage:     "Workflow name",
		},
	},
}

func groupRevokeRun(v cli.Values) error {
	groupName := v.GetString("group-name")
	project := v.GetString(_ProjectKey)
	workflow := v.GetString("workflow")

	if workflow != "" {
		if err := client.WorkflowGroupDelete(project, workflow, groupName); err != nil {
			return sdk.WrapError(err, "cannot delete group %s in workflow %s/%s", groupName, project, workflow)
		}
		fmt.Printf("Group '%s' deleted on workflow '%s/%s' with success\n", groupName, project, workflow)
	} else {
		if err := client.ProjectGroupDelete(project, groupName); err != nil {
			return sdk.WrapError(err, "cannot delete group %s on project %s", groupName, project)
		}
		fmt.Printf("Group '%s' deleted on project '%s' with success\n", groupName, project)
	}

	return nil
}
