package main

import (
	"fmt"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var projectFavoriteCmd = cli.Command{
	Name:  "favorite",
	Short: "Add or delete a CDS project to your personal bookmarks",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
}

func projectFavoriteRun(c cli.Values) error {
	params := sdk.FavoriteParams{
		Type:       "project",
		ProjectKey: c[_ProjectKey],
	}

	res, err := client.UpdateFavorite(params)
	if err != nil {
		return err
	}

	if proj, ok := res.(sdk.Project); ok {
		if proj.Favorite {
			fmt.Printf("Bookmarks added for project %s\n", proj.Name)
		} else {
			fmt.Printf("Bookmarks deleted for project %s\n", proj.Name)
		}
	} else {
		fmt.Println("Bookmarks updated")
	}

	return nil
}
