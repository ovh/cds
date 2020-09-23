package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"

	"github.com/fsamin/go-repo"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/cli/cdsctl/internal"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

var (
	_ProjectKey      = "project-key"
	_ApplicationName = "application-name"
	_WorkflowName    = "workflow-name"
)

func userHomeDir() string {
	env := "HOME"
	if sdk.GOOS == "windows" {
		env = "USERPROFILE"
	} else if sdk.GOOS == "plan9" {
		env = "home"
	}
	return os.Getenv(env)
}

func loadConfig(cmd *cobra.Command) (string, *cdsclient.Config, error) {
	var configFile, _ = cmd.Flags().GetString("file")
	if configFile == "" {
		configFile = os.Getenv("CDS_FILE")
	}
	var verbose, _ = cmd.Flags().GetBool("verbose")
	verbose = verbose || os.Getenv("CDS_VERBOSE") == "true"
	var insecureSkipVerifyTLS, _ = cmd.Flags().GetBool("insecure")
	insecureSkipVerifyTLS = insecureSkipVerifyTLS || os.Getenv("CDS_INSECURE") == "true"
	var contextName, _ = cmd.Flags().GetString("context")
	if contextName == "" {
		contextName = os.Getenv("CDS_CONTEXT")
	}

	cdsctxEnv := &internal.CDSContext{}
	cdsctxEnv.Host = os.Getenv("CDS_API_URL")
	cdsctxEnv.Session = os.Getenv("CDS_SESSION_TOKEN")
	cdsctxEnv.Token = os.Getenv("CDS_TOKEN")
	cdsctxEnv.InsecureSkipVerifyTLS = insecureSkipVerifyTLS

	if cdsctxEnv.Host != "" {
		if verbose {
			fmt.Println("Configuration loaded from environment variables")
		}
	}

	if configFile == "" {
		configFile = path.Join(userHomeDir(), ".cdsrc")
	}

	if verbose {
		fmt.Println("Configuration loaded from", configFile)
	}

	cdsctx := &internal.CDSContext{}
	if _, err := os.Stat(configFile); !os.IsNotExist(err) {
		f, err := os.Open(configFile)
		if err != nil {
			return "", nil, fmt.Errorf("unable to read file %s: %v", configFile, err)
		}
		defer f.Close()

		if contextName != "" {
			if cdsctx, err = internal.GetContext(f, contextName); err != nil {
				return "", nil, fmt.Errorf("unable to load the current context from %s", contextName)
			}
		} else if cdsctx, err = internal.GetCurrentContext(f); err != nil {
			return "", nil, fmt.Errorf("unable to load the current context from %s", configFile)
		}

		if verbose {
			fmt.Printf("Configuration loaded from %s with context %s\n", configFile, contextName)
		}
	}

	if cdsctxEnv.Host != "" {
		cdsctx.Host = cdsctxEnv.Host
	}
	if cdsctxEnv.Session != "" {
		cdsctx.Session = cdsctxEnv.Session
	}
	if cdsctxEnv.Token != "" {
		cdsctx.Token = cdsctxEnv.Token
	}

	// if there is no session token, but if we have a sign in token, recreate a session token
	if cdsctxEnv.Session == "" && cdsctxEnv.Token != "" {
		var err error
		cdsctx, err = recreateSessionToken(configFile, *cdsctx, contextName)
		if err != nil {
			return "", nil, err
		}
	}

	if cdsctxEnv.InsecureSkipVerifyTLS {
		cdsctx.InsecureSkipVerifyTLS = cdsctxEnv.InsecureSkipVerifyTLS
	}

	cdsctx.Verbose = verbose

	if cdsctx.Host == "" {
		return "", nil, fmt.Errorf("invalid cdsctl configuration to reach a CDS API")
	}

	config := &cdsclient.Config{
		Host:                              cdsctx.Host,
		SessionToken:                      cdsctx.Session,
		BuitinConsumerAuthenticationToken: cdsctx.Token,
		Verbose:                           verbose,
		InsecureSkipVerifyTLS:             insecureSkipVerifyTLS,
	}

	return configFile, config, nil
}

// regenerate a session token from a singin token if needed
func recreateSessionToken(configFile string, cdsctx internal.CDSContext, contextName string) (*internal.CDSContext, error) {
	req := sdk.AuthConsumerSigninRequest{
		"token": cdsctx.Token,
	}
	client := cdsclient.New(cdsclient.Config{
		Host:                  cdsctx.Host,
		Verbose:               os.Getenv("CDS_VERBOSE") == "true" || cdsctx.Verbose,
		InsecureSkipVerifyTLS: os.Getenv("CDS_INSECURE") == "true" || cdsctx.InsecureSkipVerifyTLS,
	})
	res, err := client.AuthConsumerSignin(sdk.ConsumerBuiltin, req)
	if err != nil {
		return nil, fmt.Errorf("cannot signin: %v", err)
	}
	if res.Token == "" || res.User == nil {
		return nil, fmt.Errorf("invalid username or token returned by sign in token")
	}
	cdsctx.Session = res.Token
	// resave session token
	if res.Token == cdsctx.Session {
		return &cdsctx, nil
	}

	fi, err := os.OpenFile(configFile, os.O_RDONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("Error while opening file %s: %v", configFile, err)
	}
	wdata := &bytes.Buffer{}
	if err := internal.StoreContext(fi, wdata, cdsctx); err != nil {
		fi.Close() // nolint
		return nil, err
	}

	if err := fi.Close(); err != nil {
		return nil, fmt.Errorf("Error while closing file %s: %v", configFile, err)
	}
	if err := writeConfigFile(configFile, wdata); err != nil {
		return nil, err
	}

	return &cdsctx, nil
}

func withAllCommandModifiers() []cli.CommandModifier {
	return []cli.CommandModifier{cli.CommandWithExtraFlags, cli.CommandWithExtraAliases, withAutoConf()}
}

func withAutoConf() cli.CommandModifier {
	return cli.CommandWithPreRun(
		func(c *cli.Command, args *[]string) error {
			// if args length equals or over context args length means that all
			// context args were given so ignore discover conf
			if len(*args) >= len(c.Ctx)+len(c.Args) {
				return nil
			}

			preargs, err := discoverConf(c.Ctx)
			if err != nil {
				return err
			}

			(*args) = append(preargs, *args...)

			return nil
		},
	)
}

func discoverConf(ctxArg []cli.Arg) ([]string, error) {
	var needProject, needApplication, needWorkflow bool

	// populates needs from ctxArg
	mctx := make(map[string]cli.Arg, len(ctxArg))
	for _, arg := range ctxArg {
		mctx[arg.Name] = arg
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
	ctx := context.Background()

	// try to find existing .git repository
	var repoExists bool
	r, err := repo.New(ctx, ".")
	if err == nil {
		repoExists = true
	}

	// if repo exists ask for usage
	if repoExists {
		gitProjectKey, _ := r.LocalConfigGet(ctx, "cds", "project")
		gitApplicationName, _ := r.LocalConfigGet(ctx, "cds", "application")
		gitWorkflowName, _ := r.LocalConfigGet(ctx, "cds", "workflow")

		// if all needs were found in git do not ask for confirmation and use the config
		needConfirmation := (needProject && gitProjectKey == "") || (needApplication && gitApplicationName == "") || (needWorkflow && gitWorkflowName == "")

		if needConfirmation {
			fetchURL, err := r.FetchURL(ctx)
			if err != nil {
				return nil, errors.Wrap(err, "cannot get url from local git repository")
			}
			name, err := r.Name(ctx)
			if err != nil {
				return nil, errors.Wrap(err, "cannot get name from local git repository")
			}
			repoExists = cli.AskConfirm(fmt.Sprintf("Detected repository as %s (%s). Is it correct?", name, fetchURL))
		}
	}

	// if repo exists and is correct get existing config from it's config
	if repoExists {
		if needProject {
			projectKey, _ = r.LocalConfigGet(ctx, "cds", "project")
		}
		if needApplication {
			applicationName, _ = r.LocalConfigGet(ctx, "cds", "application")
		}
		if needWorkflow {
			workflowName, _ = r.LocalConfigGet(ctx, "cds", "workflow")
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
			name, err := r.Name(ctx)
			if err != nil {
				return nil, errors.Wrap(err, "cannot get name from current repository")
			}
			ps, err := client.ProjectList(true, true, cdsclient.Filter{Name: "repo", Value: name})
			if err != nil {
				return nil, err
			}

			// if there is multiple projects with current repo or zero, ask with the entire list of projects
			// else suggest the repo found
			if len(projects) == 1 {
				projects = ps
			}
		}

		if projects == nil {
			ps, err := client.ProjectList(false, false)
			if err != nil {
				return nil, err
			}
			projects = ps
		}

		var project *sdk.Project

		// try to use the given project key
		if projectKey != "" {
			for _, p := range projects {
				if p.Key == projectKey {
					project = &p
					break
				}
			}
		}

		// if given project key not valid ask for a project
		if project == nil {
			if len(projects) == 1 {
				if !cli.AskConfirm(fmt.Sprintf("Found one CDS project '%s - %s'. Is it correct?", projects[0].Key, projects[0].Name)) {
					// there is no filter on repo so there was only one choice possible
					if !repoExists {
						return nil, errors.New("can't find a project to use")
					}
				} else {
					project = &projects[0]
				}
			}
			if project == nil {
				// if the project found for current repo was not selected load all projects list
				if repoExists && len(projects) == 1 {
					ps, err := client.ProjectList(false, false)
					if err != nil {
						return nil, err
					}
					projects = ps
				}

				opts := make([]string, len(projects))
				for i := range projects {
					opts[i] = fmt.Sprintf("%s - %s", projects[i].Key, projects[i].Name)
				}
				selected := cli.AskChoice("Choose the CDS project", opts...)
				project = &projects[selected]
			}
		}

		// set project key and override repository config if exists
		projectKey = project.Key
		if repoExists {
			if err := r.LocalConfigSet(ctx, "cds", "project", projectKey); err != nil {
				return nil, errors.Wrap(err, "cannot set local git configuration")
			}
		}

		if needApplication {
			applications, err := client.ApplicationList(project.Key)
			if err != nil {
				return nil, err
			}

			var application *sdk.Application
			if len(applications) == 1 {
				if cli.AskConfirm(fmt.Sprintf("Found one CDS application '%s'. Is it correct?", applications[0].Name)) {
					application = &applications[0]
				}
			} else if len(applications) > 1 {
				opts := make([]string, len(applications))
				for i := range applications {
					opts[i] = applications[i].Name
				}
				if mctx[_ApplicationName].AllowEmpty {
					opts = append(opts, "Use a new application")
				}
				selected := cli.AskChoice("Choose the CDS application", opts...)
				if selected < len(applications) {
					application = &applications[selected]
				}
			}
			if application == nil && !mctx[_ApplicationName].AllowEmpty {
				return nil, errors.New("can't find an application to use")
			}

			// set application name and override repository config if exists
			if application != nil {
				applicationName = application.Name
				if repoExists {
					if err := r.LocalConfigSet(ctx, "cds", "application", applicationName); err != nil {
						return nil, errors.Wrap(err, "cannot set local git configuration")
					}
				}
			}
		}

		if needWorkflow {
			workflows, err := client.WorkflowList(project.Key)
			if err != nil {
				return nil, err
			}

			var workflow *sdk.Workflow
			if len(workflows) == 1 {
				if cli.AskConfirm(fmt.Sprintf("Found one CDS workflow '%s'. Is it correct?", workflows[0].Name)) {
					workflow = &workflows[0]
				}
			} else if len(workflows) > 1 {
				opts := make([]string, len(workflows))
				for i := range workflows {
					opts[i] = workflows[i].Name
				}
				if mctx[_WorkflowName].AllowEmpty {
					opts = append(opts, "Use a new workflow")
				}
				selected := cli.AskChoice("Choose the CDS workflow", opts...)
				if selected < len(workflows) {
					workflow = &workflows[selected]
				}
			}
			if workflow == nil && !mctx[_WorkflowName].AllowEmpty {
				return nil, errors.New("can't find a workflow to use")
			}

			// set workflow name and override repository config if exists
			if workflow != nil {
				workflowName = workflow.Name
				if repoExists {
					if err := r.LocalConfigSet(ctx, "cds", "workflow", workflowName); err != nil {
						return nil, errors.Wrap(err, "cannot set local git configuration")
					}
				}
			}
		}
	}

	// for all required context args override or add the value in cli args
	preargs := make([]string, len(ctxArg))
	for i, arg := range ctxArg {
		switch arg.Name {
		case _ProjectKey:
			preargs[i] = projectKey
		case _ApplicationName:
			preargs[i] = applicationName
		case _WorkflowName:
			preargs[i] = workflowName
		}
	}

	return preargs, nil
}
