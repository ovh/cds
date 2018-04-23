package main

import (
	"fmt"
	"reflect"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var (
	adminBroadcastsCmd = cli.Command{
		Name:  "broadcasts",
		Short: "Manage CDS broadcasts",
	}

	adminBroadcasts = cli.NewCommand(adminBroadcastsCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(adminBroadcastListCmd, adminBroadcastListRun, nil),
			cli.NewGetCommand(adminBroadcastShowCmd, adminBroadcastShowRun, nil),
			cli.NewCommand(adminBroadcastDeleteCmd, adminBroadcastDeleteRun, nil),
			cli.NewCommand(adminBroadcastCreateCmd, adminBroadcastCreateRun, nil),
		})
)

var adminBroadcastCreateCmd = cli.Command{
	Name:  "create",
	Short: "Create a CDS broadcast",
	Args: []cli.Arg{
		{Name: "title"},
		{Name: "content"},
	},
	Flags: []cli.Flag{
		{
			Kind:      reflect.String,
			Name:      "level",
			ShortHand: "l",
			Usage:     "Level of broadcast: info or warning",
			Default:   "info",
		},
	},
	Example: `level info:
	
	cdsctl admin broadcasts create "the title" "the content"

level warning:	

	cdsctl admin broadcasts create --level warning "the title" "the content"
	`,
	Aliases: []string{"add"},
}

func adminBroadcastCreateRun(v cli.Values) error {
	if v["level"] != "info" && v["level"] != "warning" {
		return fmt.Errorf("level have to be 'info' or 'warning'")
	}
	bc := &sdk.Broadcast{
		Level:   v["level"],
		Title:   v["title"],
		Content: v["content"],
	}
	return client.BroadcastCreate(bc)
}

var adminBroadcastShowCmd = cli.Command{
	Name:  "show",
	Short: "Show a CDS broadcast",
	Args: []cli.Arg{
		{Name: "id"},
	},
}

func adminBroadcastShowRun(v cli.Values) (interface{}, error) {
	bc, err := client.BroadcastGet(v["id"])
	if err != nil {
		return nil, err
	}
	return bc, nil
}

var adminBroadcastDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a CDS broadcast",
	Args: []cli.Arg{
		{Name: "id"},
	},
	Flags: []cli.Flag{
		{
			Name:  "force",
			Usage: "if true, do not fail if action does not exist",
			IsValid: func(s string) bool {
				if s != "true" && s != "false" {
					return false
				}
				return true
			},
			Default: "false",
			Kind:    reflect.Bool,
		},
	},
}

func adminBroadcastDeleteRun(v cli.Values) error {
	err := client.BroadcastDelete(v["id"])
	if v.GetBool("force") && sdk.ErrorIs(err, sdk.ErrNoBroadcast) {
		fmt.Println(err)
		return nil
	}
	return err
}

var adminBroadcastListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS broadcasts",
	Flags: []cli.Flag{
		{
			Kind:      reflect.String,
			Name:      "level",
			ShortHand: "t",
			Usage:     "Filter broadcast by level",
			Default:   "",
		},
	},
}

func adminBroadcastListRun(v cli.Values) (cli.ListResult, error) {
	srvs, err := client.Broadcasts()
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(srvs), nil
}
