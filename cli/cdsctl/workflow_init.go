package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	repo "github.com/fsamin/go-repo"
	"github.com/rockbears/log"
	giturls "github.com/whilp/git-urls"

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
			Usage: "(Optional) Set the repository remote URL. Default is the fetch URL",
		},
		{
			Name:  "repository-fullname",
			Usage: "(Optional) Set the repository fullname defined in repository manager",
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
			Usage: "(Optional) Set the root pipeline you want to use. If empty it will propose you to reuse of create a pipeline.",
		},
		{
			Name:  "application",
			Usage: "(Optional) Set the application name. If empty, it will deduce application name from the repository.",
		},
		{
			Name:  "workflow",
			Usage: "(Optional) Set the workflow name. If empty, it will deduce workflow name from the repository.",
		},
		{
			Name:      "yes",
			ShortHand: "y",
			Type:      cli.FlagBool,
			Usage:     "Automatic yes to prompts. Assume \"yes\" as answer to all prompts and run non-interactively.",
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
	selected := cli.AskChoice("Choose the CDS project:", opts...)
	chosenProj = &projs[selected]

	if err := gitRepo.LocalConfigSet(context.Background(), "cds", "project", chosenProj.Key); err != nil {
		return "", err
	}

	return chosenProj.Key, nil
}

func interactiveChooseVCSServer(proj *sdk.Project, gitRepo repo.Repo) (string, error) {
	switch len(proj.VCSServers) {
	case 0:
		//TODO ask to link the project
		return "", cli.NewError("your CDS project must be linked to a repositories manager to perform this operation")
	case 1:
		return proj.VCSServers[0].Name, nil
	default:
		fetchURL, err := gitRepo.FetchURL(context.Background())
		if err != nil {
			return "", cli.WrapError(err, "Unable to get remote URL")
		}

		originURL, err := giturls.Parse(fetchURL)
		if err != nil {
			return "", cli.WrapError(err, "Unable to parse remote URL")
		}
		originHost := strings.TrimSpace(strings.SplitN(originURL.Host, ":", 2)[0])

		vcsConf, err := client.VCSConfiguration()
		if err != nil {
			return "", cli.WrapError(err, "Unable to get VCS Configuration")
		}

		for rmName, cfg := range vcsConf {
			rmURL, err := url.Parse(cfg.URL)
			if err != nil {
				return "", cli.WrapError(err, "Unable to get VCS Configuration")
			}
			rmHost := strings.TrimSpace(strings.SplitN(rmURL.Host, ":", 2)[0])
			if originHost == rmHost {
				fmt.Printf(" * using repositories manager %s (%s)", cli.Magenta(rmName), cli.Magenta(rmURL.String()))
				fmt.Println()
				return rmName, nil
			}
		}
	}

	// Ask the user to choose the repository
	repoManagerNames := make([]string, len(proj.VCSServers))
	for i, s := range proj.VCSServers {
		repoManagerNames[i] = s.Name
	}

	selected := cli.AskChoice("Choose the repository manager:", repoManagerNames...)
	return proj.VCSServers[selected].Name, nil
}

func interactiveChooseApplication(pkey, repoFullname, repoName string) (string, *sdk.Application, error) {
	// Try to find application or create a new one from the repo
	apps, err := client.ApplicationList(pkey)
	if err != nil {
		return "", nil, cli.WrapError(err, "unable to list applications")
	}

	for i, a := range apps {
		if a.RepositoryFullname == repoFullname {
			fmt.Printf(" * application %s/%s (%s) found in CDS\n", cli.Magenta(a.ProjectKey), cli.Magenta(a.Name), cli.Magenta(a.RepositoryFullname))
			return a.Name, &apps[i], nil
		} else if a.Name == repoName {
			fmt.Printf(" * application %s/%s found in CDS.\n", cli.Magenta(a.ProjectKey), cli.Magenta(a.Name))
			fmt.Println(cli.Red(" * but it's not linked to repository"), cli.Red(repoFullname))
			if !cli.AskConfirm(cli.Red("Do you want to overwrite it?")) {
				return "", nil, cli.NewError("operation aborted")
			}
			return a.Name, nil, nil
		}
	}

	return repoName, nil, nil
}

func searchRepository(pkey, repoManagerName, repoFullname string) (string, error) {
	var resync bool
	for {
		// Get repositories from the repository manager
		repos, err := client.RepositoriesList(pkey, repoManagerName, resync)
		if err != nil {
			return "", cli.WrapError(err, "unable to list repositories from %s", repoManagerName)
		}

		// Check it the project with it's delegated oauth knows the current repo
		// Search the repo
		for _, r := range repos {
			// r.Fullname = CDS/demo
			if strings.ToLower(r.Fullname) == strings.ToLower(repoFullname) {
				fmt.Printf(" * using repository %s from %s", cli.Magenta(r.Fullname), cli.Magenta(repoManagerName))
				fmt.Println()
				return r.Fullname, nil
			}
		}
		if resync {
			break
		}
		resync = true
	}
	return "", cli.NewError("unable to find repository %s from %s: please check your credentials", repoFullname, repoManagerName)
}

// Check for given repo URL that it exists in project vcs.
// This func will resync the repositories list if not found at first time.
func checkRepositoryExists(proj sdk.Project, repoURL string) error {
	for _, vcs := range proj.VCSServers {
		var resync bool
	vcs:
		for {
			repos, err := client.RepositoriesList(proj.Key, vcs.Name, resync)
			if err != nil {
				log.Warn(context.Background(), "unable to list repositories from %s: %v", vcs.Name, err)
				break vcs
			}
			for _, r := range repos {
				if repoURL == r.HTTPCloneURL || repoURL == r.SSHCloneURL {
					return nil
				}
			}
			if resync {
				break
			}
			resync = true
		}
	}
	return cli.NewError("unable to find repository %s for project: please check project vcs configuration", repoURL)
}

func interactiveChoosePipeline(pkey, defaultPipeline string) (string, *sdk.Pipeline, error) {
	// Try to find pipeline or create a new pipeline
	pips, err := client.PipelineList(pkey)
	if err != nil {
		return "", nil, cli.WrapError(err, "unable to list pipelines")
	}
	if len(pips) == 0 {
		// If the project doesn't have any pipeline, lets return
		fmt.Printf(" * using new pipeline %s", cli.Magenta(defaultPipeline))
		fmt.Println()
		return defaultPipeline, nil, nil
	} else if defaultPipeline != "" {
		// Try to find the defaultPipeline in the list of pipelines
		for _, p := range pips {
			if p.Name == defaultPipeline {
				fmt.Printf(" * using pipeline %s/%s", cli.Magenta(pkey), cli.Magenta(defaultPipeline))
				fmt.Println()
				return defaultPipeline, &p, nil
			}
		}
		fmt.Printf(" * using new pipeline %s", cli.Magenta(defaultPipeline))
		fmt.Println()
		return defaultPipeline, nil, nil
	}

	pipelineNames := make([]string, len(pips))
	for i, p := range pips {
		pipelineNames[i] = p.Name
	}
	pipelineNames = append([]string{"new pipeline"}, pipelineNames...)
	selected := cli.AskChoice("Choose your pipeline:", pipelineNames...)

	if selected == 0 {
		fmt.Print("Enter your pipeline name: ")
		pipName := cli.ReadLine()
		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(pipName) {
			return "", nil, cli.NewError("Pipeline name '%s' do not respect pattern %s", pipName, sdk.NamePattern)
		}
		return pipName, nil, nil
	}
	return pips[selected-1].Name, &pips[selected-1], nil
}

func craftWorkflowFile(workflowName, appName, pipName, destinationDir string) (string, error) {
	// Crafting the workflow
	wkflw := exportentities.InitWorkflow(workflowName, appName, pipName)
	b, err := exportentities.Marshal(wkflw, exportentities.FormatYAML)
	if err != nil {
		return "", cli.WrapError(err, "Unable to write workflow file format")
	}

	wFilePath := filepath.Join(destinationDir, workflowName+".yml")
	if err := os.WriteFile(wFilePath, b, os.FileMode(0644)); err != nil {
		return "", cli.WrapError(err, "Unable to write workflow file")
	}

	fmt.Printf("File %s has been created\n", cli.Cyan(wFilePath))
	return wFilePath, nil
}

func craftApplicationFile(proj *sdk.Project, existingApp *sdk.Application, fetchURL, appName, repoFullname, repoManagerName, defaultSSHKey, defaultPGPKey, destinationDir string) (string, error) {
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

	// First collect all PGP and SSSH Keys/
	// And try to find the chosen key
	projectPGPKeys := make([]sdk.ProjectKey, 0, len(proj.Keys))
	projectSSHKeys := make([]sdk.ProjectKey, 0, len(proj.Keys))
	for i := range proj.Keys {
		switch proj.Keys[i].Type {
		case "pgp":
			projectPGPKeys = append(projectPGPKeys, proj.Keys[i])
			if defaultPGPKey == proj.Keys[i].Name {
				app.VCSPGPKey = proj.Keys[i].Name
				break
			}
		case "ssh":
			projectSSHKeys = append(projectSSHKeys, proj.Keys[i])
			if defaultSSHKey == proj.Keys[i].Name {
				app.VCSSSHKey = proj.Keys[i].Name
				break
			}
		}
	}

	if app.VCSPGPKey == "" {
		if defaultPGPKey != "" {
			if !strings.HasPrefix(defaultPGPKey, "app-pgp-") {
				defaultPGPKey = fmt.Sprintf("app-pgp-%s", defaultPGPKey)
			}
			// The key is unknown, we have to create a new one
			app.VCSPGPKey = defaultPGPKey
			app.Keys[app.VCSPGPKey] = exportentities.KeyValue{Type: string(sdk.KeyTypePGP)}

			fmt.Printf(" * using PGP Key %s/%s for application VCS settings", cli.Magenta(proj.Key), cli.Magenta(app.VCSPGPKey))
			fmt.Println()
		} else {
			// ask for pgp key, if not selected or no existing key create a new one.
			if len(projectPGPKeys) > 1 {
				opts := make([]string, len(projectPGPKeys)+1)
				opts[0] = "Use a new PGP key"
				for i := range projectPGPKeys {
					opts[i+1] = projectPGPKeys[i].Name
				}
				selected := cli.AskChoice("Select a PGP key to use in application VCS strategy", opts...)
				if selected > 0 {
					app.VCSPGPKey = opts[selected]
				} else {
					app.VCSPGPKey = fmt.Sprintf("app-pgp-%s", repoManagerName)
					app.Keys[app.VCSPGPKey] = exportentities.KeyValue{Type: string(sdk.KeyTypePGP)}
				}
			} else if len(projectPGPKeys) == 1 {
				app.VCSPGPKey = projectPGPKeys[0].Name

				fmt.Printf(" * using PGP Key %s/%s for application VCS settings", cli.Magenta(proj.Key), cli.Magenta(app.VCSPGPKey))
				fmt.Println()
			}
		}
	}

	// ask for ssh key if connection type is ssh, if not selected or no existing key create a new one
	if connectionType == "ssh" {

		if app.VCSSSHKey == "" {
			if defaultSSHKey != "" {
				// The key is unknown, we have to create a new one
				if !strings.HasPrefix(defaultSSHKey, "app-ssh-") {
					defaultSSHKey = fmt.Sprintf("app-ssh-%s", defaultSSHKey)
				}

				app.VCSSSHKey = defaultSSHKey
				app.Keys[app.VCSSSHKey] = exportentities.KeyValue{Type: string(sdk.KeyTypeSSH)}

				fmt.Printf(" * using SSH Key %s/%s for application VCS settings", cli.Magenta(proj.Key), cli.Magenta(app.VCSSSHKey))
				fmt.Println()
			} else {
				// ask for ssh key, if not selected or no existing key create a new one.
				if len(projectSSHKeys) > 1 {
					var opts = []string{
						"Use a new SSH key",
					}
					for i := range projectSSHKeys {
						opts = append(opts, projectSSHKeys[i].Name)
					}
					selected := cli.AskChoice("Select a SSH key to use in application VCS strategy", opts...)
					if selected > 0 {
						app.VCSSSHKey = opts[selected]
					} else {
						app.VCSSSHKey = fmt.Sprintf("app-ssh-%s", repoManagerName)
						app.Keys[app.VCSSSHKey] = exportentities.KeyValue{Type: string(sdk.KeyTypePGP)}
					}
				} else if len(projectSSHKeys) == 1 {
					app.VCSSSHKey = projectSSHKeys[0].Name

					fmt.Printf(" * using SSH Key %s/%s for application VCS settings", cli.Magenta(proj.Key), cli.Magenta(app.VCSSSHKey))
					fmt.Println()
				}
			}
		}

	}

	b, err := exportentities.Marshal(app, exportentities.FormatYAML)
	if err != nil {
		return "", cli.WrapError(err, "Unable to write application file format")
	}

	appFilePath := filepath.Join(destinationDir, fmt.Sprintf(exportentities.PullApplicationName, appName))
	if err := os.WriteFile(appFilePath, b, os.FileMode(0644)); err != nil {
		return "", cli.WrapError(err, "Unable to write application file")
	}

	fmt.Printf("File %s has been created\n", cli.Cyan(appFilePath))
	return appFilePath, nil
}

func craftPipelineFile(proj *sdk.Project, existingPip *sdk.Pipeline, pipName, destinationDir string) (string, error) {
	// Crafting the pipeline
	if existingPip != nil {
		return "", nil
	}

	checkout := exportentities.StepCheckout("{{.cds.workspace}}")
	pip := exportentities.PipelineV1{
		Name:    pipName,
		Version: exportentities.PipelineVersion1,
		Jobs: []exportentities.Job{
			{
				Name: "First job",
				Steps: []exportentities.Step{
					{
						Checkout: &checkout,
					},
				},
			},
		},
	}

	b, err := exportentities.Marshal(pip, exportentities.FormatYAML)
	if err != nil {
		return pipName, cli.WrapError(err, "Unable to write pipeline file format")
	}

	pipFilePath := filepath.Join(destinationDir, fmt.Sprintf(exportentities.PullPipelineName, pipName))
	if err := os.WriteFile(pipFilePath, b, os.FileMode(0644)); err != nil {
		return pipName, cli.WrapError(err, "Unable to write application file")
	}

	fmt.Printf("File %s has been created\n", cli.Cyan(pipFilePath))
	return pipFilePath, nil
}

func workflowInitRun(c cli.Values) error {
	path := "."
	ctx := context.Background()
	gitRepo, err := repo.New(ctx, path)
	if err != nil {
		return err
	}

	pkey, err := interactiveChooseProject(gitRepo, c.GetString(_ProjectKey))
	if err != nil {
		return err
	}

	// Check if the project is linked to a repository
	proj, err := client.ProjectGet(pkey, cdsclient.WithKeys())
	if err != nil {
		return cli.WrapError(err, "unable to get project")
	}

	repoFullname := c.GetString("repository-fullname")
	if repoFullname == "" {
		repoFullname, err = gitRepo.Name(ctx)
		if err != nil {
			return cli.WrapError(err, "unable to retrieve repository name")
		}
	}

	fullnames := strings.SplitN(repoFullname, "/", 2)
	repoShortname := fullnames[1]

	fetchURL := c.GetString("repository-url")
	if fetchURL == "" {
		fetchURL, err = gitRepo.FetchURL(ctx)
		if err != nil {
			return cli.WrapError(err, "unable to retrieve origin URL")
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

	if len(files) > 0 {
		if c.GetString("pipeline") != "" {
			return errors.New("you can't set a pipeline name while files already exists in .cds/ folder")
		}
	}

	workflowName := repoShortname
	if c.GetString("workflow") != "" {
		workflowName = c.GetString("workflow")
	}

	if len(files) == 0 {
		repoManagerName, err := interactiveChooseVCSServer(proj, gitRepo)
		if err != nil {
			return cli.WrapError(err, "unable to get vcs server")
		}

		repoFullname, err = searchRepository(pkey, repoManagerName, repoFullname)
		if err != nil {
			return err
		}

		appName, existingApp, err := interactiveChooseApplication(pkey, repoFullname, repoShortname)
		if err != nil {
			return err
		}

		if c.GetString("application") != "" {
			appName = c.GetString("application")
		}

		pipDefaultName := c.GetString("pipeline")
		if pipDefaultName == "" {
			pipDefaultName = fmt.Sprintf("%s-build", appName)
		}
		pipName, existingPip, err := interactiveChoosePipeline(pkey, pipDefaultName)
		if err != nil {
			return err
		}

		wFilePath, err := craftWorkflowFile(workflowName, appName, pipName, dotCDS)
		if err != nil {
			return err
		}
		files = []string{wFilePath}

		appFilePath, err := craftApplicationFile(proj, existingApp, fetchURL, appName, repoFullname, repoManagerName, c.GetString("repository-ssh-key"), c.GetString("repository-pgp-key"), dotCDS)
		if err != nil {
			return err
		}
		if appFilePath != "" {
			files = append(files, appFilePath)
		}

		pipFilePath, err := craftPipelineFile(proj, existingPip, pipName, dotCDS)
		if err != nil {
			return err
		}
		if pipFilePath != "" {
			files = append(files, pipFilePath)
		}
	} else {
		// Check that current repository is accessible for CDS, also resync repositories
		if err := checkRepositoryExists(*proj, fetchURL); err != nil {
			return err
		}
		fmt.Println("Reading files:")
		for _, f := range files {
			fmt.Printf(" * %s", cli.Magenta(f))
			fmt.Println()
		}
	}

	if !c.GetBool("yes") && !cli.AskConfirm(cli.Red("CDS Files are ready, continue ?")) {
		return nil
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
	if err := gitRepo.LocalConfigSet(ctx, "cds", "project", proj.Key); err != nil {
		fmt.Printf("error: unable to setup git local config to store cds project key: %v\n", err)
	}

	if err := gitRepo.LocalConfigSet(ctx, "cds", "workflow", workflowName); err != nil {
		fmt.Printf("error: unable to setup git local config to store cds workflow name: %v\n", err)
	}

	fmt.Printf("Now you can run: ")
	fmt.Printf(cli.Magenta("git add %s/ && git commit -s -m \"chore: init CDS workflow files\"\n", dotCDS))

	return nil
}
