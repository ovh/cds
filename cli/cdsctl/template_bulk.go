package main

import (
	"fmt"
	"reflect"

	"github.com/AlecAivazis/survey"

	"github.com/ovh/cds/cli"
)

var templateBulkCmd = cli.Command{
	Name:    "bulk",
	Short:   "Bulk apply CDS workflow template and push all given workflows",
	Example: "cdsctl template bulk group-name/template-slug",
	OptionalArgs: []cli.Arg{
		{Name: "template-path"},
	},
	Flags: []cli.Flag{
		{
			Kind:      reflect.Slice,
			Name:      "params",
			ShortHand: "p",
			Usage:     "Specify params for template",
			Default:   "",
		},
		{
			Kind:      reflect.Bool,
			Name:      "no-interactive",
			ShortHand: "n",
			Usage:     "Set to not ask interactively for params",
		},
		{
			Kind:      reflect.String,
			Name:      "output-dir",
			ShortHand: "d",
			Usage:     "Output directory",
			Default:   ".cds",
		},
		{
			Kind:    reflect.Bool,
			Name:    "force",
			Usage:   "Force, may override files",
			Default: "false",
		},
		{
			Kind:    reflect.Bool,
			Name:    "quiet",
			Usage:   "If true, do not output filename created",
			Default: "false",
		},
	},
}

func templateBulkRun(v cli.Values) error {
	wt, err := getTemplateFromCLI(v)
	if err != nil {
		return err
	}

	// if no template found for workflow or no instance, suggest one
	if wt == nil {
		wt, err = suggestTemplate()
		if err != nil {
			return err
		}
	}

	// ask interactively for params if prompt not disabled
	if !v.GetBool("no-interactive") {
		// get all existings template instances
		wtis, err := client.TemplateGetInstances(wt.Group.Name, wt.Slug)
		if err != nil {
			return err
		}

		opts := make([]cli.CustomMultiSelectOption, len(wtis))
		for i := range wtis {
			if wtis[i].Workflow != nil {
				var upToDate string
				if wtis[i].WorkflowTemplateVersion < wt.Version {
					upToDate = cli.Red("not up to date")
				} else {
					upToDate = cli.Red("not up to date")
				}
				opts[i] = cli.CustomMultiSelectOption{
					Value:   fmt.Sprintf("%s/%s", wtis[i].Project.Key, wtis[i].Workflow.Name),
					Info:    upToDate,
					Default: true,
				}
			}
		}

		results := []string{}
		prompt := &cli.CustomMultiSelect{
			Message: "Select template's instances that you want to update",
			Options: opts,
		}
		prompt.Init()
		survey.AskOne(prompt, &results, nil)

		// TODO iterate over selected elements, if not updatable ask for params
		// TODO send bulk request
	}

	return nil
}
