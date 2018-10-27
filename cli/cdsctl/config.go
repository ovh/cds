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
			if len(*args) >= len(c.Ctx)+len(c.Args) {
				return nil
			}

			preargs, err := discoverConf(c.Ctx)
			if err != nil {
				return err
			}

			*args = append(preargs, *args...)

			return nil
		},
	)
}

func discoverConf(args []cli.Arg) ([]string, error) {
	var needProject, needApplication, needWorkflow bool
	for _, arg := range args {
		switch arg.Name {
		case _ProjectKey:
			needProject = true
		case _ApplicationName:
			needApplication = true
		case _WorkflowName:
			needWorkflow = true
		}
	}

	if !(needProject || needApplication || needWorkflow) {
		return nil, nil
	}

	var projectKey, applicationName, workflowName string

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
		needConfirmation := !(needProject != (gitProjectKey != "") || needApplication != (gitApplicationName != "") || needWorkflow == (gitWorkflowName != ""))

		if needConfirmation {
			fetchURL, err := r.FetchURL()
			if err != nil {
				return nil, err
			}
			name, err := r.Name()
			if err != nil {
				return nil, err
			}
			repoExists = cli.AskForConfirmation(fmt.Sprintf("Detected repository as %s (%s). Is it correct?", name, fetchURL))
		}
	}

	// if repo exists and is correct get existing config from it's config
	if repoExists {
		projectKey, _ = r.LocalConfigGet("cds", "project")
		applicationName, _ = r.LocalConfigGet("cds", "application")
		workflowName, _ = r.LocalConfigGet("cds", "workflow")
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
				return nil, err
			}
			projects, err = client.ProjectList(true, true, cdsclient.Filter{Name: "repo", Value: name})
		} else {
			projects, err = client.ProjectList(true, true)
		}
		if err != nil {
			return nil, err
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
					return nil, fmt.Errorf("Can't find a project to use")
				}
				project = &projects[0]
			} else {
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
				return nil, err
			}
		}

		if needApplication {
			var application *sdk.Application
			if len(project.Applications) == 1 {
				if !cli.AskForConfirmation(fmt.Sprintf("Found one CDS application %s. Is it correct?", project.Applications[0].Name)) {
					return nil, fmt.Errorf("Can't find an application to use")
				}
				application = &project.Applications[0]
			} else {
				opts := make([]string, len(project.Applications))
				for i := 0; i < len(project.Applications); i++ {
					opts[i] = project.Applications[i].Name
				}
				selected := cli.MultiChoice("Choose the CDS application:", opts...)
				application = &project.Applications[selected]
			}

			// set application name and override repository config if exists
			applicationName = application.Name
			if repoExists {
				if err := r.LocalConfigSet("cds", "application", applicationName); err != nil {
					return nil, err
				}
			}
		}

		if needWorkflow {
			var workflow *sdk.Workflow
			if len(project.Workflows) == 1 {
				if !cli.AskForConfirmation(fmt.Sprintf("Found one CDS workflow %s. Is it correct?", project.Workflows[0].Name)) {
					return nil, fmt.Errorf("Can't find a workflow to use")
				}
				workflow = &project.Workflows[0]
			} else {
				opts := make([]string, len(project.Workflows))
				for i := 0; i < len(project.Workflows); i++ {
					opts[i] = project.Workflows[i].Name
				}
				selected := cli.MultiChoice("Choose the CDS workflow:", opts...)
				workflow = &project.Workflows[selected]
			}

			// set workflow name and override repository config if exists
			workflowName = workflow.Name
			if repoExists {
				if err := r.LocalConfigSet("cds", "workflow", workflowName); err != nil {
					return nil, err
				}
			}
		}
	}

	var values []string

	// set then returns values in right order
	for _, arg := range args {
		switch arg.Name {
		case _ProjectKey:
			values = append(values, projectKey)
		case _ApplicationName:
			values = append(values, applicationName)
		case _WorkflowName:
			values = append(values, workflowName)
		}
	}

	return values, nil
}
