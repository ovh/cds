package main

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strconv"

	"github.com/ovh/cds/sdk"

	repo "github.com/fsamin/go-repo"
	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/cdsclient"
)

type config struct {
	Host                  string
	User                  string
	Token                 string
	InsecureSkipVerifyTLS bool
}

func userHomeDir() string {
	env := "HOME"
	if runtime.GOOS == "windows" {
		env = "USERPROFILE"
	} else if runtime.GOOS == "plan9" {
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
		Host:    c.Host,
		User:    c.User,
		Token:   c.Token,
		Verbose: verbose,
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

			for _, arg := range c.Ctx {
				if arg.Name == "project-key" {
					*args = []string{autoDiscoveredProj}
				}
				if arg.Name == "application-name" {
					*args = append(*args, autoDiscoveredApp)
				}
				if arg.Name == "workflow-name" {
					*args = append(*args, autoDiscoveredWorkflow)
				}
			}

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

	if cli.AskForConfirmation(fmt.Sprintf("Detected repository as %s (%s). Is it correct ?", name, fetchURL)) {
		projs, err := client.ProjectList(true, true, cdsclient.Filter{
			Name:  "repo",
			Value: name,
		})
		if err != nil {
			return err
		}

		var chosenProj *sdk.Project

		filteredProjs := []sdk.Project{}
		for i, p := range projs {
			if len(p.Applications) > 0 || len(p.Workflows) > 0 {
				filteredProjs = append(filteredProjs, projs[i])
			}
		}
		projs = nil

		//Set cds.project
		if len(filteredProjs) == 1 {
			if cli.AskForConfirmation(fmt.Sprintf("Found CDS project %s - %s. Is it correct ?", filteredProjs[0].Key, filteredProjs[0].Name)) {
				chosenProj = &filteredProjs[0]
			}
		} else {
			opts := make([]string, len(filteredProjs))
			for i := range filteredProjs {
				opts[i] = fmt.Sprintf("%s - %s", filteredProjs[i].Key, filteredProjs[i].Name)
			}
			choice := cli.MultiChoice("Choose the CDS project", opts...)

			for i := range filteredProjs {
				if choice == fmt.Sprintf("%s - %s", filteredProjs[i].Key, filteredProjs[i].Name) {
					chosenProj = &filteredProjs[i]
				}
			}

			if err := r.LocalConfigSet("cds", "project", chosenProj.Key); err != nil {
				return err
			}
		}

		//Set cds.application
		if len(chosenProj.Applications) == 1 {
			if cli.AskForConfirmation(fmt.Sprintf("Found CDS application %s. Is it correct ?", chosenProj.Applications[0].Name)) {
				if err := r.LocalConfigSet("cds", "application", chosenProj.Applications[0].Name); err != nil {
					return err
				}
			}
		} else {
			opts := make([]string, len(chosenProj.Applications))
			for i := range chosenProj.Applications {
				opts[i] = chosenProj.Applications[i].Name
			}
			choice := cli.MultiChoice("Choose the CDS application", opts...)

			for i := range chosenProj.Applications {
				if choice == chosenProj.Applications[i].Name {
					if err := r.LocalConfigSet("cds", "application", chosenProj.Applications[i].Name); err != nil {
						return err
					}
					break
				}
			}
		}

		//Set cds.workflow
		if len(chosenProj.Workflows) == 1 {
			if cli.AskForConfirmation(fmt.Sprintf("Found CDS workflow %s. Is it correct ?", chosenProj.Workflows[0].Name)) {
				if err := r.LocalConfigSet("cds", "workflow", chosenProj.Workflows[0].Name); err != nil {
					return err
				}
			}
		} else {
			opts := make([]string, len(chosenProj.Workflows))
			for i := range chosenProj.Workflows {
				opts[i] = chosenProj.Workflows[i].Name
			}
			choice := cli.MultiChoice("Choose the CDS workflow", opts...)

			for i := range chosenProj.Workflows {
				if choice == chosenProj.Workflows[i].Name {
					return r.LocalConfigSet("cds", "workflow", chosenProj.Workflows[i].Name)
				}
			}
		}
	}
	return nil
}
