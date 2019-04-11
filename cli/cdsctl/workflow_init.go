package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	repo "github.com/fsamin/go-repo"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/exportentities"
)

var workflowInitCmd = cli.Command{
	Name:  "init",
	Short: "Init a workflow",
	Long: `Initialize a workflow from your current repository, this will read or create yml files and push them to CDS.

Documentation: https://ovh.github.io/cds/docs/tutorials/init_workflow_with_cdsctl/

`,
	OptionalArgs: []cli.Arg{
		{Name: _ProjectKey},
	},
	Flags: []cli.Flag{
		{
			Name:  "repository-url",
			Usage: "(Optionnal) Set the repository remote URL. Default is the fetch URL",
		},
		{
			Name:  "repository-fullname",
			Usage: "(Optionnal) Set the repository fullname defined in repository manager",
		},
		{
			Name:  "repository-ssh-key",
			Usage: "Set the repository access key you want to use",
		},
		{
			Name:  "repository-pgp-key",
			Usage: "Set the repository pgp key you want to use",
		},
		{
			Name:  "pipeline",
			Usage: "(Optionnal) Set the root pipeline you want to use. If empty it will propose you to reuse of create a pipeline.",
		},
	},
}

func interactiveChooseProject(gitRepo repo.Repo, defaultValue string) (string, error) {
	if cfg.Verbose {
		fmt.Println("interactiveChooseProject: ", defaultValue)
	}
	if defaultValue != "" {
		return defaultValue, nil
	}

	projs, err := client.ProjectList(false, false)
	if err != nil {
		return "", err
	}

	var chosenProj *sdk.Project
	opts := make([]string, len(projs))
	for i := range projs {
		opts[i] = fmt.Sprintf("%s - %s", projs[i].Key, projs[i].Name)
	}
	selected := cli.MultiChoice("Choose the CDS project:", opts...)
	chosenProj = &projs[selected]

	if err := gitRepo.LocalConfigSet("cds", "project", chosenProj.Key); err != nil {
		return "", err
	}

	return chosenProj.Key, nil
}

func interactiveChooseVCSServer(proj *sdk.Project, gitRepo repo.Repo) (string, error) {
	switch len(proj.VCSServers) {
	case 0:
		//TODO ask to link the project
		return "", fmt.Errorf("your CDS project must be linked to a repositories manager to perform this operation")
	case 1:
		return proj.VCSServers[0].Name, nil
	default:

		fetchURL, err := gitRepo.FetchURL()
		if err != nil {
			return "", fmt.Errorf("Unable to get remote URL: %v", err)
		}

		originURL, err := url.Parse(fetchURL)
		if err != nil {
			return "", fmt.Errorf("Unable to parse remote URL: %v", err)
		}

		vcsConf, err := client.VCSConfiguration()
		if err != nil {
			return "", fmt.Errorf("Unable to get VCS Configuration: %v", err)
		}

		for rmName, cfg := range vcsConf {
			rmURL, err := url.Parse(cfg.URL)
			if err != nil {
				return "", fmt.Errorf("Unable to get VCS Configuration: %v", err)
			}
			if rmURL.Host == originURL.Host {
				return rmName, nil
			}
		}
	}

	// Ask the user to choose the repository
	repoManagerNames := make([]string, len(proj.VCSServers))
	for i, s := range proj.VCSServers {
		repoManagerNames[i] = s.Name
	}

	selected := cli.MultiChoice("Choose the repository manager:", repoManagerNames...)
	return proj.VCSServers[selected].Name, nil
}

func interactiveChooseApplication(pkey, repoFullname, repoName string) (string, *sdk.Application, error) {
	// Try to find application or create a new one from the repo
	apps, err := client.ApplicationList(pkey)
	if err != nil {
		return "", nil, fmt.Errorf("unable to list applications: %v", err)
	}

	for i, a := range apps {
		if a.RepositoryFullname == repoFullname {
			fmt.Printf("application %s/%s (%s) found in CDS\n", cli.Magenta(a.ProjectKey), cli.Magenta(a.Name), cli.Magenta(a.RepositoryFullname))
			return a.Name, &apps[i], nil
		} else if a.Name == repoName {
			fmt.Printf("application %s/%s found in CDS.\n", cli.Magenta(a.ProjectKey), cli.Magenta(a.Name))
			fmt.Println(cli.Red("But it's not linked to repository"), cli.Red(repoFullname))
			if !cli.AskForConfirmation(cli.Red("Do you want to overwrite it?")) {
				return "", nil, fmt.Errorf("operation aborted")
			}
			return a.Name, nil, nil
		}
	}

	return repoName, nil, nil
}

func searchRepository(pkey, repoManagerName, repoFullname string) (string, error) {
	// Get repositories from the repository manager
	repos, err := client.RepositoriesList(pkey, repoManagerName, true)
	if err != nil {
		return "", fmt.Errorf("unable to list repositories from %s: %v", repoManagerName, err)
	}

	// Check it the project with it's delegated oauth knows the current repo
	// Search the repo
	for _, r := range repos {
		// r.Fullname = CDS/demo
		if strings.ToLower(r.Fullname) == repoFullname {
			return r.Fullname, nil
		}
	}

	return "", fmt.Errorf("unable to find repository %s from %s: please check your credentials", repoFullname, repoManagerName)
}

func interactiveChoosePipeline(pkey, defaultPipeline string) (string, *sdk.Pipeline, error) {
	// Try to find pipeline or create a new pipeline
	pips, err := client.PipelineList(pkey)
	if err != nil {
		return "", nil, fmt.Errorf("unable to list pipelines: %v", err)
	}
	if len(pips) == 0 {
		// If the project doesn't have any pipeline, lets return
		return defaultPipeline, nil, nil
	} else if defaultPipeline != "" {
		// Try to find the defaultPipeline in the list of pipelines
		for _, p := range pips {
			if p.Name == defaultPipeline {
				return defaultPipeline, &p, nil
			}
		}
		return defaultPipeline, nil, nil
	}

	pipelineNames := make([]string, len(pips))
	for i, p := range pips {
		pipelineNames[i] = p.Name
	}
	pipelineNames = append([]string{"new pipeline"}, pipelineNames...)
	selected := cli.MultiChoice("Choose your pipeline:", pipelineNames...)

	if selected == 0 {
		fmt.Print("Enter your pipeline name: ")
		pipName := cli.ReadLine()
		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(pipName) {
			return "", nil, fmt.Errorf("Pipeline name '%s' do not respect pattern %s", pipName, sdk.NamePattern)
		}
		return pipName, nil, nil
	}
	return pips[selected-1].Name, &pips[selected-1], nil
}

func craftWorkflowFile(workflowName, appName, pipName, destinationDir string) (string, error) {
	// Crafting the workflow
	wkflw := exportentities.Workflow{
		Version:         exportentities.WorkflowVersion1,
		Name:            workflowName,
		ApplicationName: appName,
		PipelineName:    pipName,
	}

	b, err := exportentities.Marshal(wkflw, exportentities.FormatYAML)
	if err != nil {
		return "", fmt.Errorf("Unable to write workflow file format: %v", err)
	}

	wFilePath := filepath.Join(destinationDir, workflowName+".yml")
	if err := ioutil.WriteFile(wFilePath, b, os.FileMode(0644)); err != nil {
		return "", fmt.Errorf("Unable to write workflow file: %v", err)
	}

	fmt.Printf("File %s created\n", cli.Magenta(wFilePath))
	return wFilePath, nil
}

func craftApplicationFile(proj *sdk.Project, existingApp *sdk.Application, fetchURL, appName, repoFullname, repoManagerName, destinationDir string) (string, error) {
	if existingApp != nil {
		return "", nil
	}

	// Crafting the application
	connectionType := "ssh"
	if strings.HasPrefix(fetchURL, "https") {
		connectionType = "https"
	}

	app := exportentities.Application{
		Name:              appName,
		RepositoryName:    repoFullname,
		VCSServer:         repoManagerName,
		VCSConnectionType: connectionType,
		Keys:              map[string]exportentities.KeyValue{},
	}

	projectPGPKeys := make([]sdk.ProjectKey, 0, len(proj.Keys))
	projectSSHKeys := make([]sdk.ProjectKey, 0, len(proj.Keys))
	for i := range proj.Keys {
		switch proj.Keys[i].Type {
		case "pgp":
			projectPGPKeys = append(projectPGPKeys, proj.Keys[i])
		case "ssh":
			projectSSHKeys = append(projectSSHKeys, proj.Keys[i])
		}
	}

	// ask for pgp key, if not selected or no existing key create a new one.
	if len(projectPGPKeys) > 1 {
		opts := make([]string, len(projectPGPKeys)+1)
		opts[0] = "Use a new pgp key"
		for i := range projectPGPKeys {
			opts[i+1] = projectPGPKeys[i].Name
		}
		selected := cli.MultiChoice("Select a PGP key to use in application VCS strategy", opts...)
		if selected > 0 {
			app.VCSPGPKey = opts[selected]
		}
	} else if len(projectPGPKeys) == 1 {
		if cli.AskForConfirmation(fmt.Sprintf("Found one existing PGP key '%s' on project. Use it in application VCS strategy?", projectPGPKeys[0].Name)) {
			app.VCSPGPKey = projectPGPKeys[0].Name
		}
	}
	if app.VCSPGPKey == "" {
		app.VCSPGPKey = fmt.Sprintf("app-pgp-%s", repoManagerName)
		app.Keys[app.VCSPGPKey] = exportentities.KeyValue{Type: sdk.KeyTypePGP}
	}

	// ask for ssh key if connection type is ssh, if not selected or no existing key create a new one
	if connectionType == "ssh" {
		if len(projectSSHKeys) > 1 {
			opts := make([]string, len(projectSSHKeys)+1)
			opts[0] = "Use a new ssh key"
			for i := range projectSSHKeys {
				opts[i+1] = projectSSHKeys[i].Name
			}
			selected := cli.MultiChoice("Select a SSH key to use in application VCS strategy", opts...)
			if selected > 0 {
				app.VCSSSHKey = opts[selected]
			}
		} else if len(projectSSHKeys) == 1 {
			if cli.AskForConfirmation(fmt.Sprintf("Found one existing SSH key '%s' on project. Use it in application VCS strategy?", projectSSHKeys[0].Name)) {
				app.VCSSSHKey = projectSSHKeys[0].Name
			}
		}
		if app.VCSSSHKey == "" {
			app.VCSSSHKey = fmt.Sprintf("app-ssh-%s", repoManagerName)
			app.Keys[app.VCSSSHKey] = exportentities.KeyValue{Type: sdk.KeyTypeSSH}
		}
	}

	b, err := exportentities.Marshal(app, exportentities.FormatYAML)
	if err != nil {
		return "", fmt.Errorf("Unable to write application file format: %v", err)
	}

	appFilePath := filepath.Join(destinationDir, fmt.Sprintf(exportentities.PullApplicationName, appName))
	if err := ioutil.WriteFile(appFilePath, b, os.FileMode(0644)); err != nil {
		return "", fmt.Errorf("Unable to write application file: %v", err)
	}

	fmt.Printf("File %s created\n", cli.Magenta(appFilePath))
	return appFilePath, nil
}

func craftPipelineFile(proj *sdk.Project, existingPip *sdk.Pipeline, pipName, destinationDir string) (string, error) {
	// Crafting the pipeline
	if existingPip != nil {
		return "", nil
	}

	pip := exportentities.PipelineV1{
		Name:    pipName,
		Version: exportentities.PipelineVersion1,
		Jobs: []exportentities.Job{
			{
				Name: "First job",
				Steps: []exportentities.Step{
					{
						"checkout": "{{.cds.workspace}}",
					},
				},
			},
		},
	}

	b, err := exportentities.Marshal(pip, exportentities.FormatYAML)
	if err != nil {
		return pipName, fmt.Errorf("Unable to write pipeline file format: %v", err)
	}

	pipFilePath := filepath.Join(destinationDir, fmt.Sprintf(exportentities.PullPipelineName, pipName))
	if err := ioutil.WriteFile(pipFilePath, b, os.FileMode(0644)); err != nil {
		return pipName, fmt.Errorf("Unable to write application file: %v", err)
	}

	fmt.Printf("File %s created\n", cli.Magenta(pipFilePath))
	return pipFilePath, nil
}

func workflowInitRun(c cli.Values) error {
	path := "."
	gitRepo, errRepo := repo.New(path)
	if errRepo != nil {
		return errRepo
	}

	pkey, err := interactiveChooseProject(gitRepo, c.GetString(_ProjectKey))
	if err != nil {
		return err
	}

	// Check if the project is linked to a repository
	proj, err := client.ProjectGet(pkey, func(r *http.Request) {
		q := r.URL.Query()
		q.Set("withKeys", "true")
		r.URL.RawQuery = q.Encode()
	})
	if err != nil {
		return fmt.Errorf("unable to get project: %v", err)
	}

	repoFullname := c.GetString("repository-fullname")
	if repoFullname == "" {
		repoFullname, err = gitRepo.Name()
		if err != nil {
			return fmt.Errorf("unable to retrieve repository name: %v", err)
		}
	}

	fullnames := strings.SplitN(repoFullname, "/", 2)
	repoShortname := fullnames[1]

	fetchURL := c.GetString("repository-url")
	if fetchURL == "" {
		fetchURL, err = gitRepo.FetchURL()
		if err != nil {
			return fmt.Errorf("unable to retrieve origin URL: %v", err)
		}
	}

	fmt.Printf("Initializing workflow from %s (%v)...\n", cli.Magenta(repoFullname), cli.Magenta(fetchURL))

	dotCDS := filepath.Join(path, ".cds")
	if err := os.MkdirAll(dotCDS, os.FileMode(0755)); err != nil {
		return err
	}

	files, err := filepath.Glob(dotCDS + "/*.yml")
	if err != nil {
		return err
	}

	if len(files) == 0 {
		repoManagerName, err := interactiveChooseVCSServer(proj, gitRepo)
		if err != nil {
			return fmt.Errorf("unable to get vcs server: %v", err)
		}

		repoFullname, err = searchRepository(pkey, repoManagerName, repoFullname)
		if err != nil {
			return err
		}

		appName, existingApp, err := interactiveChooseApplication(pkey, repoFullname, repoShortname)
		if err != nil {
			return err
		}

		pipName, existingPip, err := interactiveChoosePipeline(pkey, c.GetString("pipeline"))
		if err != nil {
			return err
		}

		wFilePath, err := craftWorkflowFile(repoShortname, appName, pipName, dotCDS)
		if err != nil {
			return err
		}
		files = []string{wFilePath}

		appFilePath, err := craftApplicationFile(proj, existingApp, fetchURL, appName, repoFullname, repoManagerName, dotCDS)
		if err != nil {
			return err
		}
		files = append(files, appFilePath)

		pipFilePath, err := craftPipelineFile(proj, existingPip, pipName, dotCDS)
		if err != nil {
			return err
		}
		files = append(files, pipFilePath)
	}

	buf := new(bytes.Buffer)
	if err := workflowFilesToTarWriter(files, buf); err != nil {
		return err
	}

	fmt.Println("Pushing workflow to CDS...")
	mods := []cdsclient.RequestModifier{
		func(r *http.Request) {
			r.Header.Set(sdk.WorkflowAsCodeHeader, fetchURL)
		},
	}

	msgList, tr, err := client.WorkflowPush(pkey, buf, mods...)
	for _, msg := range msgList {
		fmt.Println("\t" + msg)
	}
	if err != nil {
		return err
	}

	if err := workflowTarReaderToFiles(c, dotCDS, tr); err != nil {
		return err
	}

	//Configure local git
	if err := gitRepo.LocalConfigSet("cds", "workflow", repoShortname); err != nil {
		return err
	}

	fmt.Printf("Now you can run: ")
	fmt.Printf(cli.Magenta("git add %s/ && git commit -s -m \"chore: init CDS workflow files\"\n", dotCDS))

	return nil
}
