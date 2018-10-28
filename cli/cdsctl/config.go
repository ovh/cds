package main

import (
	"fmt"
	"os"
	"path"
	"strconv"

	repo "github.com/fsamin/go-repo"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

var (
	_ProjectKey      = "project-key"
	_ApplicationName = "application-name"
	_WorkflowName    = "workflow-name"
)

type config struct {
	Host                  string
	User                  string
	Token                 string
	InsecureSkipVerifyTLS bool
}

func userHomeDir() string {
	env := "HOME"
	if sdk.GOOS == "windows" {
		env = "USERPROFILE"
	} else if sdk.GOOS == "plan9" {
		env = "home"
	}
	return os.Getenv(env)
}

func loadConfig(configFile string) (*cdsclient.Config, error) {
	var verbose = os.Getenv("CDS_VERBOSE") == "true"

	c := &config{}
	c.Host = os.Getenv("CDS_API_URL")
	c.User = os.Getenv("CDS_USER")
	c.Token = os.Getenv("CDS_TOKEN")
	c.InsecureSkipVerifyTLS, _ = strconv.ParseBool(os.Getenv("CDS_INSECURE"))
	if insecureSkipVerifyTLS { // if set from command line
		c.InsecureSkipVerifyTLS = true
	}

	if c.Host != "" && c.User != "" {
		if verbose {
			fmt.Println("Configuration loaded from environment variables")
		}
	}

	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	homedir := userHomeDir()

	var configFiles []string
	if configFile != "" {
		configFiles = []string{configFile}
	} else {
		configFiles = []string{
			path.Join(dir, ".cdsrc"),
			path.Join(homedir, ".cdsrc"),
		}
	}

	var i int
	for c.Host == "" && i < len(configFiles) {
		if _, err := os.Stat(configFiles[i]); err == nil {
			f, err := os.Open(configFiles[i])
			if err != nil {
				if verbose {
					fmt.Printf("Unable to read %s \n", configFiles[i])
				}
				return nil, err
			}
			defer f.Close()

			if err := loadSecret(f, c); err != nil {
				if verbose {
					fmt.Printf("Unable to load configuration %s \n", configFiles[i])
				}
				return nil, err
			}

			if verbose {
				fmt.Println("Configuration loaded from", configFiles[i])
			}
		}
		i++
	}

	if c.Host == "" {
		return nil, fmt.Errorf("unable to load configuration, you should try to login first")
	}

	conf := &cdsclient.Config{
		Host:                  c.Host,
		User:                  c.User,
		Token:                 c.Token,
		Verbose:               verbose,
		InsecureSkipVerifyTLS: c.InsecureSkipVerifyTLS,
	}

	return conf, nil
}

func withAllCommandModifiers() []cli.CommandModifier {
	return []cli.CommandModifier{cli.CommandWithExtraFlags, cli.CommandWithExtraAliases, withAutoConf()}
}

func withAutoConf() cli.CommandModifier {
	return cli.CommandWithPreRun(
		func(c *cli.Command, args *[]string) error {
			// if args length equals or over context args length means that all
			// context args were given so ignore discover conf
			if len(*args) >= len(c.Ctx) {
				return nil
			}

			return discoverConf(c.Ctx, args)
		},
	)
}

func discoverConf(ctx []cli.Arg, args *[]string) error {
	var needProject, needApplication, needWorkflow bool
	var projectKey, applicationName, workflowName string

	// populates needs an init values from args and ctx
	mctx := make(map[string]cli.Arg, len(ctx))
	for i, arg := range ctx {
		mctx[arg.Name] = arg
		switch arg.Name {
		case _ProjectKey:
			needProject = true
			if i < len(*args) {
				projectKey = (*args)[i]
			}
		case _ApplicationName:
			needApplication = true
			if i < len(*args) {
				applicationName = (*args)[i]
			}
		case _WorkflowName:
			needWorkflow = true
			if i < len(*args) {
				workflowName = (*args)[i]
			}
		}
	}

	// updates needs from values found in args
	needProject = needProject && projectKey == ""
	needApplication = needApplication && applicationName == ""
	needWorkflow = needWorkflow && workflowName == ""

	if !(needProject || needApplication || needWorkflow) {
		return nil
	}

	// try to find existing .git repository
	var repoExists bool
	r, err := repo.New(".")
	if err == nil {
		repoExists = true
	}

	// if repo exists ask for usage
	if repoExists {
		gitProjectKey, _ := r.LocalConfigGet("cds", "project")
		gitApplicationName, _ := r.LocalConfigGet("cds", "application")
		gitWorkflowName, _ := r.LocalConfigGet("cds", "workflow")

		// if all needs were found in git do not ask for confirmation and use the config
		needConfirmation := (needProject && gitProjectKey == "") || (needApplication && gitApplicationName == "") || (needWorkflow && gitWorkflowName == "")

		if needConfirmation {
			fetchURL, err := r.FetchURL()
			if err != nil {
				return err
			}
			name, err := r.Name()
			if err != nil {
				return err
			}
			repoExists = cli.AskForConfirmation(fmt.Sprintf("Detected repository as %s (%s). Is it correct?", name, fetchURL))
		}
	}

	// if repo exists and is correct get existing config from it's config
	if repoExists {
		if needProject {
			projectKey, _ = r.LocalConfigGet("cds", "project")
		}
		if needApplication {
			applicationName, _ = r.LocalConfigGet("cds", "application")
		}
		if needWorkflow {
			workflowName, _ = r.LocalConfigGet("cds", "workflow")
		}
	}

	// updates needs from values found in git config
	needProject = needProject && projectKey == ""
	needApplication = needApplication && applicationName == ""
	needWorkflow = needWorkflow && workflowName == ""

	// populate project, application and workflow if required
	if needProject || needApplication || needWorkflow {
		var projects []sdk.Project

		if repoExists {
			name, err := r.Name()
			if err != nil {
				return err
			}
			ps, err := client.ProjectList(true, true, cdsclient.Filter{Name: "repo", Value: name})
			if err != nil {
				return err
			}

			// if there is multiple projects with current repo or zero, ask with the entire list of projects
			// else suggest the repo found
			if len(projects) == 1 {
				projects = ps
			}
		}

		if projects == nil {
			ps, err := client.ProjectList(true, true)
			if err != nil {
				return err
			}
			projects = ps
		}

		var project *sdk.Project

		// try to use the given project key
		if projectKey != "" {
			for _, p := range projects {
				if p.Key == projectKey {
					project = &p
				}
			}
		}

		// if given project key not valid ask for a project
		if project == nil {
			if len(projects) == 1 {
				if !cli.AskForConfirmation(fmt.Sprintf("Found one CDS project %s - %s. Is it correct?", projects[0].Key, projects[0].Name)) {
					// there is no filter on repo so there was only one choice possible
					if !repoExists {
						return fmt.Errorf("Can't find a project to use")
					}
				} else {
					project = &projects[0]
				}
			}
			if project == nil {
				// if the project found for current repo was not selected load all projects list
				if repoExists && len(projects) == 1 {
					ps, err := client.ProjectList(true, true)
					if err != nil {
						return err
					}
					projects = ps
				}

				opts := make([]string, len(projects))
				for i := range projects {
					opts[i] = fmt.Sprintf("%s - %s", projects[i].Key, projects[i].Name)
				}
				selected := cli.MultiChoice("Choose the CDS project:", opts...)
				project = &projects[selected]
			}
		}

		// set project key and override repository config if exists
		projectKey = project.Key
		if repoExists {
			if err := r.LocalConfigSet("cds", "project", projectKey); err != nil {
				return err
			}
		}

		if needApplication {
			var application *sdk.Application
			if len(project.Applications) == 1 {
				if cli.AskForConfirmation(fmt.Sprintf("Found one CDS application %s. Is it correct?", project.Applications[0].Name)) {
					application = &project.Applications[0]
				}
			} else if len(project.Applications) > 1 {
				opts := make([]string, len(project.Applications))
				for i := 0; i < len(project.Applications); i++ {
					opts[i] = project.Applications[i].Name
				}
				selected := cli.MultiChoice("Choose the CDS application:", opts...)
				application = &project.Applications[selected]
			}
			if application == nil && !mctx[_ApplicationName].AllowEmpty {
				return fmt.Errorf("Can't find an application to use")
			}

			// set application name and override repository config if exists
			applicationName = application.Name
			if application != nil {
				if repoExists {
					if err := r.LocalConfigSet("cds", "application", applicationName); err != nil {
						return err
					}
				}
			}
		}

		if needWorkflow {
			var workflow *sdk.Workflow
			if len(project.Workflows) == 1 {
				if cli.AskForConfirmation(fmt.Sprintf("Found one CDS workflow %s. Is it correct?", project.Workflows[0].Name)) {
					workflow = &project.Workflows[0]
				}
			} else if len(project.Workflows) > 1 {
				opts := make([]string, len(project.Workflows))
				for i := 0; i < len(project.Workflows); i++ {
					opts[i] = project.Workflows[i].Name
				}
				selected := cli.MultiChoice("Choose the CDS workflow:", opts...)
				workflow = &project.Workflows[selected]
			}
			if workflow == nil && !mctx[_WorkflowName].AllowEmpty {
				return fmt.Errorf("Can't find a workflow to use")
			}

			// set workflow name and override repository config if exists
			if workflow != nil {
				workflowName = workflow.Name
				if repoExists {
					if err := r.LocalConfigSet("cds", "workflow", workflowName); err != nil {
						return err
					}
				}
			}
		}
	}

	// for all required context args override or add the value in cli args
	for i, arg := range ctx {
		if i >= len(*args) {
			(*args) = append((*args), "")
		}
		switch arg.Name {
		case _ProjectKey:
			(*args)[i] = projectKey
		case _ApplicationName:
			(*args)[i] = applicationName
		case _WorkflowName:
			(*args)[i] = workflowName
		}
	}

	return nil
}
