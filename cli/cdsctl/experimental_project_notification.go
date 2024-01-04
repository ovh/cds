package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	yaml "github.com/rockbears/yaml"
)

var projectNotifCmd = cli.Command{
	Name:    "notification",
	Aliases: []string{"notif"},
	Short:   "Manage Notification on a CDS project",
}

func projectNotification() *cobra.Command {
	return cli.NewCommand(projectNotifCmd, nil, []*cobra.Command{
		cli.NewListCommand(projectNotificationListCmd, projectNotificationListFunc, nil, withAllCommandModifiers()...),
		cli.NewDeleteCommand(projectNotificationDeleteCmd, projectNotificationDeleteFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(projectNotificationImportCmd, projectNotificationImportFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(projectNotificationExportCmd, projectNotificationExportFunc, nil, withAllCommandModifiers()...),
	})
}

var projectNotificationListCmd = cli.Command{
	Name:  "list",
	Short: "List available notifications on a project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
}

func projectNotificationListFunc(v cli.Values) (cli.ListResult, error) {
	notifs, err := client.ProjectNotificationList(context.Background(), v.GetString(_ProjectKey))
	return cli.AsListResult(notifs), err
}

var projectNotificationDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a notification on a project",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "name"},
	},
}

func projectNotificationDeleteFunc(v cli.Values) error {
	return client.ProjectNotificationDelete(context.Background(), v.GetString(_ProjectKey), v.GetString("name"))
}

var projectNotificationImportCmd = cli.Command{
	Name:    "import",
	Short:   "Import a notification on a project from a yaml file",
	Example: "cdsctl project notification import MY-PROJECT file.yml",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "filename"},
	},
	Flags: []cli.Flag{
		{Name: "force", Type: cli.FlagBool},
	},
}

func projectNotificationImportFunc(v cli.Values) error {
	btes, err := os.ReadFile(v.GetString("filename"))
	if err != nil {
		return cli.WrapError(err, "unable to open file %s", v.GetString("filename"))
	}

	var content sdk.ProjectNotification
	if err := yaml.Unmarshal(btes, &content); err != nil {
		return cli.WrapError(err, "unable to parse file %s", v.GetString("filename"))
	}

	n, err := client.ProjectNotificationGet(context.Background(), v.GetString(_ProjectKey), content.Name)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return cli.WrapError(err, "unable to get notification")
	}
	if n.ID != "" && !v.GetBool("force") {
		fmt.Printf("The notification %s already exists. Please use --force flag to update it.\n", content.Name)
		return nil
	}

	if n != nil && v.GetBool("force") {
		content.ID = n.ID
		return client.ProjectNotificationUpdate(context.Background(), v.GetString(_ProjectKey), &content)
	}
	return client.ProjectNotificationCreate(context.Background(), v.GetString(_ProjectKey), &content)
}

var projectNotificationExportCmd = cli.Command{
	Name:    "export",
	Short:   "Export a notification from a project to stdout",
	Example: "cdsctl notification export MY-PROJECT MY-VCS-SERVER-NAME > file.yaml",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "name"},
	},
}

func projectNotificationExportFunc(v cli.Values) error {
	pf, err := client.ProjectNotificationGet(context.Background(), v.GetString(_ProjectKey), v.GetString("name"))
	if err != nil {
		return err
	}

	btes, err := yaml.Marshal(pf)
	if err != nil {
		return err
	}

	fmt.Println(string(btes))
	return nil
}
