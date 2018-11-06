package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var (
	templateCmd = cli.Command{
		Name:  "template",
		Short: "Manage CDS workflow template",
	}

	template = cli.NewCommand(templateCmd, nil, []*cobra.Command{
		cli.NewCommand(templateApplyCmd, templateApplyRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(templatePullCmd, templatePullRun, nil, withAllCommandModifiers()...),
	})
)

var templatePullCmd = cli.Command{
	Name:    "pull",
	Short:   "Pull CDS workflow template",
	Example: "cdsctl pull group-name/template-slug",
	OptionalArgs: []cli.Arg{
		{Name: "template-path"},
	},
	Flags: []cli.Flag{
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

func templatePullRun(v cli.Values) error {
	wt, err := getTemplateFromCLI(v)
	if err != nil {
		return err
	}
	if wt == nil {
		wt, err = suggestTemplate()
		if err != nil {
			return err
		}
	}

	dir := strings.TrimSpace(v.GetString("output-dir"))
	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, os.FileMode(0744)); err != nil {
		return fmt.Errorf("Unable to create directory %s: %v", v.GetString("output-dir"), err)
	}

	t, err := client.TemplatePull(wt.Group.Name, wt.Slug)
	if err != nil {
		return err
	}

	return workflowTarReaderToFiles(dir, t, v.GetBool("force"), v.GetBool("quiet"))
}
