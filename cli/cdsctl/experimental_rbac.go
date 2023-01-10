package main

import (
	"context"
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
	})
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
	return client.RBACDelete(context.Background(), v.GetString("permission_identifier"))
}
