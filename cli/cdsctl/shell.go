package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var shellCmd = cli.Command{
	Name:  "shell",
	Short: "cdsctl interactive shell",
	Long: `
CDS Shell Mode. Keywords:

- cd: reset current object. running "ls" after "cd" will display Projects List
- cd <KEY>: go to an object, try to run "ls" after a cd <KEY>
- help: display this help
- ls: display current list
- ls <KEY>: display current object, ls MY_PRJ is the same as cdsctl project show MY_PRJ
- open: open CDS WebUI with current context
- run: run current workflow
- version: same as cdsctl version command

`,
}

func shellRun(v cli.Values) error {
	shellASCII()
	version, err := client.Version()
	if err != nil {
		return err
	}
	fmt.Printf("Connected. cdsctl version: %s connected to CDS API %s version:%s \n\n", sdk.VERSION, client.APIURL(), version.Version)
	fmt.Println("enter `exit` quit")

	// enable shell mode, this will prevent to os.Exit if there is an error on a command
	cli.ShellMode = true

	current := &shellCurrent{}
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("> ")
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		if text == "exit" || text == "quit" {
			break
		}
		if len(text) > 0 {
			shellProcessCommand(text, current)
		}
	}
	return nil
}

type shellCurrent struct {
	project     string
	workflow    string
	application string
	pipeline    string
	environment string
	position    shellPosition
}

type shellPosition int

const (
	shellInProjects shellPosition = iota
	shellInProject
	shellInWorkflow
	shellInWorkflows
	shellInApplication
	shellInApplications
	shellInPipeline
	shellInPipelines
	shellInEnvironment
	shellInEnvironments
)

// isInProjects returns true if there is nothing selected
func (s *shellCurrent) reset() {
	s.position = shellInProjects
	s.project = ""
	s.workflow = ""
	s.application = ""
	s.pipeline = ""
	s.environment = ""
}

func (s *shellCurrent) openBrowser() {
	var baseURL string
	configUser, err := client.ConfigUser()
	if err != nil {
		fmt.Printf("Error while getting URL UI: %s", err)
		return
	}

	if b, ok := configUser[sdk.ConfigURLUIKey]; ok {
		baseURL = b
	}

	if baseURL == "" {
		fmt.Println("Unable to retrieve workflow URI")
		return
	}

	url := fmt.Sprintf("%s", baseURL)
	switch s.position {
	case shellInProjects:
		// nothing
	case shellInProject:
		url += fmt.Sprintf("/project/%s?tab=workflows", s.project)
	case shellInWorkflow:
		url += fmt.Sprintf("/project/%s/workflow/%s", s.project, s.workflow)
	case shellInWorkflows:
		url += fmt.Sprintf("/project/%s?tab=workflows", s.project)
	case shellInApplication:
		url += fmt.Sprintf("/project/%s/application/%s", s.project, s.application)
	case shellInApplications:
		url += fmt.Sprintf("/project/%s?tab=workflows", s.project)
	case shellInPipeline:
		url += fmt.Sprintf("/project/%s/pipeline/%s", s.project, s.pipeline)
	case shellInPipelines:
		url += fmt.Sprintf("/project/%s?tab=pipelines", s.project)
	case shellInEnvironment:
		url += fmt.Sprintf("/project/%s&envName=%s", s.project, s.environment)
	case shellInEnvironments:
		url += fmt.Sprintf("/project/%s?tab=environments", s.project)
	}
	fmt.Printf("Opening %s...\n", url)
	browser.OpenURL(url)
}

func (s *shellCurrent) getArgs() []string {
	r := []string{}
	if s.project != "" {
		r = append(r, s.project)
	}
	if s.workflow != "" {
		r = append(r, s.workflow)
	}
	if s.application != "" {
		r = append(r, s.application)
	}
	if s.pipeline != "" {
		r = append(r, s.pipeline)
	}
	return r
}

func (s *shellCurrent) getPwd() string {
	r := "/ "
	if s.project != "" {
		r += s.project
	}
	if s.position == shellInWorkflows || s.position == shellInWorkflow {
		r += " / workflows"
	}
	if s.workflow != "" {
		r += " / " + s.workflow
	}
	if s.position == shellInApplications || s.position == shellInApplication {
		r += " / applications"
	}
	if s.application != "" {
		r += " / " + s.application
	}
	if s.position == shellInPipelines || s.position == shellInPipeline {
		r += " / pipelines"
	}
	if s.pipeline != "" {
		r += " / " + s.pipeline
	}
	if s.position == shellInEnvironments || s.position == shellInEnvironment {
		r += " / environments"
	}
	if s.environment != "" {
		r += " / " + s.environment
	}
	return r
}

func (s *shellCurrent) setPositionInsideProject(input string) {
	s.workflow = ""
	s.application = ""
	s.pipeline = ""
	if input == "workflows" {
		s.position = shellInWorkflows
	} else if input == "applications" {
		s.position = shellInApplications
	} else if input == "environments" {
		s.position = shellInEnvironments
	} else if input == "pipelines" {
		s.position = shellInPipelines
	} else {
		fmt.Printf("Invalid argument. Must be workflows, applications, environments or pipelines")
	}
}

type shellCommandFunc func(args []string, current *shellCurrent) *cobra.Command

var (
	shellCommands = map[string]shellCommandFunc{
		"cd":  cdCommand,
		"ls":  lsCommand,
		"dir": lsCommand,
		"help": func(args []string, current *shellCurrent) *cobra.Command {
			fmt.Println(shellCmd.Long)
			return nil
		},
		"open": func(args []string, current *shellCurrent) *cobra.Command {
			current.openBrowser()
			return nil
		},
		"pwd": func(args []string, current *shellCurrent) *cobra.Command {
			fmt.Println(current.getPwd())
			return nil
		},
		"version": func(args []string, current *shellCurrent) *cobra.Command {
			return cli.NewCommand(versionCmd, versionRun, nil, cli.CommandWithoutExtraFlags)
		},
		"run": func(args []string, current *shellCurrent) *cobra.Command {
			return cli.NewCommand(workflowRunManualCmd, workflowRunManualRun, nil, withAllCommandModifiers()...)
		},
	}

	cdCommand = func(args []string, current *shellCurrent) *cobra.Command {
		if len(args) == 0 {
			current.reset()
			return nil
		}

		// remove ./, useful if user enter "cd ./workflows"
		args[0] = strings.TrimPrefix(args[0], "./")

		// cd ..
		if len(args) == 1 && args[0] == ".." {
			if current.position == shellInProject { // inside a project, go to list project
				current.reset()
				current.position = shellInProjects
			} else if current.position != shellInProjects { // inside apps, workflows... go to project
				prj := current.project
				current.reset()
				current.project = prj
				current.position = shellInProject
			}
			return nil
		}

		switch current.position {
		case shellInProjects:
			current.reset()
			current.project = args[0]
			current.position = shellInProject
		case shellInProject:
			current.setPositionInsideProject(args[0])
		case shellInWorkflows:
			current.position = shellInWorkflow
			current.workflow = args[0]
		case shellInApplications:
			current.position = shellInApplication
			current.application = args[0]
		case shellInEnvironments:
			current.position = shellInEnvironment
			current.environment = args[0]
		case shellInPipelines:
			current.position = shellInPipeline
			current.pipeline = args[0]
		}
		return nil
	}

	lsCommand = func(args []string, current *shellCurrent) *cobra.Command {
		if current.position == shellInProjects {
			if len(args) == 1 {
				return cli.NewGetCommand(projectShowCmd, projectShowRun, nil, withAllCommandModifiers()...)
			}
			return cli.NewListCommand(projectListCmd, projectListRun, nil, withAllCommandModifiers()...)
		} else if current.position == shellInProject {
			fmt.Println("./workflows\n./applications\n./pipelines\n./environments")
			return cli.NewGetCommand(projectShowCmd, projectShowRun, nil, withAllCommandModifiers()...)
		} else if current.position == shellInApplications {
			return cli.NewListCommand(applicationListCmd, applicationListRun, nil, withAllCommandModifiers()...)
		} else if current.position == shellInApplication {
			return cli.NewGetCommand(applicationShowCmd, applicationShowRun, nil, withAllCommandModifiers()...)
		} else if current.position == shellInEnvironments {
			return cli.NewListCommand(environmentListCmd, environmentListRun, nil, withAllCommandModifiers()...)
		} else if current.position == shellInPipelines {
			return cli.NewListCommand(pipelineListCmd, pipelineListRun, nil, withAllCommandModifiers()...)
		} else if current.position == shellInWorkflows {
			return cli.NewListCommand(workflowListCmd, workflowListRun, nil, withAllCommandModifiers()...)
		} else if current.position == shellInWorkflow {
			return cli.NewGetCommand(workflowShowCmd, workflowShowRun, nil, withAllCommandModifiers()...)
		}
		return nil
	}
)

func shellProcessCommand(input string, current *shellCurrent) {
	tuple := strings.Split(input, " ")
	if f, ok := shellCommands[tuple[0]]; ok {
		if f == nil {
			fmt.Printf("Command %s not defined in this context\n", input)
			return
		}
		cmd := f(tuple[1:], current)
		if cmd == nil {
			return
		}
		fmt.Printf("--> Command %s", cmd.Short)
		args := append(current.getArgs(), tuple[1:]...)
		if len(args) > 0 {
			fmt.Printf(" with args: %+v", args)
		}
		fmt.Println()
		cmd.SetArgs(args)
		cmd.Execute()
		if input != "pwd" {
			fmt.Println(current.getPwd())
		}
		return
	}
	fmt.Printf("Invalid command %s\n", input)
}

func shellASCII() {
	fmt.Printf(`

                                           
  ,ad8888ba,   88888888ba,     ad88888ba   
 d8"'    ` + "`" + `"8b  88      ` + "`" + `"8b   d8"     "8b  
d8'            88        ` + "`" + `8b  Y8,          
88             88         88  ` + "`" + `Y8aaaaa,    
88             88         88    ` + "`" + `"""""8b,  
Y8,            88         8P          ` + "`" + `8b  
 Y8a.    .a8P  88      .a8P   Y8a     a8P  
  ` + "`" + `"Y8888Y"'   88888888Y"'     "Y88888P"   
						
  
connecting to cds api...                                           
  > `)
}
