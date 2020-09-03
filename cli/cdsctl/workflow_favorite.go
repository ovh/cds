package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var workflowFavoriteCmd = cli.Command{
	Name:    "favorites",
	Aliases: []string{"favorite"},
	Short:   "Manage CDS group linked to a workflow",
}

func workflowFavorite() *cobra.Command {
	return cli.NewCommand(workflowFavoriteCmd, nil, []*cobra.Command{
		cli.NewCommand(workflowFavoriteToggleCmd, workflowFavoriteToggleRun, nil, withAllCommandModifiers()...),
		cli.NewListCommand(workflowFavoriteListCmd, workflowListFavoriteRun, nil, withAllCommandModifiers()...),
	})
}

var workflowFavoriteToggleCmd = cli.Command{
	Name:  "toggle",
	Short: "Add or delete a CDS workflow to your personal bookmarks",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
	},
}

func workflowFavoriteToggleRun(c cli.Values) error {
	params := sdk.FavoriteParams{
		Type:         "workflow",
		ProjectKey:   c.GetString(_ProjectKey),
		WorkflowName: c.GetString(_WorkflowName),
	}

	res, err := client.UpdateFavorite(params)
	if err != nil {
		return err
	}

	if wf, ok := res.(sdk.Workflow); ok {
		if wf.Favorite {
			fmt.Printf("Bookmarks added for workflow %s\n", wf.Name)
		} else {
			fmt.Printf("Bookmarks deleted for workflow %s\n", wf.Name)
		}
	} else {
		fmt.Println("Bookmarks updated")
	}

	return nil
}

var workflowFavoriteListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS favorites workflows",
}

func workflowListFavoriteRun(v cli.Values) (cli.ListResult, error) {
	workflows, err := client.WorkflowFavoritesList("me")
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(workflows), nil
}
