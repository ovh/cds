package main

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
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
				Type:      cli.FlagArray,
				Name:      "params",
				ShortHand: "p",
				Usage:     "Specify params for template like --params paramKey=paramValue",
				Default:   "",
			},
			{
				Type:    cli.FlagBool,
				Name:    "detach",
				Usage:   "Set to generate a workflow detached from the template",
				Default: "",
			},
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
			{
				Type:    cli.FlagBool,
				Name:    "import-as-code",
				Usage:   "If true, will import the generated workflow as code on given project",
				Default: "false",
			},
			{
				Type:    cli.FlagBool,
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
		groupName, templateSlug, err := cli.ParsePath(templatePath)
		if err != nil {
			return nil, err
		}

		// try to get the template from cds
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
	if len(wts) == 0 {
		return nil, fmt.Errorf("no existing template found from CDS")
	}
	opts := make([]string, len(wts))
	for i := range wts {
		opts[i] = fmt.Sprintf("%s (%s/%s)", wts[i].Name, wts[i].Group.Name, wts[i].Slug)
	}
	selected := cli.AskChoice("Choose the CDS template to apply:", opts...)
	return &wts[selected], nil
}

func templateApplyRun(v cli.Values) error {
	projectKey := v.GetString(_ProjectKey)
	workflowName := v.GetString(_WorkflowName)

	var existingWorkflow *sdk.Workflow
	var existingWorkflowTemplateInstance *sdk.WorkflowTemplateInstance
	var err error
	if workflowName != "" {
		// try to get an existing template instance for current workflow
		existingWorkflow, err = client.WorkflowGet(projectKey, workflowName, cdsclient.WithTemplate())
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}
		if existingWorkflow != nil && existingWorkflow.TemplateInstance != nil {
			existingWorkflowTemplateInstance = existingWorkflow.TemplateInstance
		}
	}

	wt, err := getTemplateFromCLI(v)
	if err != nil {
		return err
	}

	// if no template given from args, and exiting instance try to get it's template
	if wt == nil && existingWorkflowTemplateInstance != nil {
		wt = existingWorkflowTemplateInstance.Template
	}

	// if no template found for workflow or no instance, suggest one
	if wt == nil {
		if v.GetBool("no-interactive") {
			return fmt.Errorf("you should give a template path")
		}
		wt, err = suggestTemplate()
		if err != nil {
			return err
		}
	}

	// init params map from previous template instance if exists
	params := make(map[string]string)
	if existingWorkflowTemplateInstance != nil {
		for _, p := range wt.Parameters {
			if v, ok := existingWorkflowTemplateInstance.Request.Parameters[p.Key]; ok {
				params[p.Key] = v
			}
		}
	}

	// set params from cli flags
	paramPairs := v.GetStringArray("params")
	for _, p := range paramPairs {
		ps := strings.Split(p, "=")
		if len(ps) < 2 {
			return fmt.Errorf("Invalid given param %s", ps[0])
		}
		params[ps[0]] = strings.Join(ps[1:], "=")
	}

	// Import flags are not allowed if an existing ascode workflow exists
	importPush := v.GetBool("import-push") && (existingWorkflow == nil || existingWorkflow.FromRepository == "")
	importAsCode := v.GetBool("import-as-code") && (existingWorkflow == nil || existingWorkflow.FromRepository == "")
	detached := v.GetBool("detach")

	// try to find existing .git repository
	var localRepoURL string
	var localRepoName string
	ctx := context.Background()
	r, err := repo.New(ctx, ".")
	if err == nil {
		localRepoURL, err = r.FetchURL(ctx)
		if err != nil {
			return err
		}
		localRepoName, err = r.Name(ctx)
		if err != nil {
			return err
		}
	}

	// ask interactively for params if prompt not disabled
	if !v.GetBool("no-interactive") {
		if workflowName == "" {
			if localRepoName != "" {
				ss := strings.Split(localRepoName, "/")
				if len(ss) == 2 && cli.AskConfirm(fmt.Sprintf("Use the current repository name '%s' as workflow name", ss[1])) {
					workflowName = ss[1]
				}
			}
			// if no repo or current repo name not used
			if workflowName == "" {
				workflowName = cli.AskValue("Give a valid name for the new generated workflow")
			}
		}

		var listRepositories []string
		var listSSHKeys []string
		var listPGPKeys []string
		var localRepoPath string

		// if there are params of type repository in list of params to fill prepare
		// the list of repositories for project
		var withRepository bool
		var withKey bool
		for _, p := range wt.Parameters {
			if _, ok := params[p.Key]; !ok {
				if p.Type == sdk.ParameterTypeRepository {
					withRepository = true
				}
				if p.Type == sdk.ParameterTypeSSHKey || p.Type == sdk.ParameterTypePGPKey {
					withKey = true
				}
				if withRepository && withKey {
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
				rs, err := client.RepositoriesList(p.Key, vcs.Name, false)
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
		if withKey {
			pKeys, err := client.ProjectKeysList(projectKey)
			if err != nil {
				return err
			}
			for _, k := range pKeys {
				switch k.Type {
				case sdk.KeyTypeSSH:
					listSSHKeys = append(listSSHKeys, k.Name)
				case sdk.KeyTypePGP:
					listPGPKeys = append(listPGPKeys, k.Name)
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
					if localRepoPath != "" && cli.AskConfirm(fmt.Sprintf("Use detected repository '%s' for param '%s'", localRepoPath, p.Key)) {
						choice = localRepoPath
					} else if len(listRepositories) > 0 {
						selected := cli.AskChoice(label, listRepositories...)
						choice = listRepositories[selected]
					}
				case sdk.ParameterTypeSSHKey:
					if len(listSSHKeys) > 0 {
						selected := cli.AskChoice(label, listSSHKeys...)
						choice = listSSHKeys[selected]
					}
				case sdk.ParameterTypePGPKey:
					if len(listPGPKeys) > 0 {
						selected := cli.AskChoice(label, listPGPKeys...)
						choice = listPGPKeys[selected]
					}
				case sdk.ParameterTypeBoolean:
					choice = fmt.Sprintf("%t", cli.AskConfirm(fmt.Sprintf("Set value to 'true' for param '%s'", p.Key)))
				}
				if choice == "" {
					choice = cli.AskValue(label)
				}

				params[p.Key] = choice
			}
		}

		// We ask for import only if there is no existing workflow or if exists but not ascode
		if !importAsCode && !importPush && (existingWorkflow == nil || existingWorkflow.FromRepository == "") {
			if localRepoURL != "" {
				importAsCode = cli.AskConfirm(fmt.Sprintf("Import the generated workflow as code to the %s project", projectKey))
			}
			if !importAsCode {
				importPush = cli.AskConfirm(fmt.Sprintf("Push the generated workflow to the %s project", projectKey))
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

	// Check request before submit
	req := sdk.WorkflowTemplateRequest{
		ProjectKey:   projectKey,
		WorkflowName: workflowName,
		Parameters:   params,
		Detached:     detached,
	}
	if err := wt.CheckParams(req); err != nil {
		return err
	}

	tr, err := client.TemplateApply(wt.Group.Name, wt.Slug, req)
	if err != nil {
		return err
	}

	// Import or push the generated workflow if one option is set
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

		// store the chosen workflow name to git config
		if localRepoName != "" {
			if err := r.LocalConfigSet(ctx, "cds", "workflow", workflowName); err != nil {
				return err
			}
		}

		fmt.Println("Workflow successfully pushed !")
	}

	return workflowTarReaderToFiles(v, dir, tr)
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
