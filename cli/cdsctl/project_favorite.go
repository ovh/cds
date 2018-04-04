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

	if err := client.UpdateFavorite(params); err != nil {
		return err
	}
	fmt.Println("Bookmarks updated")

	return nil
}
