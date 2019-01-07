package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/exportentities"
)

var templateCmd = cli.Command{
	Name:  "template",
	Short: "Manage CDS workflow template",
}

func template() *cobra.Command {
	return cli.NewCommand(templateCmd, nil, []*cobra.Command{
		cli.NewCommand(templateApplyCmd("apply"), templateApplyRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(templateBulkCmd, templateBulkRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(templatePullCmd, templatePullRun, nil, withAllCommandModifiers()...),
		cli.NewCommand(templatePushCmd, templatePushRun, nil, withAllCommandModifiers()...),
		cli.NewListCommand(templateInstancesCmd, templateInstancesRun, nil, withAllCommandModifiers()...),
	})
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
		return fmt.Errorf("unable to create directory %s: %v", v.GetString("output-dir"), err)
	}

	t, err := client.TemplatePull(wt.Group.Name, wt.Slug)
	if err != nil {
		return err
	}

	return workflowTarReaderToFiles(dir, t, v.GetBool("force"), v.GetBool("quiet"))
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
			Kind:  reflect.Bool,
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
	readFromLink := len(files) > 0 && exportentities.IsURL(files[0]) && strings.HasSuffix(files[0], ".yml")
	if readFromLink {
		manifestURL := files[0]
		baseURL := manifestURL[0:strings.LastIndex(manifestURL, "/")]

		// get the manifest file
		contentFile, _, err := exportentities.OpenPath(manifestURL)
		if err != nil {
			return err
		}
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(contentFile); err != nil {
			return fmt.Errorf("cannot read from given remote file: %v", err)
		}
		var t exportentities.Template
		if err := yaml.Unmarshal(buf.Bytes(), &t); err != nil {
			return fmt.Errorf("cannot unmarshal given remote yaml file: %v", err)
		}

		// get all components of the template
		paths := []string{t.Workflow}
		paths = append(paths, t.Pipelines...)
		paths = append(paths, t.Applications...)
		paths = append(paths, t.Environments...)

		links := make([]string, len(paths))
		for i := range paths {
			links[i] = fmt.Sprintf("%s/%s", baseURL, paths[i])
		}

		if err := workflowLinksToTarWriter(append(links, manifestURL), tar); err != nil {
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

	return workflowTarReaderToFiles(dir, tr, false, false)
}

var templateInstancesCmd = cli.Command{
	Name:    "instances",
	Short:   "Get instances for a CDS workflow template",
	Example: "cdsctl template instances group-name/template-slug",
	OptionalArgs: []cli.Arg{
		{Name: "template-path"},
	},
}

func templateInstancesRun(v cli.Values) (cli.ListResult, error) {
	wt, err := getTemplateFromCLI(v)
	if err != nil {
		return nil, err
	}
	if wt == nil {
		wt, err = suggestTemplate()
		if err != nil {
			return nil, err
		}
	}

	wtis, err := client.TemplateGetInstances(wt.Group.Name, wt.Slug)
	if err != nil {
		return nil, err
	}

	type TemplateInstanceDisplay struct {
		ID       int64  `cli:"ID,key"`
		Created  string `cli:"Created"`
		Project  string `cli:"Project"`
		Workflow string `cli:"Workflow"`
		Params   string `cli:"Params"`
	}

	tids := make([]TemplateInstanceDisplay, len(wtis))
	for i := range wtis {
		tids[i].ID = wtis[i].ID
		tids[i].Created = fmt.Sprintf("On %s by %s", wtis[i].FirstAudit.Created.Format(time.RFC3339),
			wtis[i].FirstAudit.AuditCommon.TriggeredBy)
		tids[i].Project = wtis[i].Project.Name
		if wtis[i].Workflow != nil {
			tids[i].Workflow = wtis[i].Workflow.Name
		} else {
			tids[i].Workflow = fmt.Sprintf("%s (not imported)", wtis[i].WorkflowName)
		}
		for k, v := range wtis[i].Request.Parameters {
			tids[i].Params = fmt.Sprintf("%s%s:%s\n", tids[i].Params, k, v)
		}
	}

	return cli.AsListResult(tids), nil
}
