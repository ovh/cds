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
