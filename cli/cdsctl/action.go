package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"time"

	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

var actionCmd = cli.Command{
	Name:  "action",
	Short: "Manage CDS action",
}

func action() *cobra.Command {
	return cli.NewCommand(actionCmd, nil, []*cobra.Command{
		cli.NewListCommand(actionListCmd, actionListRun, nil),
		cli.NewGetCommand(actionShowCmd, actionShowRun, nil),
		cli.NewCommand(actionDeleteCmd, actionDeleteRun, nil),
		cli.NewCommand(actionDocCmd, actionDocRun, nil),
		cli.NewCommand(actionImportCmd, actionImportRun, nil),
		cli.NewCommand(actionExportCmd, actionExportRun, nil),
	})
}

func newActionDisplay(a sdk.Action) actionDisplay {
	name := a.Name
	if a.Group != nil {
		name = fmt.Sprintf("%s/%s", a.Group.Name, a.Name)
	}

	return actionDisplay{
		ID: a.ID,
		Created: fmt.Sprintf("On %s by %s", a.FirstAudit.Created.Format(time.RFC3339),
			a.FirstAudit.AuditCommon.TriggeredBy),
		Name: name,
		Type: a.Type,
	}
}

type actionDisplay struct {
	ID      int64  `cli:"ID,key"`
	Created string `cli:"Created"`
	Name    string `cli:"Name"`
	Type    string `cli:"Type"`
}

func actionParsePath(path string) (string, string, error) {
	pathSplitted := strings.Split(path, "/")
	if len(pathSplitted) != 2 {
		return "", "", fmt.Errorf("invalid given action path")
	}
	return pathSplitted[0], pathSplitted[1], nil
}

var actionListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS actions",
}

func actionListRun(v cli.Values) (cli.ListResult, error) {
	as, err := client.ActionList()
	if err != nil {
		return nil, err
	}

	ads := make([]actionDisplay, len(as))
	for i := range as {
		ads[i] = newActionDisplay(as[i])
	}

	return cli.AsListResult(ads), nil
}

var actionShowCmd = cli.Command{
	Name:  "show",
	Short: "Show a CDS action",
	Args: []cli.Arg{
		{Name: "action-path"},
	},
}

func actionShowRun(v cli.Values) (interface{}, error) {
	groupName, actionName, err := actionParsePath(v.GetString("action-path"))
	if err != nil {
		return nil, err
	}

	action, err := client.ActionGet(groupName, actionName)
	if err != nil {
		return nil, err
	}

	return newActionDisplay(*action), nil
}

var actionDeleteCmd = cli.Command{
	Name:  "delete",
	Short: "Delete a CDS action",
	Long: `Useful to delete a CDS action

	cdsctl action delete myAction

	# this will not fail if action does not exist
	cdsctl action delete myActionNotExist --force
`,
	Args: []cli.Arg{
		{Name: "action-path"},
	},
	Flags: []cli.Flag{
		{
			Name:  "force",
			Usage: "if true, do not fail if action does not exist",
			IsValid: func(s string) bool {
				return s == "true" || s == "false"
			},
			Default: "false",
			Type:    cli.FlagBool,
		},
	},
}

func actionDeleteRun(v cli.Values) error {
	groupName, actionName, err := actionParsePath(v.GetString("action-path"))
	if err != nil {
		return err
	}

	err = client.ActionDelete(groupName, actionName)
	if v.GetBool("force") && sdk.ErrorIs(err, sdk.ErrNoAction) {
		fmt.Println(err)
		return nil
	}

	return err
}

var actionDocCmd = cli.Command{
	Name:  "doc",
	Short: "Generate Action Documentation: cdsctl action doc <path-to-file>",
	Args: []cli.Arg{
		{Name: "path"},
	},
}

func actionDocRun(v cli.Values) error {
	btes, errRead := ioutil.ReadFile(v.GetString("path"))
	if errRead != nil {
		return fmt.Errorf("Error while reading file: %s", errRead)
	}

	var ea exportentities.Action
	var err error
	if strings.HasSuffix(path.Base(v.GetString("path")), ".json") {
		err = json.Unmarshal(btes, &ea)
	} else if strings.HasSuffix(path.Base(v.GetString("path")), ".yml") || strings.HasSuffix(path.Base(v.GetString("path")), ".yaml") {
		err = yaml.Unmarshal(btes, &ea)
	} else {
		return fmt.Errorf("unsupported extension on %s", path.Base(v.GetString("path")))
	}
	if err != nil {
		return err
	}

	act, errapp := ea.Action()
	if errapp != nil {
		return errapp
	}

	fmt.Println(sdk.ActionInfoMarkdown(act, path.Base(v.GetString("path"))))
	return nil
}

var actionImportCmd = cli.Command{
	Name:  "import",
	Short: "Import a CDS action",
	Args: []cli.Arg{
		{Name: "path"},
	},
}

func actionImportRun(v cli.Values) error {
	path := v.GetString("path")
	contentFile, format, err := exportentities.OpenPath(path)
	if err != nil {
		return err
	}
	defer contentFile.Close() //nolint
	formatStr, _ := exportentities.GetFormatStr(format)

	if err := client.ActionImport(contentFile, formatStr); err != nil {
		return err
	}

	fmt.Printf("%s successfully imported\n", path)
	return nil
}

var actionExportCmd = cli.Command{
	Name:  "export",
	Short: "Export a CDS action",
	Args: []cli.Arg{
		{Name: "action-path"},
	},
	Flags: []cli.Flag{
		{
			Name:    "format",
			Usage:   "Specify export format (json or yaml)",
			Default: "yaml",
		},
	},
}

func actionExportRun(v cli.Values) error {
	groupName, actionName, err := actionParsePath(v.GetString("action-path"))
	if err != nil {
		return err
	}

	b, err := client.ActionExport(groupName, actionName, v.GetString("format"))
	if err != nil {
		return err
	}

	fmt.Println(string(b))

	return nil
}
