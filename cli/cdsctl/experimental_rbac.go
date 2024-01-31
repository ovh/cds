package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/rockbears/yaml"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

var experimentalRbacCmd = cli.Command{
	Name:  "rbac",
	Short: "CDS Experimental rbac commands",
}

func experimentalRbac() *cobra.Command {
	return cli.NewCommand(experimentalRbacCmd, nil, []*cobra.Command{
		cli.NewCommand(rbacImportCmd, rbacImportFunc, nil, withAllCommandModifiers()...),
		cli.NewDeleteCommand(rbacDeleteCmd, rbacDeleteFunc, nil, withAllCommandModifiers()...),
		cli.NewCommand(rbacGetCmd, rbacGetFunc, nil, withAllCommandModifiers()...),
		cli.NewListCommand(rbacListCmd, rbacListFunc, nil, withAllCommandModifiers()...),
	})
}

var rbacListCmd = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Short:   "List CDS permissions",
	Example: "cdsctl rbac list ",
	Ctx:     []cli.Arg{},
	Args:    []cli.Arg{},
}

func rbacListFunc(v cli.Values) (cli.ListResult, error) {
	perms, err := client.RBACList(context.Background())
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(perms), nil
}

var rbacGetCmd = cli.Command{
	Name:    "show",
	Aliases: []string{"get"},
	Short:   "GET a CDS permission",
	Example: "cdsctl rbac get <permission identifier>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "permissionIdentifier"},
	},
	Flags: []cli.Flag{
		{
			Name: "format",
		},
	},
}

func rbacGetFunc(v cli.Values) error {
	perm, err := client.RBACGet(context.Background(), v.GetString("permissionIdentifier"))
	if err != nil {
		return err
	}
	format := v.GetString("format")
	var result []byte
	if format == "json" {
		result, _ = json.Marshal(perm)
	} else {
		result, _ = yaml.Marshal(perm)
	}
	fmt.Printf("%s", string(result))
	return nil
}

var rbacImportCmd = cli.Command{
	Name:    "import",
	Short:   "Import a rbac rule from a yaml file",
	Example: "cdsctl rbac import file.yml",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "filename"},
	},
	Flags: []cli.Flag{
		{Name: "force", Type: cli.FlagBool},
	},
}

func rbacImportFunc(v cli.Values) error {
	f, err := os.Open(v.GetString("filename"))
	if err != nil {
		return cli.WrapError(err, "unable to open file %s", v.GetString("filename"))
	}
	defer f.Close() // nolint

	var rbacRule sdk.RBAC
	body, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(body, &rbacRule); err != nil {
		return err
	}

	var mods []cdsclient.RequestModifier
	if v.GetBool("force") {
		mods = append(mods, cdsclient.Force())
	}
	_, err = client.RBACImport(context.Background(), rbacRule, mods...)
	return err
}

var rbacDeleteCmd = cli.Command{
	Name:    "delete",
	Short:   "Delete a permission",
	Aliases: []string{"remove", "rm"},
	Example: "cdsctl rbac delete <permission_identifier>",
	Ctx:     []cli.Arg{},
	Args: []cli.Arg{
		{Name: "permission_identifier"},
	},
}

func rbacDeleteFunc(v cli.Values) error {
	if err := client.RBACDelete(context.Background(), v.GetString("permission_identifier")); err != nil {
		if v.GetBool("force") && sdk.ErrorIs(err, sdk.ErrNotFound) {
			fmt.Println(err.Error())
			os.Exit(0)
		}
	}
	return nil
}
