package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var projectVariableSetItemCmd = cli.Command{
	Name:    "item",
	Aliases: []string{"vs"},
	Short:   "Manage item on a CDS project variableset",
}

func projectVariableSetItem() *cobra.Command {
	return cli.NewCommand(projectVariableSetItemCmd, nil, []*cobra.Command{
		cli.NewListCommand(projectVariableSetItemListCmd, projectVariableSetItemListFunc, nil, withAllCommandModifiers()...),
		cli.NewDeleteCommand(projectVariableSetItemDeleteCmd, projectVariableSetItemDeleteFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(projectVariableSetItemCreateCmd, projectVariableSetItemCreateFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(projectVariableSetItemUpdateCmd, projectVariableSetItemUpdateFunc, nil, withAllCommandModifiers()...),
		cli.NewGetCommand(projectVariableSetItemShowCmd, projectVariableSetItemShowFunc, nil, withAllCommandModifiers()...),
	})
}

var projectVariableSetItemListCmd = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},

	Short: "List the items of the given variableset",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "variableset-name"},
	},
}

func projectVariableSetItemListFunc(v cli.Values) (cli.ListResult, error) {
	vs, err := client.ProjectVariableSetShow(context.Background(), v.GetString(_ProjectKey), v.GetString("variableset-name"))
	return cli.AsListResult(vs.Items), err
}

var projectVariableSetItemShowCmd = cli.Command{
	Name:    "show",
	Aliases: []string{"get"},
	Short:   "Get the given variableset item",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "variableset-name"},
		{Name: "item-name"},
	},
}

func projectVariableSetItemShowFunc(v cli.Values) (interface{}, error) {
	item, err := client.ProjectVariableSetItemGet(context.Background(), v.GetString(_ProjectKey), v.GetString("variable-set-name"), v.GetString("item-name"))
	return item, err
}

var projectVariableSetItemDeleteCmd = cli.Command{
	Name:    "delete",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete an item from a variableset",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "variableset-name"},
		{Name: "item-name"},
	},
}

func projectVariableSetItemDeleteFunc(v cli.Values) error {
	return client.ProjectVariableSetItemDelete(context.Background(), v.GetString(_ProjectKey), v.GetString("variableset-name"), v.GetString("item-name"))
}

var projectVariableSetItemCreateCmd = cli.Command{
	Name:    "add",
	Aliases: []string{"create"},
	Short:   "Create a new item inside a variableset ",
	Example: "cdsctl exp project variableset item add MY-PROJECT MY-VARIABLESET-NAME ITEM-NAME ITEM-VALUE ITEM-TYPE(secret|string)",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "variableset-name"},
		{Name: "item-name"},
		{Name: "item-value"},
		{Name: "item-type"},
	},
	Flags: []cli.Flag{
		{
			Name:  "force",
			Type:  cli.FlagBool,
			Usage: "create the variable set if it not exists",
		},
	},
}

func projectVariableSetItemCreateFunc(v cli.Values) error {
	item := sdk.ProjectVariableSetItem{
		Name:  v.GetString("item-name"),
		Value: v.GetString("item-value"),
		Type:  v.GetString("item-type"),
	}
	if v.GetString("item-type") != sdk.ProjectVariableTypeSecret && v.GetString("item-type") != sdk.ProjectVariableTypeString {
		return fmt.Errorf("item type must be '%s'or '%s'", sdk.ProjectVariableTypeSecret, sdk.ProjectVariableTypeString)
	}

	// If force check if the variableset exists and create it if needed
	if v.GetBool("force") {
		_, err := client.ProjectVariableSetShow(context.Background(), v.GetString(_ProjectKey), v.GetString("variableset-name"))
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			vs := &sdk.ProjectVariableSet{
				Name: v.GetString("variableset-name"),
			}
			if err := client.ProjectVariableSetCreate(context.Background(), v.GetString(_ProjectKey), vs); err != nil {
				return err
			}
		}
	}

	return client.ProjectVariableSetItemAdd(context.Background(), v.GetString(_ProjectKey), v.GetString("variableset-name"), &item)
}

var projectVariableSetItemUpdateCmd = cli.Command{
	Name:    "update",
	Aliases: []string{""},
	Short:   "Update an item inside a variableset ",
	Example: "cdsctl exp project variableset item update MY-PROJECT MY-VARIABLESET-NAME ITEM-NAME ITEM-VALUE",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	Args: []cli.Arg{
		{Name: "variableset-name"},
		{Name: "item-name"},
		{Name: "item-value"},
	},
}

func projectVariableSetItemUpdateFunc(v cli.Values) error {
	item := sdk.ProjectVariableSetItem{
		Name:  v.GetString("item-name"),
		Value: v.GetString("item-value"),
	}
	return client.ProjectVariableSetItemUpdate(context.Background(), v.GetString(_ProjectKey), v.GetString("variableset-name"), &item)
}
