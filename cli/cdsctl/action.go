package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strings"

	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

var (
	actionCmd = cli.Command{
		Name:  "action",
		Short: "Manage CDS action",
	}

	action = cli.NewCommand(actionCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(actionListCmd, actionListRun, nil),
			cli.NewGetCommand(actionShowCmd, actionShowRun, nil),
			cli.NewCommand(actionDeleteCmd, actionDeleteRun, nil),
			cli.NewCommand(actionDocCmd, actionDocRun, nil),
			cli.NewCommand(actionImportCmd, actionImportRun, nil),
			cli.NewCommand(actionExportCmd, actionExportRun, nil),
		})
)

var actionListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS actions",
	Long: `Useful list CDS actions

cdsctl action list`,
}

func actionListRun(v cli.Values) (cli.ListResult, error) {
	actions, err := client.ActionList()
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(actions), nil
}

var actionShowCmd = cli.Command{
	Name:  "show",
	Short: "Show a CDS action",
	Args: []cli.Arg{
		{Name: "action-name"},
	},
	Long: `Useful to show a CDS action

cdsctl action show myAction`,
}

func actionShowRun(v cli.Values) (interface{}, error) {
	action, err := client.ActionGet(v["action-name"])
	if err != nil {
		return nil, err
	}
	return *action, nil
}

var actionDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a CDS action",
	Args: []cli.Arg{
		{Name: "action-name"},
	},
	Long: `Useful to delete a CDS action

cdsctl action delete myAction`,
}

func actionDeleteRun(v cli.Values) error {
	return client.ActionDelete(v["action-name"])
}

var actionDocCmd = cli.Command{
	Name:  "doc",
	Short: "Generate Action Documentation: cdsctl action doc <path-to-file>",
	Args: []cli.Arg{
		{Name: "path"},
	},
}

func actionDocRun(v cli.Values) error {
	btes, errRead := ioutil.ReadFile(v["path"])
	if errRead != nil {
		return fmt.Errorf("Error while reading file: %s", errRead)
	}

	var ea = new(exportentities.Action)
	var errapp error
	if strings.HasSuffix(path.Base(v["path"]), ".json") {
		errapp = json.Unmarshal(btes, ea)
	} else if strings.HasSuffix(path.Base(v["path"]), ".yml") || strings.HasSuffix(path.Base(v["path"]), ".yaml") {
		errapp = yaml.Unmarshal(btes, ea)
	} else {
		return fmt.Errorf("unsupported extension on %s", path.Base(v["path"]))
	}

	if errapp != nil {
		return errapp
	}

	act, errapp := ea.Action()
	if errapp != nil {
		return errapp
	}

	fmt.Println(sdk.ActionInfoMarkdown(act, path.Base(v["path"])))
	return nil
}

var actionImportCmd = cli.Command{
	Name:  "import",
	Short: "Import a CDS action",
	Args: []cli.Arg{
		{Name: "path"},
	},
	Long: `Useful to import a CDS action from a file

cdsctl action import myAction.yml`,
}

func actionImportRun(v cli.Values) error {
	path := v.GetString("path")
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var format = "yaml"
	if strings.HasSuffix(path, ".json") {
		format = "json"
	}

	errImport := client.ActionImport(f, format)
	if errImport != nil {
		return errImport
	}

	fmt.Printf("%s successfully imported\n", path)
	return nil
}

var actionExportCmd = cli.Command{
	Name:  "export",
	Short: "Export a CDS action",
	Long: `Useful to export a CDS action

cdsctl action export myAction`,
	Args: []cli.Arg{
		{Name: "action-name"},
	},
	Flags: []cli.Flag{
		{
			Kind:    reflect.String,
			Name:    "format",
			Usage:   "Specify export format (json or yaml)",
			Default: "yaml",
		},
	},
}

func actionExportRun(v cli.Values) error {
	b, err := client.ActionExport(v.GetString("action-name"), v.GetString("format"))
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
