package main

import (
	"fmt"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var workflowFavoriteCmd = cli.Command{
	Name:  "favorite",
	Short: "Add or delete a CDS workflow to your personal bookmarks",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
	},
}

func workflowFavoriteRun(c cli.Values) error {
	params := sdk.FavoriteParams{
		Type:         "workflow",
		ProjectKey:   c[_ProjectKey],
		WorkflowName: c[_WorkflowName],
	}

	if err := client.UpdateFavorite(params); err != nil {
		return err
	}
	fmt.Println("Bookmarks updated")

	return nil
}
