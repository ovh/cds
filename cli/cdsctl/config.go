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

			if _, err := repo.New("."); err != nil {
				//Ignore error
				return nil
			}

			if err := discoverConf(); err != nil {
				return err
			}

			preargs := []string{}
			for _, arg := range c.Ctx {
				if arg.Name == _ProjectKey {
					preargs = []string{autoDiscoveredProj}
				}
				if arg.Name == _ApplicationName {
					preargs = append(preargs, autoDiscoveredApp)
				}
				if arg.Name == _WorkflowName {
					preargs = append(preargs, autoDiscoveredWorkflow)
				}
			}

			*args = append(preargs, *args...)

			return nil
		},
	)
}

var (
	autoDiscoveredProj     string
	autoDiscoveredApp      string
	autoDiscoveredWorkflow string
)

func discoverConf() error {
	r, err := repo.New(".")
	if err != nil {
		return err
	}

	if proj, _ := r.LocalConfigGet("cds", "project"); proj != "" {
		//It's already configured
		autoDiscoveredProj = proj
		autoDiscoveredApp, _ = r.LocalConfigGet("cds", "application")
		autoDiscoveredWorkflow, _ = r.LocalConfigGet("cds", "workflow")
		return nil
	}

	fetchURL, err := r.FetchURL()
	if err != nil {
		return err
	}

	name, err := r.Name()
	if err != nil {
		return err
	}

	if cli.AskForConfirmation(fmt.Sprintf("Detected repository as %s (%s). Is it correct?", name, fetchURL)) {
		projects, err := client.ProjectList(true, true, cdsclient.Filter{
			Name:  "repo",
			Value: name,
		})
		if err != nil {
			return err
		}

		// set cds.project key in git config and context
		var project *sdk.Project
		if len(projects) == 1 {
			if cli.AskForConfirmation(fmt.Sprintf("Found CDS project %s - %s. Is it correct?", projects[0].Key, projects[0].Name)) {
				project = &projects[0]
			}
		} else {
			opts := make([]string, len(projects))
			for i := range projects {
				opts[i] = fmt.Sprintf("%s - %s", projects[i].Key, projects[i].Name)
			}
			selected := cli.MultiChoice("Choose the CDS project", opts...)
			project = &projects[selected]
		}
		if project == nil {
			return nil
		}
		if err := r.LocalConfigSet("cds", "project", project.Key); err != nil {
			return err
		}
		autoDiscoveredProj = project.Name

		// set cds.application name in git config and context
		var application *sdk.Application
		if len(project.Applications) == 1 {
			if cli.AskForConfirmation(fmt.Sprintf("Found CDS application %s. Is it correct?", project.Applications[0].Name)) {
				application = &project.Applications[0]
			}
		} else {
			opts := make([]string, len(project.Applications))
			for i := 0; i < len(project.Applications); i++ {
				opts[i] = project.Applications[i].Name
			}
			selected := cli.MultiChoice("Choose the CDS application", opts...)
			application = &project.Applications[selected]
		}
		if application != nil {
			if err := r.LocalConfigSet("cds", "application", application.Name); err != nil {
				return err
			}
			autoDiscoveredApp = application.Name
		}

		// set cds.workflow name in git config and context
		var workflow *sdk.Workflow
		if len(project.Workflows) == 1 {
			if cli.AskForConfirmation(fmt.Sprintf("Found CDS workflow %s. Is it correct?", project.Workflows[0].Name)) {
				workflow = &project.Workflows[0]
			}
		} else {
			opts := make([]string, len(project.Workflows))
			for i := 0; i < len(project.Workflows); i++ {
				opts[i] = project.Workflows[i].Name
			}
			selected := cli.MultiChoice("Choose the CDS workflow", opts...)
			workflow = &project.Workflows[selected]
		}
		if workflow != nil {
			if err := r.LocalConfigSet("cds", "workflow", workflow.Name); err != nil {
				return err
			}
			autoDiscoveredWorkflow = workflow.Name
		}
	}

	return nil
}
