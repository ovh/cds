package main

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/cli"
	"github.com/spf13/cobra"
)

var (
	templateCmd = cli.Command{
		Name:  "template",
		Short: "Manage CDS workflow template",
	}

	template = cli.NewCommand(templateCmd, nil,
		[]*cobra.Command{
			cli.NewCommand(templateApplyCmd, templateApplyRun, nil, withAllCommandModifiers()...),
		})
)

var templateApplyCmd = cli.Command{
	Name:    "apply",
	Short:   "Apply CDS workflow template",
	Example: "cdsctl template apply group-name/template-slug PROJKEY workflow-name",
	OptionalArgs: []cli.Arg{
		{Name: "template-path"},
	},
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
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
			Name:      "ignore-prompt",
			ShortHand: "i",
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
		{
			Kind:    reflect.Bool,
			Name:    "push",
			Usage:   "If true, will push the generated workflow on given project",
			Default: "false",
		},
	},
}

func getTemplateFromCLI(v cli.Values) (*sdk.WorkflowTemplate, error) {
	var template *sdk.WorkflowTemplate

	// search template path from params or suggest one
	templatePath := v.GetString("template-path")
	if templatePath != "" {
		templatePathSplitted := strings.Split(templatePath, "/")
		if len(templatePathSplitted) != 2 {
			return nil, fmt.Errorf("Invalid given template path")
		}

		groupName, templateSlug := templatePathSplitted[0], templatePathSplitted[1]

		// try to get the template from cds
		var err error
		template, err = client.TemplateGet(groupName, templateSlug)
		if err != nil {
			return nil, err
		}
	}

	// if no template given, suggest one
	if template == nil {
		templates, err := client.TemplateGetAll()
		if err != nil {
			return nil, err
		}

		opts := make([]string, len(templates))
		for i := 0; i < len(templates); i++ {
			opts[i] = fmt.Sprintf("%s (%s/%s) - %s", templates[i].Name, templates[i].Group.Name, templates[i].Slug, templates[i].Description)
		}
		selected := cli.MultiChoice("Choose the CDS template to apply", opts...)
		template = templates[selected]

		// TODO maybe store the selected template to git config for next runs
	}

	return template, nil
}

func getProjectFromCli(v cli.Values) (*sdk.Project, error) {
	projectKey := v.GetString(_ProjectKey)

	// search project from params or suggest one
	if projectKey != "" {

	}

	return nil, nil
}

func templateApplyRun(v cli.Values) error {
	if _, err := getTemplateFromCLI(v); err != nil {
		return err
	}

	if _, err := getProjectFromCli(v); err != nil {
		return err
	}

	return nil

	/*workflowPath := strings.Split(v.GetString("workflowPath"), "/")
	if len(workflowPath) != 2 {
		return fmt.Errorf("Invalid given workflow path")
	}
	projectKey, workflowSlug := workflowPath[0], workflowPath[1]

	// try to get the project from cds
	p, err := client.ProjectGet(projectKey)
	if err != nil {
		return err
	}

	// try to get an existing workflow instance from cds
	wti, err := client.WorkflowTemplateInstanceGet(p.Key, workflowSlug)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return err
	}
	old := map[string]string{}
	if wti != nil {
		// init old params from previous request
		for _, p := range wt.Parameters {
			if v, ok := wti.Request.Parameters[p.Key]; ok {
				old[p.Key] = v
			}
		}
	}

	// init params from cli flags
	paramPairs := v.GetStringSlice("params")
	params := map[string]string{}
	for _, p := range paramPairs {
		if p != "" { // when no params given GetStringSlice returns one empty string
			ps := strings.Split(p, "=")
			if len(ps) < 2 {
				return fmt.Errorf("Invalid given param %s", ps[0])
			}
			params[ps[0]] = strings.Join(ps[1:], "=")
		}
	}

	// for parameters not given with flags, ask interactively if not disabled
	if !v.GetBool("ignore-prompt") {
		// if there is parameters of type vcs or repository get suggestions from project
		listVCS := make([]string, len(p.VCSServers))
		suggestVCS := make(map[string][]string, len(p.VCSServers))
		for i, vcs := range p.VCSServers {
			listVCS[i] = vcs.Name
			suggestVCS[vcs.Name] = nil
		}
		var withRepository bool
		for _, parameter := range wt.Parameters {
			if parameter.Type == sdk.ParameterTypeRepository {
				withRepository = true
				break
			}
		}
		if withRepository {
			for name := range suggestVCS {
				res, err := client.RepositoriesList(p.Key, name)
				if err != nil {
					return err
				}
				suggestVCS[name] = make([]string, len(res))
				for i := 0; i < len(res); i++ {
					suggestVCS[name][i] = res[i].Slug
				}
			}
		}

		for _, p := range wt.Parameters {
			if _, ok := params[p.Key]; !ok {
				var oldValue string
				if o, ok := old[p.Key]; ok {
					oldValue = fmt.Sprintf(", old: %s", o)
				}
				label := fmt.Sprintf("Value for param %s (type: %s, required: %t%s): ", p.Key, p.Type, p.Required, oldValue)

				var choice string
				switch p.Type {
				case sdk.ParameterTypeRepository:
					var selectedVCS string
					if len(listVCS) > 0 {
						selected = cli.MultiChoice(fmt.Sprintf("Select a VCS needed to fill %s param", p.Key), listVCS...)
					}
					if selectedVCS != "" && len(suggestVCS[selectedVCS]) > 0 {
						selected := cli.MultiChoice(label, suggestVCS[selectedVCS]...)
						choice = fmt.Sprintf("%s/%s", selectedVCS, repo)
					}
				}
				if choice == "" {
					choice = cli.AskValueChoice(label)
				}

				params[p.Key] = choice
			}
		}
	}

	dir := strings.TrimSpace(v.GetString("output-dir"))
	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, os.FileMode(0744)); err != nil {
		return fmt.Errorf("Unable to create directory %s: %v", v.GetString("output-dir"), err)
	}

	// check request before submit
	req := sdk.WorkflowTemplateRequest{
		ProjectKey:   p.Key,
		WorkflowSlug: workflowSlug,
		Parameters:   params,
	}
	if err := wt.CheckParams(req); err != nil {
		return err
	}

	tr, err := client.TemplateApply(groupName, templateSlug, req)
	if err != nil {
		return err
	}

	// push the generated workflow if option set
	if v.GetBool("push") {
		var buf bytes.Buffer
		tr, err = teeTarReader(tr, &buf)
		if err != nil {
			return err
		}

		msgList, _, err := client.WorkflowPush(p.Key, bytes.NewBuffer(buf.Bytes()))
		for _, msg := range msgList {
			fmt.Println(msg)
		}
		if err != nil {
			return err
		}

		fmt.Println("Workflow successfully pushed !")
	}

	if err := workflowTarReaderToFiles(dir, tr, v.GetBool("force"), v.GetBool("quiet")); err != nil {
		return err
	}*/
}

func teeTarReader(r *tar.Reader, buf io.Writer) (*tar.Reader, error) {
	var b bytes.Buffer
	tw1, tw2 := tar.NewWriter(&b), tar.NewWriter(buf)

	for {
		hdr, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if err := tw1.WriteHeader(hdr); err != nil {
			return nil, err
		}
		if err := tw2.WriteHeader(hdr); err != nil {
			return nil, err
		}
		var bs bytes.Buffer
		if n, err := io.Copy(&bs, r); err != nil {
			return nil, err
		} else if n == 0 {
			return nil, fmt.Errorf("Nothing to read")
		}
		if n, err := tw1.Write(bs.Bytes()); err != nil {
			return nil, err
		} else if n == 0 {
			return nil, fmt.Errorf("Nothing to write")
		}
		if n, err := tw2.Write(bs.Bytes()); err != nil {
			return nil, err
		} else if n == 0 {
			return nil, fmt.Errorf("Nothing to write")
		}
	}

	tw1.Close()
	tw2.Close()

	return tar.NewReader(&b), nil
}
