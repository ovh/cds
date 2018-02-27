package main

import (
	"bytes"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"

	"github.com/chzyer/readline"
	repo "github.com/fsamin/go-repo"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var shellCmd = cli.Command{
	Name:  "shell",
	Short: "cdsctl interactive shell",
	Long: `
CDS Shell Mode. default commands:

- cd: reset current position.
- cd <SOMETHING>: go to an object. Try cd /project/ and tabulation to autocomplete
- find <SOMETHING>: find a project / application / workflow. not case sensitive
- help: display this help
- ls: display current list, quiet format
- ll: display current list
- mode: display current mode. Choose mode with "mode vi" ou "mode emacs"
- open: open CDS WebUI with current context
- version: same as cdsctl version command

Other commands are available depending on your position. Example, run interactively a workflow:


	cd /project/MY_PRJ_KEY/workflow/MY_WF
	run -i

[![asciicast](https://asciinema.org/a/fTFpJ5uqClJ0Oq2EsiejGSeBk.png)](https://asciinema.org/a/fTFpJ5uqClJ0Oq2EsiejGSeBk)

[![asciicast](https://asciinema.org/a/H67HlKNS2r0daQaEcuNfZhZZd.png)](https://asciinema.org/a/H67HlKNS2r0daQaEcuNfZhZZd)
	

`,
}

var current *shellCurrent

type shellCurrent struct {
	cmd   string // contains first word: "ls", "cd", etc...
	path  string
	rline *readline.Instance
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

	l, err := readline.NewEx(&readline.Config{
		Prompt:            "\033[31m»\033[0m ",
		HistoryFile:       path.Join(userHomeDir(), ".cdsctl_history"),
		AutoComplete:      getCompleter(),
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
	})

	if err != nil {
		fmt.Printf("Error: %s\n", err)
	}

	defer l.Close()

	current = &shellCurrent{rline: l}

	// auto-discover current project with .git
	if err := discoverConf(); err == nil {
		if r, err := repo.New("."); err == nil {
			if proj, _ := r.LocalConfigGet("cds", "project"); proj != "" {
				current.path = "/project/" + proj
				if wf, _ := r.LocalConfigGet("cds", "workflow"); wf != "" {
					current.path += "/workflow/" + wf
				} else if app, _ := r.LocalConfigGet("cds", "application"); app != "" {
					current.path += "/application/" + app
				}
			}
		}
	}

	for {
		l.SetPrompt(fmt.Sprintf("%s \033[31m»\033[0m ", current.pwd()))
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
		if len(line) > 0 {
			current.shellProcessCommand(line)
		}
	}
	return nil
}

func getCompleter() *readline.PrefixCompleter {
	return readline.NewPrefixCompleter(
		readline.PcItem("mode",
			readline.PcItem("vi"),
			readline.PcItem("emacs"),
		),
		readline.PcItem("help"),
		readline.PcItem("cd",
			readline.PcItemDynamic(listCurrent(false)),
			readline.PcItemDynamic(findComplete()),
		),
		readline.PcItem("ls",
			readline.PcItemDynamic(listCurrent(false)),
		),
		readline.PcItem("ll",
			readline.PcItemDynamic(listCurrent(false)),
		),
		readline.PcItem("open"),
		readline.PcItem("pwd"),
		readline.PcItem("version"),
		readline.PcItem("exit"),
		readline.PcItemDynamic(listCurrent(true)),
	)
}

func listCurrent(onlyCommands bool) func(string) []string {
	return func(line string) []string {
		if onlyCommands {
			_, _, cmds, _ := current.shellListCommand(current.path, nil, onlyCommands)
			return sdk.DeleteEmptyValueFromArray(cmds)
		}
		output, submenus, _, _ := current.shellListCommand(current.path, nil, onlyCommands)
		return sdk.DeleteEmptyValueFromArray(append(output, submenus...))
	}
}

type shellCommandFunc func(current *shellCurrent, args []string)

func getShellCommands() map[string]shellCommandFunc {
	m := map[string]shellCommandFunc{
		"mode": func(current *shellCurrent, args []string) {
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
		},
		"help": func(current *shellCurrent, args []string) {
			fmt.Println(shellCmd.Long)
		},
		"cd": func(current *shellCurrent, args []string) {
			if len(args) == 0 {
				current.path = ""
				return
			}

			if args[0] == ".." {
				idx := strings.LastIndex(current.path, "/")
				current.path = current.path[:idx]
				return
			}

			// path must start with / and end without /
			if strings.HasPrefix(args[0], "/") { // absolute cd /...
				current.path = args[0]
			} else { // relative cd foo...
				current.path += "/" + args[0]
			}
			current.path = strings.TrimSuffix(current.path, "/")
		},
		"find": func(current *shellCurrent, args []string) {
			if len(args) == 0 {
				current.path = ""
				return
			}
			current.findCmd(args[0])
		},
		"open": func(current *shellCurrent, args []string) {
			current.openBrowser()
		},
		"ls": func(current *shellCurrent, args []string) {
			current.lsCmd(args)
		},
		"ll": func(current *shellCurrent, args []string) {
			current.lsCmd(args)
		},
		"pwd": func(current *shellCurrent, args []string) {
			fmt.Println(current.pwd())
		},
		"version": func(current *shellCurrent, args []string) {
			versionRun(nil)
		},
	}
	return m
}

func findComplete() func(string) []string {
	return func(line string) []string {
		words := strings.Split(line, " ")
		if len(words) == 0 {
			return nil
		}
		nav, err := client.Navbar()
		if err != nil {
			return []string{fmt.Sprintf("Error while getting data: %s\n", err)}
		}
		if len(nav.Projects) == 0 {
			return []string{fmt.Sprintf("no project found")}
		}
		out := []string{}
		for _, p := range nav.Projects {
			out = append(out, "/project/"+p.Key)
			for _, app := range p.ApplicationNames {
				out = append(out, "/project/"+p.Key+"/application/"+app)
			}
			for _, wf := range p.WorkflowNames {
				out = append(out, "/project/"+p.Key+"/workflow/"+wf)
			}
		}
		return out
	}
}

func (current *shellCurrent) findCmd(search string) {
	nav, err := client.Navbar()
	if err != nil {
		fmt.Printf("Error while getting data: %s\n", err)
	}
	if len(nav.Projects) == 0 {
		fmt.Println("no project found")
	}
	r, _ := regexp.Compile("(?i).*(" + search + ").*")

	for _, prj := range nav.Projects {
		s := r.FindStringSubmatch(prj.Name)
		s2 := r.FindStringSubmatch(prj.Key)
		if len(s) == 2 || len(s2) == 2 {
			fmt.Println("/project/" + prj.Key)
		}
		for _, app := range prj.ApplicationNames {
			s := r.FindStringSubmatch(app)
			if len(s) == 2 {
				fmt.Println("/project/" + prj.Key + "/application/" + app)
			}
		}
		for _, wf := range prj.WorkflowNames {
			s := r.FindStringSubmatch(wf)
			if len(s) == 2 {
				fmt.Println("/project/" + prj.Key + "/workflow/" + wf)
			}
		}
	}
}

func (current *shellCurrent) lsCmd(args []string) {
	inargs := args
	path := current.path
	if len(args) == 0 { // ls -> no path
		// default values
	} else {
		if strings.HasPrefix(args[0], "/") { // ls /foo -> absolute path
			path = args[0]
			inargs = args[1:]
		} else if strings.HasPrefix(args[0], "-") { // ls foo -> relative path
			// default values
		} else { // ls foo -> relative path
			path = current.path + args[0]
			inargs = args[1:]
		}
	}

	output, submenus, cmds, _ := current.shellListCommand(path, inargs, false)
	for _, s := range output {
		if len(strings.TrimSpace(s)) > 0 {
			fmt.Println(s)
		}
	}
	if len(submenus) > 0 || len(cmds) > 0 {
		fmt.Println() // empty line between list data and sub-menus/commands list
	}
	if len(submenus) > 0 {
		fmt.Printf("\033[32m»\033[0m sub-menu: %s\n", strings.Join(submenus, " - "))
	}

	if len(cmds) > 0 {
		fmt.Printf("\033[32m»\033[0m additional commands: %s\n", strings.Join(cmds, " - "))
	}
}

func (current *shellCurrent) pwd() string {
	if current.path == "" {
		return "/"
	}
	return current.path
}

func (current *shellCurrent) shellProcessCommand(input string) {
	tuple := strings.Split(input, " ")
	current.cmd = tuple[0]
	if f, ok := getShellCommands()[current.cmd]; ok {
		if f == nil {
			fmt.Printf("Command %s not defined in this context\n", input)
			return
		}
		f(current, tuple[1:])
		return
	}
	// default commands not found, search a sub commands
	_, _, _, cdmsCobra := current.shellListCommand(current.path, tuple[1:], true)
	for _, c := range cdmsCobra {
		if c.Name() == current.cmd {
			flags := tuple[1:]
			if sdk.IsInArray("-h", flags) || sdk.IsInArray("--help", flags) {
				c.Usage()
				return
			}
			c.ParseFlags(flags)
			args := []string{}
			for _, a := range flags {
				if !strings.HasPrefix(a, "-") {
					args = append(args, a)
				}
			}
			c.Run(c, append(current.getArgs(c), args...))
			return
		}
	}
	fmt.Println("unknown command", input)
}

func (current *shellCurrent) shellListCommand(path string, flags []string, onlyCommands bool) ([]string, []string, []string, []*cobra.Command) {
	spath := strings.Split(path, "/")
	cmd := getRoot(true)
	for index := 1; index < len(spath); index++ {
		key := spath[index]
		if f := findCommand(cmd, key); f != nil {
			cmd = f
		}
	}
	if cmd.Name() == "" {
		return []string{"root cmd NOT found"}, nil, nil, nil
	}

	var out []string
	if !onlyCommands {
		buf := new(bytes.Buffer)
		if cmd.Name() == spath[len(spath)-1] { // list command
			if lsCmd := findCommand(cmd, "list"); lsCmd != nil {
				if len(flags) == 0 && current.cmd != "ll" {
					flags = []string{"-q"}
				}
				lsCmd.ParseFlags(flags)
				lsCmd.SetOutput(buf)
				lsCmd.Run(lsCmd, current.getArgs(lsCmd))
			}
		} else { // try show command
			if showCmd := findCommand(cmd, "show"); showCmd != nil {
				showCmd.ParseFlags(flags)
				showCmd.SetOutput(buf)
				showCmd.Run(showCmd, current.getArgs(showCmd))
			}
		}
		out = strings.Split(buf.String(), "\n")
	}

	// compute list sub-menus and commands
	var submenus, cmds []string
	var cmdsCobra []*cobra.Command
	for _, c := range cmd.Commands() {
		// list only command with sub commands
		if len(c.Commands()) > 0 && current.isCtxOK(c) {
			submenus = append(submenus, c.Name())
		} else if c.Name() != "list" && c.Name() != "show" { // list and show are the "ls" cmd
			cmds = append(cmds, c.Name())
			cmdsCobra = append(cmdsCobra, c)
		}
	}

	if onlyCommands {
		return nil, nil, cmds, cmdsCobra
	}

	return out, submenus, cmds, nil
}

func (current *shellCurrent) isCtxOK(cmd *cobra.Command) bool {
	if a, withContext := current.extractArg(cmd, _ProjectKey); withContext && a == "" {
		return false
	}
	if a, withContext := current.extractArg(cmd, _ApplicationName); withContext && a == "" {
		return false
	}
	if a, withContext := current.extractArg(cmd, _WorkflowName); withContext && a == "" {
		return false
	}
	return true
}

// key: _ProjectKey, _ApplicationName, _WorkflowName
// pos: position to extract
func (current *shellCurrent) extractArg(cmd *cobra.Command, key string) (string, bool) {
	var inpath string
	switch key {
	case _ApplicationName:
		inpath = "application"
	case _WorkflowName:
		inpath = "workflow"
	}
	var cmdWithContext bool
	if strings.Contains(cmd.Use, strings.ToUpper(key)) {
		cmdWithContext = true
		if strings.HasPrefix(current.path, "/project/") {
			t := strings.Split(current.path, "/")
			if inpath == "" {
				return t[2], cmdWithContext
			} else if inpath != "" && len(t) >= 5 && t[3] == inpath {
				return t[4], cmdWithContext
			}
		}
	}
	return "", cmdWithContext
}

func (current *shellCurrent) getArgs(cmd *cobra.Command) []string {
	args := []string{}
	if a, _ := current.extractArg(cmd, _ProjectKey); a != "" {
		args = append(args, a)
	}
	if a, _ := current.extractArg(cmd, _ApplicationName); a != "" {
		args = append(args, a)
	}
	if a, _ := current.extractArg(cmd, _WorkflowName); a != "" {
		args = append(args, a)
	}
	return args
}

func findCommand(cmd *cobra.Command, key string) *cobra.Command {
	for _, c := range cmd.Commands() {
		if c.Name() == key {
			return c
		}
	}
	return nil
}

func (current *shellCurrent) openBrowser() {
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
		fmt.Println("Unable to retrieve webui uri")
		return
	}

	browser.OpenURL(baseURL + current.path)
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
