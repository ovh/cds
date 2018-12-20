package main

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"

	repo "github.com/fsamin/go-repo"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

func templateApplyCmd(name string) cli.Command {
	return cli.Command{
		Name:    name,
		Short:   "Apply CDS workflow template",
		Example: "cdsctl template apply project-key workflow-name group-name/template-slug",
		Ctx: []cli.Arg{
			{Name: _ProjectKey},
			{Name: _WorkflowName, AllowEmpty: true},
		},
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
			{
				Kind:    reflect.Bool,
				Name:    "import-as-code",
				Usage:   "If true, will import the generated workflow as code on given project",
				Default: "false",
			},
			{
				Kind:    reflect.Bool,
				Name:    "import-push",
				Usage:   "If true, will push the generated workflow on given project",
				Default: "false",
			},
		},
	}
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

	return template, nil
}

func suggestTemplate() (*sdk.WorkflowTemplate, error) {
	wts, err := client.TemplateGetAll()
	if err != nil {
		return nil, err
	}
	opts := make([]string, len(wts))
	for i := range wts {
		opts[i] = fmt.Sprintf("%s (%s/%s)", wts[i].Name, wts[i].Group.Name, wts[i].Slug)
	}
	selected := cli.MultiChoice("Choose the CDS template to apply:", opts...)
	return &wts[selected], nil
}

func templateApplyRun(v cli.Values) error {
	projectKey := v.GetString(_ProjectKey)
	workflowName := v.GetString(_WorkflowName)

	var wti *sdk.WorkflowTemplateInstance
	var err error
	if workflowName != "" {
		// try to get an existing template instance for current workflow
		wti, err = client.WorkflowTemplateInstanceGet(projectKey, workflowName)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}
	}

	wt, err := getTemplateFromCLI(v)
	if err != nil {
		return err
	}

	// if no template given from args, and exiting instance try to get it's template
	if wt == nil && wti != nil {
		wt, err = client.TemplateGetByID(wti.WorkflowTemplateID)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}
	}

	// if no template found for workflow or no instance, suggest one
	if wt == nil {
		wt, err = suggestTemplate()
		if err != nil {
			return err
		}
	}

	// init params map from previous template instance if exists
	params := map[string]string{}
	if wti != nil {
		for _, p := range wt.Parameters {
			if v, ok := wti.Request.Parameters[p.Key]; ok {
				params[p.Key] = v
			}
		}
	}

	// set params from cli flags
	paramPairs := v.GetStringSlice("params")
	for _, p := range paramPairs {
		if p != "" { // when no params given GetStringSlice returns one empty string
			ps := strings.Split(p, "=")
			if len(ps) < 2 {
				return fmt.Errorf("Invalid given param %s", ps[0])
			}
			params[ps[0]] = strings.Join(ps[1:], "=")
		}
	}

	importPush := v.GetBool("import-push")
	importAsCode := v.GetBool("import-as-code")

	// try to find existing .git repository
	var localRepoURL string
	var localRepoName string
	r, err := repo.New(".")
	if err == nil {
		localRepoURL, err = r.FetchURL()
		if err != nil {
			return err
		}
		localRepoName, err = r.Name()
		if err != nil {
			return err
		}
	}

	// ask interactively for params if prompt not disabled
	if !v.GetBool("no-interactive") {
		if workflowName == "" {
			if localRepoName != "" {
				ss := strings.Split(localRepoName, "/")
				if len(ss) == 2 && cli.AskForConfirmation(fmt.Sprintf("Use the current repository name '%s' as workflow name", ss[1])) {
					workflowName = ss[1]
				}
			}
			// if no repo or current repo name not used
			if workflowName == "" {
				workflowName = cli.AskValueChoice("Give a valid name for the new generated workflow")
			}
		}

		var listRepositories []string
		var localRepoPath string

		// if there are params of type repository in list of params to fill prepare
		// the list of repositories for project
		var withRepository bool
		for _, p := range wt.Parameters {
			if _, ok := params[p.Key]; !ok {
				if p.Type == sdk.ParameterTypeRepository {
					withRepository = true
					break
				}
			}
		}
		if withRepository {
			// try to get the project from cds
			p, err := client.ProjectGet(projectKey)
			if err != nil {
				return err
			}

			for _, vcs := range p.VCSServers {
				rs, err := client.RepositoriesList(p.Key, vcs.Name)
				if err != nil {
					return err
				}
				for _, r := range rs {
					path := fmt.Sprintf("%s/%s", vcs.Name, r.Fullname)
					if localRepoURL != "" && (localRepoURL == r.HTTPCloneURL || localRepoURL == r.SSHCloneURL) {
						localRepoPath = path
					}
					listRepositories = append(listRepositories, path)
				}
			}
		}

		// for each param not already fill ask for the value
		for _, p := range wt.Parameters {
			if _, ok := params[p.Key]; !ok {
				label := fmt.Sprintf("Value for param '%s' (type: %s, required: %t)", p.Key, p.Type, p.Required)

				var choice string
				switch p.Type {
				case sdk.ParameterTypeRepository:
					if localRepoPath != "" && cli.AskForConfirmation(fmt.Sprintf("Use detected repository '%s' for param '%s'", localRepoPath, p.Key)) {
						choice = localRepoPath
					} else if len(listRepositories) > 0 {
						selected := cli.MultiChoice(label, listRepositories...)
						choice = listRepositories[selected]
					}
				case sdk.ParameterTypeBoolean:
					choice = fmt.Sprintf("%t", cli.AskForConfirmation(fmt.Sprintf("Set value to 'true' for param '%s'", p.Key)))
				}
				if choice == "" {
					choice = cli.AskValueChoice(label)
				}

				params[p.Key] = choice
			}
		}

		if !importAsCode && !importPush {
			if localRepoURL != "" {
				importAsCode = cli.AskForConfirmation(fmt.Sprintf("Import the generated workflow as code to the %s project", projectKey))
			}
			if !importAsCode {
				importPush = cli.AskForConfirmation(fmt.Sprintf("Push the generated workflow to the %s project", projectKey))
			}
		}
	}

	if importAsCode && localRepoURL == "" {
		return fmt.Errorf("Can't import current workflow because no local repository was found")
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
		ProjectKey:   projectKey,
		WorkflowName: workflowName,
		Parameters:   params,
	}
	if err := wt.CheckParams(req); err != nil {
		return err
	}

	tr, err := client.TemplateApply(wt.Group.Name, wt.Slug, req)
	if err != nil {
		return err
	}

	// import or push the generated workflow if one option is set
	if importAsCode || importPush {
		var buf bytes.Buffer
		tr, err = teeTarReader(tr, &buf)
		if err != nil {
			return err
		}

		var msgList []string
		if importAsCode {
			msgList, _, err = client.WorkflowPush(projectKey, bytes.NewBuffer(buf.Bytes()), []cdsclient.RequestModifier{
				func(r *http.Request) { r.Header.Set(sdk.WorkflowAsCodeHeader, localRepoURL) },
			}...)
		} else {
			msgList, _, err = client.WorkflowPush(projectKey, bytes.NewBuffer(buf.Bytes()))
		}
		for _, msg := range msgList {
			fmt.Println(msg)
		}
		if err != nil {
			return err
		}

		// store the choosen workflow name to git config
		if localRepoName != "" {
			if err := r.LocalConfigSet("cds", "workflow", workflowName); err != nil {
				return err
			}
		}

		fmt.Println("Workflow successfully pushed !")
	}

	return workflowTarReaderToFiles(dir, tr, v.GetBool("force"), v.GetBool("quiet"))
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
