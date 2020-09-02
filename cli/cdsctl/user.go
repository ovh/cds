package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var userCmd = cli.Command{
	Name:    "user",
	Aliases: []string{"users"},
	Short:   "Manage CDS user",
}

func usr() *cobra.Command {
	return cli.NewCommand(userCmd, nil, []*cobra.Command{
		cli.NewGetCommand(userMeCmd, userMeRun, nil),
		cli.NewListCommand(userListCmd, userListRun, nil),
		cli.NewGetCommand(userShowCmd, userShowRun, nil),
		cli.NewCommand(userFavoriteCmd, userFavoriteRun, nil),
	})
}

var userListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS users",
}

func userListRun(v cli.Values) (cli.ListResult, error) {
	users, err := client.UserList()
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(users), nil
}

var userMeCmd = cli.Command{
	Name:  "me",
	Short: "Show Current CDS user details",
}

func userMeRun(v cli.Values) (interface{}, error) {
	u, err := client.UserGetMe()
	if err != nil {
		return nil, err
	}
	return u, nil
}

var userShowCmd = cli.Command{
	Name:  "show",
	Short: "Show CDS user details",
	Args: []cli.Arg{
		{
			Name: "username",
		},
	},
}

func userShowRun(v cli.Values) (interface{}, error) {
	u, err := client.UserGet(v.GetString("username"))
	if err != nil {
		return nil, err
	}
	return *u, nil
}

var userFavoriteCmd = cli.Command{
	Name:  "favorite",
	Short: "Display all the user favorites",
}

func userFavoriteRun(v cli.Values) error {
	config, err := client.ConfigUser()
	if err != nil {
		return nil
	}

	navbarInfos, err := client.Navbar()
	if err != nil {
		return err
	}

	projFavs := []sdk.NavbarProjectData{}
	wfFavs := []sdk.NavbarProjectData{}
	for _, elt := range navbarInfos {
		if elt.Favorite {
			switch elt.Type {
			case "workflow":
				wfFavs = append(wfFavs, elt)
			case "project":
				projFavs = append(projFavs, elt)
			}
		}
	}

	fmt.Println(" -=-=-=-=- Projects bookmarked -=-=-=-=-")
	for _, prj := range projFavs {
		fmt.Printf("- %s %s\n", prj.Name, config.URLUI+"/project/"+prj.Key)
	}

	fmt.Println("\n -=-=-=-=- Workflows bookmarked -=-=-=-=-")
	for _, wf := range wfFavs {
		fmt.Printf("- %s %s\n", wf.WorkflowName, config.URLUI+"/project/"+wf.Key+"/workflow/"+wf.WorkflowName)
	}

	return nil
}
