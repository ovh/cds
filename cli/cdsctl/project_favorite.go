package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var projectFavoriteCmd = cli.Command{
	Name:    "favorites",
	Aliases: []string{"favorite"},
	Short:   "Manage CDS group linked to a project",
}

func projectFavorite() *cobra.Command {
	return cli.NewCommand(projectFavoriteCmd, nil, []*cobra.Command{
		cli.NewCommand(projectFavoriteToggleCmd, projectFavoriteToggleRun, nil, withAllCommandModifiers()...),
		cli.NewListCommand(projectFavoriteListCmd, projectListFavoriteRun, nil, withAllCommandModifiers()...),
	})
}

var projectFavoriteToggleCmd = cli.Command{
	Name:  "toggle",
	Short: "Add or delete a CDS project to your personal bookmarks",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
}

func projectFavoriteToggleRun(c cli.Values) error {
	params := sdk.FavoriteParams{
		Type:       "project",
		ProjectKey: c.GetString(_ProjectKey),
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

var projectFavoriteListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS favorites projects",
}

func projectListFavoriteRun(v cli.Values) (cli.ListResult, error) {
	projects, err := client.ProjectFavoritesList("me")
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(projects), nil
}
