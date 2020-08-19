package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/exportentities"
)

var templateCmd = cli.Command{
	Name:    "template",
	Aliases: []string{"templates"},
	Short:   "Manage CDS workflow template",
}

func template() *cobra.Command {
	return cli.NewCommand(templateCmd, nil, []*cobra.Command{
		cli.NewListCommand(templateListCmd, templateListRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(templateApplyCmd("apply"), templateApplyRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(templateBulkCmd, templateBulkRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(templatePullCmd, templatePullRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(templatePushCmd, templatePushRun, nil, withAllCommandModifiers()...),
		cli.NewDeleteCommand(templateDeleteCmd, templateDeleteRun, nil, withAllCommandModifiers()...),
		cli.NewListCommand(templateInstancesCmd, templateInstancesRun, []*cobra.Command{
			cli.NewCommand(templateInstancesExportCmd, templateInstancesExportRun, nil, withAllCommandModifiers()...),
		}),
		cli.NewCommand(templateDetachCmd, templateDetachRun, nil, withAllCommandModifiers()...),
	})
}

var templateListCmd = cli.Command{
	Name:    "list",
	Short:   "Get all available workflow template from CDS",
	Example: "cdsctl template list",
}

func templateListRun(v cli.Values) (cli.ListResult, error) {
	wts, err := client.TemplateGetAll()
	if err != nil {
		return nil, err
	}

	type TemplateDisplay struct {
		Path        string `cli:"path,key"`
		Name        string `cli:"name"`
		Description string `cli:"description"`
	}

	tds := make([]TemplateDisplay, len(wts))
	for i := range wts {
		tds[i].Path = fmt.Sprintf("%s/%s", wts[i].Group.Name, wts[i].Slug)
		tds[i].Name = wts[i].Name
		tds[i].Description = wts[i].Description
	}

	return cli.AsListResult(tds), nil
}

var templatePullCmd = cli.Command{
	Name:    "pull",
	Short:   "Pull CDS workflow template",
	Example: "cdsctl template pull group-name/template-slug",
	OptionalArgs: []cli.Arg{
		{Name: "template-path"},
	},
	Flags: []cli.Flag{
		{
			Name:      "output-dir",
			ShortHand: "d",
			Usage:     "Output directory",
			Default:   ".cds",
		},
		{
			Type:    cli.FlagBool,
			Name:    "force",
			Usage:   "Force, may override files",
			Default: "false",
		},
		{
			Type:    cli.FlagBool,
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
		return fmt.Errorf("unable to create directory %s: %v", v.GetString("output-dir"), err)
	}

	t, err := client.TemplatePull(wt.Group.Name, wt.Slug)
	if err != nil {
		return err
	}

	return workflowTarReaderToFiles(v, dir, t)
}

var templatePushCmd = cli.Command{
	Name:    "push",
	Short:   "Push CDS workflow template",
	Example: "cdsctl template push my-template.yml workflow.yml 1.pipeline.yml",
	VariadicArgs: cli.Arg{
		Name: "yaml-file",
	},
	Flags: []cli.Flag{
		{
			Type:  cli.FlagBool,
			Name:  "skip-update-files",
			Usage: "Useful if you don't want to update yaml files after pushing the template.",
		},
	},
}

func templatePushRun(v cli.Values) error {
	files := strings.Split(v.GetString("yaml-file"), ",")

	// create a new tar archive
	var dir string
	tar := new(bytes.Buffer)

	// if the first args is an url, try to download all files
	readFromLink := len(files) > 0 && sdk.IsURL(files[0]) && strings.HasSuffix(files[0], ".yml")
	if readFromLink {
		if err := exportentities.DownloadTemplate(files[0], tar); err != nil {
			return err
		}
	} else {
		filesToRead := []string{}
		for _, file := range files {
			fi, err := os.Lstat(file)
			if err != nil {
				fmt.Printf("skipping file %s: %v\n", file, err)
				continue
			}

			//Skip the directory
			if fi.IsDir() {
				continue
			}

			fmt.Println("Reading file ", cli.Magenta(file))
			if dir == "" {
				dir = filepath.Dir(file)
			}
			if dir != filepath.Dir(file) {
				return fmt.Errorf("files must be ine the same directory")
			}

			filesToRead = append(filesToRead, file)
		}

		if len(filesToRead) == 0 {
			return fmt.Errorf("wrong usage: you should specify your workflow template YAML files. See %s template push --help for more details", os.Args[0])
		}

		if err := workflowFilesToTarWriter(filesToRead, tar); err != nil {
			return err
		}
	}

	btes := tar.Bytes()
	r := bytes.NewBuffer(btes)
	msgList, tr, err := client.TemplatePush(r)
	for _, msg := range msgList {
		fmt.Println(msg)
	}
	if err != nil {
		return err
	}

	fmt.Println("Template successfully pushed !")

	if readFromLink || v.GetBool("skip-update-files") {
		return nil
	}

	return workflowTarReaderToFiles(v, dir, tr)
}

var templateDeleteCmd = cli.Command{
	Name:    "delete",
	Short:   "Delete a workflow template",
	Example: "cdsctl template delete group-name/template-slug",
	OptionalArgs: []cli.Arg{
		{Name: "template-path"},
	},
}

func templateDeleteRun(v cli.Values) error {
	wt, err := getTemplateFromCLI(v)
	if err != nil {
		if v.GetBool("force") && sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil
		}
		return err
	}
	if wt == nil {
		wt, err = suggestTemplate()
		if err != nil {
			if v.GetBool("force") && sdk.ErrorIs(err, sdk.ErrNotFound) {
				return nil
			}
			return err
		}
	}

	if err := client.TemplateDelete(wt.Group.Name, wt.Slug); err != nil {
		if v.GetBool("force") && sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil
		}
		return err
	}

	return nil
}

var templateDetachCmd = cli.Command{
	Name:    "detach",
	Short:   "Detach a workflow from template",
	Example: "cdsctl template detach project-key workflow-name",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName, AllowEmpty: true},
	},
}

func templateDetachRun(v cli.Values) error {
	projectKey := v.GetString(_ProjectKey)
	workflowName := v.GetString(_WorkflowName)

	// try to get workflow with a template instance if exists
	wk, err := client.WorkflowGet(projectKey, workflowName, cdsclient.WithTemplate())
	if err != nil {
		return err
	}
	if wk.TemplateInstance == nil {
		return fmt.Errorf("given workflow is was not generated by a template")
	}

	if err := client.TemplateDeleteInstance(wk.TemplateInstance.Template.Group.Name, wk.TemplateInstance.Template.Slug, wk.TemplateInstance.ID); err != nil {
		return err
	}

	fmt.Printf("Template instance successfully detached for workflow %s/%s\n", projectKey, workflowName)

	return nil
}
