package main

import (
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/chzyer/readline"
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
- mode: display current mode. Choose mode with "mode vi" ou "mode emacs"
- open: open CDS WebUI with current context
- run: run current workflow
- version: same as cdsctl version command

`,
}

var current *shellCurrent

func pcFromCommands(parent readline.PrefixCompleterInterface, c *cobra.Command) {
	pc := readline.PcItem(c.Use)
	parent.SetChildren(append(parent.GetChildren(), pc))
	for _, child := range c.Commands() {
		pcFromCommands(pc, child)
	}
}

func shellRun(v cli.Values) error {
	shellASCII()
	version, err := client.Version()
	if err != nil {
		return err
	}
	fmt.Printf("Connected. cdsctl version: %s connected to CDS API version:%s \n\n", sdk.VERSION, version.Version)
	fmt.Println("enter `exit` quit")

	// enable shell mode, this will prevent to os.Exit if there is an error on a command
	cli.ShellMode = true

	completer := readline.NewPrefixCompleter()
	for _, child := range root.Commands() {
		pcFromCommands(completer, child)
	}

	l, err := readline.NewEx(&readline.Config{
		Prompt:            "\033[31m»\033[0m ",
		HistoryFile:       path.Join(userHomeDir(), ".cdsctl_history"),
		AutoComplete:      completer,
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
	})

	if err != nil {
		fmt.Printf("Error: %s\n", err)
	}

	defer l.Close()

	current = &shellCurrent{rline: l, command: root}

	for {
		line, err := l.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}

		line = strings.TrimSpace(line)

		if line == "exit" || line == "quit" {
			break
		}
		// if len(line) > 0 {
		// 	shellProcessCommand(line, current)
		// }
		cmd, flags, err := root.Find(strings.Fields(line))
		if err != nil {
			fmt.Printf("Error: %s\n", []byte(err.Error()))
		}

		cmd.ParseFlags([]string{"--format", "json"})
		cmd.SetArgs([]string{})
		fmt.Printf("Flags: %+v\n", flags)
		fmt.Printf("Flags parsed: %+v\n", cmd.CommandPath())

		cmd.Run(cmd, []string{"00SIM"})

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
	rline       *readline.Instance
	command     *cobra.Command
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
	r := "/"
	if s.project != "" {
		r += s.project
	}
	if s.position == shellInWorkflows || s.position == shellInWorkflow {
		r += "/workflows"
	}
	if s.workflow != "" {
		r += "/" + s.workflow
	}
	if s.position == shellInApplications || s.position == shellInApplication {
		r += "/applications"
	}
	if s.application != "" {
		r += "/" + s.application
	}
	if s.position == shellInPipelines || s.position == shellInPipeline {
		r += "/pipelines"
	}
	if s.pipeline != "" {
		r += "/" + s.pipeline
	}
	if s.position == shellInEnvironments || s.position == shellInEnvironment {
		r += "/environments"
	}
	if s.environment != "" {
		r += "/" + s.environment
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

func listCurrent() func(string) []string {
	return func(line string) []string {
		r := []string{}
		switch current.position {
		case shellInProjects:
			values, _ := client.ProjectList(false, false)
			for _, v := range values {
				r = append(r, v.Key)
			}
		case shellInProject:
			r = []string{"..", "workflows", "applications", "environments", "pipelines"}
		case shellInWorkflows:
			values, _ := client.WorkflowList(current.project)
			for _, v := range values {
				r = append(r, v.Name)
			}
		case shellInApplications:
			values, _ := client.ApplicationList(current.project)
			for _, v := range values {
				r = append(r, v.Name)
			}
		case shellInPipelines:
			values, _ := client.PipelineList(current.project)
			for _, v := range values {
				r = append(r, v.Name)
			}
		case shellInEnvironments:
			values, _ := client.EnvironmentList(current.project)
			for _, v := range values {
				r = append(r, v.Name)
			}
		}
		return r
	}
}

var (
	// completer = readline.NewPrefixCompleter(
	// 	readline.PcItem("mode",
	// 		readline.PcItem("vi"),
	// 		readline.PcItem("emacs"),
	// 	),
	// 	readline.PcItem("help"),
	// 	readline.PcItem("cd",
	// 		readline.PcItemDynamic(listCurrent()),
	// 	),
	// 	readline.PcItem("ls",
	// 		readline.PcItemDynamic(listCurrent()),
	// 	),
	// 	readline.PcItem("open"),
	// 	readline.PcItem("pwd"),
	// 	readline.PcItem("version"),
	// 	readline.PcItem("run"),
	// 	readline.PcItem("exit"),
	// )

	shellCommands = map[string]shellCommandFunc{
		"cd": cdCommand,
		"ls": lsCommand,
		"mode": func(args []string, current *shellCurrent) *cobra.Command {
			if len(args) == 0 {
				if current.rline.IsVimMode() {
					println("current mode: vim")
				} else {
					println("current mode: emacs")
				}
			} else {
				switch args[0] {
				case "vi":
					current.rline.SetVimMode(true)
				case "emacs":
					current.rline.SetVimMode(false)
				default:
					fmt.Println("invalid mode:", args[0])
				}
			}
			return nil
		},
		"help": func(args []string, current *shellCurrent) *cobra.Command {
			fmt.Println(shellCmd.Long)
			return nil
		},
		"open": func(args []string, current *shellCurrent) *cobra.Command {
			current.openBrowser()
			return nil
		},
		"pwd": func(args []string, current *shellCurrent) *cobra.Command {
			fmt.Println(current.getPwd() + " // " + current.command.CommandPath())
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
			//	current.reset()
			return nil
		}

		cmds := current.command.Commands()
		for _, cmd := range cmds {

			if cmd.Name() == args[0] {
				current.command = cmd
			}
		}

		// cd ..
		// if len(args) == 1 && args[0] == ".." {
		// 	if current.position == shellInProject { // inside a project, go to list project
		// 		current.reset()
		// 		current.position = shellInProjects
		// 	} else if current.position != shellInProjects { // inside apps, workflows... go to project
		// 		prj := current.project
		// 		current.reset()
		// 		current.project = prj
		// 		current.position = shellInProject
		// 	}
		// 	return nil
		// }

		// switch current.position {
		// case shellInProjects:
		// 	current.reset()
		// 	current.project = args[0]
		// 	current.position = shellInProject
		// case shellInProject:
		// 	current.setPositionInsideProject(args[0])
		// case shellInWorkflows:
		// 	current.position = shellInWorkflow
		// 	current.workflow = args[0]
		// case shellInApplications:
		// 	current.position = shellInApplication
		// 	current.application = args[0]
		// case shellInEnvironments:
		// 	current.position = shellInEnvironment
		// 	current.environment = args[0]
		// case shellInPipelines:
		// 	current.position = shellInPipeline
		// 	current.pipeline = args[0]
		// }
		return nil
	}

	lsCommand = func(args []string, current *shellCurrent) *cobra.Command {
		cmds := current.command.Commands()
		for _, cmd := range cmds {
			if cmd.Name() == "list" {
				return cmd
			}
		}

		// switch current.position {
		// case shellInProjects:
		// 	if len(args) == 1 {
		// 		return cli.NewGetCommand(projectShowCmd, projectShowRun, nil, withAllCommandModifiers()...)
		// 	}
		// 	return cli.NewListCommand(projectListCmd, projectListRun, nil, withAllCommandModifiers()...)
		// case shellInProject:
		// 	fmt.Println("workflows\napplications\npipelines\nenvironments")
		// 	if len(args) == 1 {
		// 		switch args[0] {
		// 		case "workflows":
		// 			return cli.NewListCommand(workflowListCmd, workflowListRun, nil, withAllCommandModifiers()...)
		// 		case "pipelines":
		// 			return cli.NewListCommand(pipelineListCmd, pipelineListRun, nil, withAllCommandModifiers()...)
		// 		case "applications":
		// 			return cli.NewListCommand(applicationListCmd, applicationListRun, nil, withAllCommandModifiers()...)
		// 		case "environments":
		// 			return cli.NewListCommand(environmentListCmd, environmentListRun, nil, withAllCommandModifiers()...)
		// 		}
		// 		return nil
		// 	}
		// 	return cli.NewGetCommand(projectShowCmd, projectShowRun, nil, withAllCommandModifiers()...)
		// case shellInApplications:
		// 	return cli.NewListCommand(applicationListCmd, applicationListRun, nil, withAllCommandModifiers()...)
		// case shellInApplication:
		// 	return cli.NewGetCommand(applicationShowCmd, applicationShowRun, nil, withAllCommandModifiers()...)
		// case shellInEnvironments:
		// 	return cli.NewListCommand(environmentListCmd, environmentListRun, nil, withAllCommandModifiers()...)
		// case shellInPipelines:
		// 	return cli.NewListCommand(pipelineListCmd, pipelineListRun, nil, withAllCommandModifiers()...)
		// case shellInWorkflows:
		// 	return cli.NewListCommand(workflowListCmd, workflowListRun, nil, withAllCommandModifiers()...)
		// case shellInWorkflow:
		// 	return cli.NewGetCommand(workflowShowCmd, workflowShowRun, nil, withAllCommandModifiers()...)
		//}
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

		// if len(tuple) > 1 {
		// 	//subcmd := strings.Join(tuple[1:], " ")
		// 	//if subcmd != "workflows" && subcmd != "applications" && subcmd != "environments" && subcmd != "pipelines" {
		// 	//args = append(current.getArgs(), tuple[1:]...)
		// 	//}
		// }

		// //if len(args) > 0 {
		// fmt.Printf(" with args: %+v", args)
		// //}
		//fmt.Println()
		//t := strings.Split(cmd.CommandPath(), " ")
		//args := []string{}
		args := tuple[1:]
		fmt.Printf(" workflow.HasParent %t\n", workflow.HasParent())
		fmt.Printf(" exec %+v %+v\n", cmd, args)
		//cmd.Run(cmd, args)
		cmd.SetArgs(args)
		cmd.Execute()
		return
	}
	fmt.Printf("Invalid command %s\n", input)
}

func shellASCII() {
	fmt.Printf(`

               .';:looddddolc;'.               .,::::::::::::::::;;;,'..           .............................
            'cdOKKXXXXXXXXXXXXKOd:.            'OXXXXXXXXXXXXXXXXXXXKK0Oxo:...',;;::ccccccccccccccccccccccccccc;.
         .:x0XXXX0OxollllodxOKXXXXOl.          'OXXXX0OOOOOOOOOO0000KXXXXX0dccccccccccccccccccccccccccccccccccc;.
       .;kKXXX0d:..         .,lOKXXXOc.        'OXXX0c..............';cdOKKkdddl;,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,.
      .oKXXX0l.                .l0XXXKo.       'OXXX0;                  .cOKKKKO:.
     .dKXXXk,                    :0XXXKl.      'OXXX0;                    .dKXXX0:
    .lKXXXO,                      :xdoc,       'OXXX0;                     ,kOOOOx:,,,,,,,''..
    ;OXXXKc                                    'OXXX0;                    .cxxxxxxxxxxxxxxxxdoc.
   .lKXXXk'                                    'OXXX0;                     'oxxxxxxxxxxxxxxxxxx:
   .xXXXXo.                                    'OXXX0;                      .:kOOOko:;;;;;;;;;;.
   'kXXXKl                                     'OXXX0;                       ,OXXXK:
   'kXXXKl                                     'OXXX0;                       ,OXXXK:
   .xXXXXo.                                    'OXXX0;                       ;0XXX0;       .;;;;;;;;;;;;;;;;,'.
    lKXXXx.                                    'OXXX0;                       lKXXXk'      .cxxxxxxxxxxxxxxxxxdl'
    ,OXXX0:                        ;c:,..      'OXXX0;                      .xXXXKo.       'cdxxxxxxxxxxxxxxxxxc.
    .lKXXXx.                      ,OXXX0c      'OXXX0;                     .lKXXXO,          ..',,,;;;;;;;,;;;,.
     .xXXXKd.                    'kXXXXx.      'OXXX0;                    .l0XXX0c
      'xKXXKk;.                .:OXXXKx'       'OXXX0;                   ,dKXXX0c
       .o0XXXKxc'.           .:xKXXXKd.        'OXXX0:             ...,cx0K0OOOxc:;;;;;;;;;;;;;;;;;;;;;;;;;;;;;'
         ;xKXXXX0kdl:;;;;:cox0KXXXKx;.         'OXXXKOdddddddddddxxkO0KXXXKOxxkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkxkxc.
           ,lk0XXXXXXXXXXXXXXXXKko,.           'OXXXXXXXXXXXXXXXXXXXXXKKOxddkkxkkkkkkkkkkkkkkkkkkkkkkkxxxxdol:,.
             .';codkkOOOOOkxdl:'.              .cooooooooooooooooollc:;'.  .;;;;;;;;;;;;;;;;;;;;;;;,,,,'....
                    .......


connecting to cds api %s...
  > `, client.APIURL())
}
