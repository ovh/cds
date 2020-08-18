package main

import (
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	actionSDK "github.com/ovh/cds/sdk/action"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/slug"
)

const actionPathArg = "action-path"

var actionCmd = cli.Command{
	Name:    "action",
	Aliases: []string{"actions"},
	Short:   "Manage CDS action",
}

var actionBuiltinCmd = cli.Command{
	Name:  "builtin",
	Short: "Manage CDS builtin action",
}

func action() *cobra.Command {
	return cli.NewCommand(actionCmd, nil, []*cobra.Command{
		cli.NewListCommand(actionListCmd, actionListRun, nil),
		cli.NewListCommand(actionUsageCmd, actionUsageRun, nil),
		cli.NewGetCommand(actionShowCmd, actionShowRun, nil),
		cli.NewCommand(actionDeleteCmd, actionDeleteRun, nil),
		cli.NewCommand(actionDocCmd, actionDocRun, nil),
		cli.NewCommand(actionImportCmd, actionImportRun, nil),
		cli.NewCommand(actionExportCmd, actionExportRun, nil),
		cli.NewCommand(actionBuiltinCmd, nil, []*cobra.Command{
			cli.NewListCommand(actionBuiltinListCmd, actionBuiltinListRun, nil),
			cli.NewGetCommand(actionBuiltinShowCmd, actionBuiltinShowRun, nil),
			cli.NewCommand(actionBuiltinDocCmd, actionBuiltinDocRun, nil),
		}),
	})
}

func newActionDisplay(a sdk.Action) actionDisplay {
	name := a.Name
	if a.Group != nil {
		name = fmt.Sprintf("%s/%s", a.Group.Name, a.Name)
	}

	ad := actionDisplay{
		Fullname: name,
		Type:     a.Type,
	}

	if a.FirstAudit != nil {
		ad.Created = fmt.Sprintf("On %s by %s", a.FirstAudit.Created.Format(time.RFC3339),
			a.FirstAudit.AuditCommon.TriggeredBy)
	} else {
		ad.Created = "No audit found"
	}

	return ad
}

type actionDisplay struct {
	Created  string `cli:"Created"`
	Fullname string `cli:"Fullname,key"`
	Type     string `cli:"Type"`
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

var actionUsageCmd = cli.Command{
	Name:  "usage",
	Short: "CDS action usage",
	Args: []cli.Arg{
		{Name: actionPathArg},
	},
}

func actionUsageRun(v cli.Values) (cli.ListResult, error) {
	groupName, actionName, err := cli.ParsePath(v.GetString(actionPathArg))
	if err != nil {
		return nil, err
	}

	usages, err := client.ActionUsage(groupName, actionName)
	if err != nil {
		return nil, err
	}

	type ActionUsageDisplay struct {
		Type string `cli:"Type"`
		Path string `cli:"Path"`
	}

	au := []ActionUsageDisplay{}
	for _, v := range usages.Pipelines {
		au = append(au, ActionUsageDisplay{
			Type: "pipeline",
			Path: strings.Replace(fmt.Sprintf("%s - %s - %s", v.ProjectName, v.PipelineName, v.ActionName), " ", " ", -1),
		})
	}
	for _, v := range usages.Actions {
		au = append(au, ActionUsageDisplay{
			Type: "action",
			Path: fmt.Sprintf("%s/%s", v.GroupName, v.ParentActionName),
		})
	}
	return cli.AsListResult(au), nil
}

var actionShowCmd = cli.Command{
	Name:  "show",
	Short: "Show a CDS action",
	Args: []cli.Arg{
		{Name: actionPathArg},
	},
}

func actionShowRun(v cli.Values) (interface{}, error) {
	groupName, actionName, err := cli.ParsePath(v.GetString(actionPathArg))
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
		{Name: actionPathArg},
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
	groupName, actionName, err := cli.ParsePath(v.GetString(actionPathArg))
	if err != nil {
		return err
	}

	err = client.ActionDelete(groupName, actionName)
	if v.GetBool("force") && sdk.ErrorIs(err, sdk.ErrNotFound) {
		fmt.Println(err)
		return nil
	}

	return err
}

var actionDocCmd = cli.Command{
	Name:  "doc",
	Short: "Generate action documentation: cdsctl action doc <path-to-file>",
	Args: []cli.Arg{
		{Name: "path"},
	},
}

func actionDocRun(v cli.Values) error {
	p := v.GetString("path")

	contentFile, format, err := exportentities.OpenPath(p)
	if err != nil {
		return err
	}
	defer contentFile.Close()

	body, err := ioutil.ReadAll(contentFile)
	if err != nil {
		return err
	}

	var ea exportentities.Action
	if err := exportentities.Unmarshal(body, format, &ea); err != nil {
		return err
	}

	act, errapp := ea.GetAction()
	if errapp != nil {
		return errapp
	}

	fmt.Println(sdk.ActionInfoMarkdown(act, path.Base(p)))
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

	if err := client.ActionImport(contentFile, cdsclient.ContentType(format.ContentType())); err != nil {
		return err
	}

	fmt.Printf("%s successfully imported\n", path)
	return nil
}

var actionExportCmd = cli.Command{
	Name:  "export",
	Short: "Export a CDS action",
	Args: []cli.Arg{
		{Name: actionPathArg},
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
	groupName, actionName, err := cli.ParsePath(v.GetString(actionPathArg))
	if err != nil {
		return err
	}

	b, err := client.ActionExport(groupName, actionName, cdsclient.Format(v.GetString("format")))
	if err != nil {
		return err
	}

	fmt.Println(string(b))

	return nil
}

var actionBuiltinListCmd = cli.Command{
	Name:  "list",
	Short: "List CDS builtin actions",
}

func actionBuiltinListRun(v cli.Values) (cli.ListResult, error) {
	as, err := client.ActionBuiltinList()
	if err != nil {
		return nil, err
	}

	ads := make([]actionDisplay, len(as))
	for i := range as {
		ads[i] = newActionDisplay(as[i])
	}

	return cli.AsListResult(ads), nil
}

var actionBuiltinShowCmd = cli.Command{
	Name:  "show",
	Short: "Show a CDS builtin action",
	Args: []cli.Arg{
		{Name: "action-name"},
	},
}

func actionBuiltinShowRun(v cli.Values) (interface{}, error) {
	action, err := client.ActionBuiltinGet(v.GetString("action-name"))
	if err != nil {
		return nil, err
	}

	return newActionDisplay(*action), nil
}

var actionBuiltinDocCmd = cli.Command{
	Name:  "doc",
	Short: "Generate Builtin action documentation: cdsctl action builtin doc <name>",
	Args: []cli.Arg{
		{Name: "name"},
	},
}

func actionBuiltinDocRun(v cli.Values) error {
	n := v.GetString("name")

	var found bool
	var m actionSDK.Manifest
	for i := range actionSDK.List {
		if slug.Convert(actionSDK.List[i].Action.Name) == slug.Convert(n) {
			found = true
			m = actionSDK.List[i]
			break
		}
	}
	if !found {
		return fmt.Errorf("Invalid given action name %s", n)
	}

	fmt.Println(m.Markdown())
	return nil
}
