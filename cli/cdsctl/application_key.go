package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var applicationKeyCmd = cli.Command{
	Name:  "keys",
	Short: "Manage CDS application keys",
}

func applicationKey() *cobra.Command {
	return cli.NewCommand(applicationKeyCmd, nil, []*cobra.Command{
		cli.NewCommand(applicationKeyCreateCmd, applicationCreateKeyRun, nil, withAllCommandModifiers()...),
		cli.NewListCommand(applicationKeyListCmd, applicationListKeyRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(applicationKeyDeleteCmd, applicationDeleteKeyRun, nil, withAllCommandModifiers()...),
	})
}

var applicationKeyCreateCmd = cli.Command{
	Name:  "add",
	Short: "Add a new key on application. key type can be ssh or pgp",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _ApplicationName},
	},
	Args: []cli.Arg{
		{Name: "key-name"},
		{Name: "key-type"},
	},
}

func applicationCreateKeyRun(v cli.Values) error {
	key := &sdk.ApplicationKey{
		Key: sdk.Key{
			Name: v["key-name"],
			Type: v["key-type"],
		},
	}
	if err := client.ApplicationKeyCreate(v[_ProjectKey], v[_ApplicationName], key); err != nil {
		return err
	}

	fmt.Printf("Application key %s of type %s created with success in application %s\n", key.Name, key.Type, v[_ApplicationName])
	fmt.Println(key.Public)
	return nil
}

var applicationKeyListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS application keys",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _ApplicationName},
	},
}

func applicationListKeyRun(v cli.Values) (cli.ListResult, error) {
	keys, err := client.ApplicationKeysList(v[_ProjectKey], v[_ApplicationName])
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(keys), nil
}

var applicationKeyDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete CDS an application key",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _ApplicationName},
	},
	Args: []cli.Arg{
		{Name: "key-name"},
	},
}

func applicationDeleteKeyRun(v cli.Values) error {
	return client.ApplicationKeysDelete(v[_ProjectKey], v[_ApplicationName], v["key-name"])
}
