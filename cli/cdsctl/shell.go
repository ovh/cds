package main

import (
	"bytes"
	"context"
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

func shell() *cobra.Command {
	return cli.NewCommand(shellCmd, shellRun, nil, cli.CommandWithoutExtraFlags)
}

type shellCurrent struct {
	cmd        string // contains first word: "ls", "cd", etc...
	path, home string
	tree       *cobra.Command
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

	// prepare cobra command tree for shell
	current := &shellCurrent{
		tree: rootFromSubCommands([]*cobra.Command{
			projectShell(),
			adminShell(),
		}),
	}

	l, err := readline.NewEx(&readline.Config{
		Prompt:            "\033[31m»\033[0m ",
		HistoryFile:       path.Join(userHomeDir(), ".cdsctl_history"),
		AutoComplete:      getCompleter(current),
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
	})
	if err != nil {
		fmt.Printf("Error: %s\n", err)
	}

	defer l.Close()

	home := "/"

	ctx := context.Background()
	// try to discover conf for existing .git repository
	r, errR := repo.New(ctx, ".")
	if errR == nil {
		if _, err := discoverConf([]cli.Arg{
			{Name: _ProjectKey},
			{Name: _ApplicationName, AllowEmpty: true},
			{Name: _WorkflowName, AllowEmpty: true},
		}); err == nil {
			if proj, _ := r.LocalConfigGet(ctx, "cds", "project"); proj != "" {
				home = "/project/" + proj
				if wf, _ := r.LocalConfigGet(ctx, "cds", "workflow"); wf != "" {
					home += "/workflow/" + wf
				} else if app, _ := r.LocalConfigGet(ctx, "cds", "application"); app != "" {
					home += "/application/" + app
				}
			}
		}
	}

	current.home = home
	current.path = home

	for {
		l.SetPrompt(fmt.Sprintf("\033[96m%s\033[0m \033[31m»\033[0m ", current.pwdCmd()))
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
			current.shellProcessCommand(l, line)
		}
	}

	return nil
}

func getCompleter(s *shellCurrent) *readline.PrefixCompleter {
	return readline.NewPrefixCompleter(
		readline.PcItem("mode",
			readline.PcItem("vi"),
			readline.PcItem("emacs"),
		),
		readline.PcItem("help"),
		readline.PcItem("cd",
			readline.PcItemDynamic(listCurrent(s, false)),
			readline.PcItemDynamic(findComplete()),
		),
		readline.PcItem("ls",
			readline.PcItemDynamic(listCurrent(s, false)),
		),
		readline.PcItem("ll",
			readline.PcItemDynamic(listCurrent(s, false)),
		),
		readline.PcItem("open"),
		readline.PcItem("pwd"),
		readline.PcItem("version"),
		readline.PcItem("exit"),
		readline.PcItemDynamic(listCurrent(s, true)),
	)
}

func listCurrent(s *shellCurrent, onlyCommands bool) func(string) []string {
	return func(line string) []string {
		if onlyCommands {
			_, _, _, cmds, _ := s.shellListCommand(s.path, nil, onlyCommands)
			return sdk.DeleteEmptyValueFromArray(cmds)
		}
		_, items, submenus, _, _ := s.shellListCommand(s.path, nil, onlyCommands)
		return sdk.DeleteEmptyValueFromArray(append(items, submenus...))
	}
}

type shellCommandFunc func(current *shellCurrent, args []string)

func getShellCommands(rline *readline.Instance, s *shellCurrent) map[string]shellCommandFunc {
	return map[string]shellCommandFunc{
		"mode": func(current *shellCurrent, args []string) {
			if len(args) == 0 {
				if rline.IsVimMode() {
					println("current mode: vim")
				} else {
					println("current mode: emacs")
				}
			} else {
				switch args[0] {
				case "vi":
					rline.SetVimMode(true)
				case "emacs":
					rline.SetVimMode(false)
				default:
					fmt.Println("invalid mode:", args[0])
				}
			}
		},
		"help": func(current *shellCurrent, args []string) { fmt.Println(shellCmd.Long) },
		"cd": func(current *shellCurrent, args []string) {
			if !s.cdCmd(args) {
				fmt.Println("no such item or command")
			}
		},
		"find":    func(current *shellCurrent, args []string) { s.findCmd(args) },
		"open":    func(current *shellCurrent, args []string) { s.openBrowser() },
		"ls":      func(current *shellCurrent, args []string) { s.lsCmd(args) },
		"ll":      func(current *shellCurrent, args []string) { s.lsCmd(args) },
		"pwd":     func(current *shellCurrent, args []string) { fmt.Println(s.pwdCmd()) },
		"version": func(current *shellCurrent, args []string) { _ = versionRun(nil) },
	}
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
		if len(nav) == 0 {
			return []string{fmt.Sprintf("no project found")}
		}
		out := make([]string, len(nav))
		for i, p := range nav {
			switch p.Type {
			case "project":
				out[i] = "/project/" + p.Key
			case "application":
				out[i] = "/project/" + p.Key + "/application/" + p.ApplicationName
			case "workflow":
				out[i] = "/project/" + p.Key + "/workflow/" + p.WorkflowName
			}
		}
		return out
	}
}

func (s *shellCurrent) cdCmd(args []string) bool {
	path := s.path
	defer func() { s.path = path }()

	if len(args) == 0 || args[0] == "" {
		path = s.home
		return true
	}

	split := strings.Split(args[0], "/")
	for i, s := range split {
		if s == "" {
			// check for absolute path
			if i == 0 {
				path = "/"
			}
		} else if s == ".." {
			idx := strings.LastIndex(path, "/")
			if idx >= 0 {
				path = path[:idx]
			}
		} else if s != "" && s != "." {
			if path != "/" {
				path += "/"
			}
			path += s
		}
	}

	split = strings.Split(path, "/")
	if len(split) > 0 && path != "/" {
		for i := 1; i < len(split); i++ {
			_, items, submenus, _, _ := s.shellListCommand(strings.Join(split[:i], "/"), nil, false)
			var found bool
			for _, v := range append(items, submenus...) {
				if v == split[i] {
					found = true
					break
				}
			}
			if !found {
				path = s.path
				return false
			}
		}
	}

	return true
}

func (s *shellCurrent) findCmd(args []string) {
	var search string
	if len(args) > 0 {
		search = args[0]
	}

	nav, err := client.Navbar()
	if err != nil {
		fmt.Printf("Error while getting data: %s\n", err)
	}
	if len(nav) == 0 {
		fmt.Println("no project found")
	}
	r, _ := regexp.Compile("(?i).*(" + search + ").*")

	for _, prj := range nav {
		switch prj.Type {
		case "project":
			s := r.FindStringSubmatch(prj.Name)
			s2 := r.FindStringSubmatch(prj.Key)
			if len(s) == 2 || len(s2) == 2 {
				fmt.Println("/project/" + prj.Key)
			}
		case "application":
			s := r.FindStringSubmatch(prj.ApplicationName)
			if len(s) == 2 {
				fmt.Println("/project/" + prj.Key + "/application/" + prj.ApplicationName)
			}
		case "workflow":
			s := r.FindStringSubmatch(prj.WorkflowName)
			if len(s) == 2 {
				fmt.Println("/project/" + prj.Key + "/workflow/" + prj.WorkflowName)
			}
		}
	}
}

func (s *shellCurrent) lsCmd(args []string) {
	path := s.path
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		if strings.HasPrefix(args[0], "/") { // absolute path
			path = args[0]
		} else { // relative path
			path = s.path + args[0]
		}
		args = args[1:]
	}

	output, _, submenus, cmds, _ := s.shellListCommand(path, args, false)
	fmt.Print(output)
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

func (s *shellCurrent) pwdCmd() string {
	if s.path == "" {
		return "/"
	}
	return s.path
}

func (s *shellCurrent) shellProcessCommand(rline *readline.Instance, input string) {
	tuple := strings.Split(input, " ")
	s.cmd = tuple[0]
	if f, ok := getShellCommands(rline, s)[s.cmd]; ok {
		if f == nil {
			fmt.Printf("Command %s not defined in this context\n", input)
			return
		}
		f(s, tuple[1:])
		return
	}
	// default commands not found, search a sub commands
	_, _, _, _, cdmsCobra := s.shellListCommand(s.path, tuple[1:], true)
	for _, c := range cdmsCobra {
		if c.Name() == s.cmd {
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
			c.Run(c, append(s.getArgsFromPathForCmd(s.path, c), args...))
			return
		}
	}
	fmt.Println("unknown command", input)
}

func (s *shellCurrent) shellListCommand(path string, flags []string, onlyCommands bool) (string, []string, []string, []string, []*cobra.Command) {
	spath := strings.Split(path, "/")

	cmd := s.tree
	// try to find recursively the cobra command that match given path
	// /project/TEST/workflow/test -> cmd = workflow
	for i := range spath {
		key := spath[i]
		if f := findCommand(cmd, key); f != nil {
			cmd = f
		}
	}
	cmdStrict := cmd.Name() == spath[len(spath)-1] || (cmd.Name() == s.tree.Name() && spath[len(spath)-1] == "")

	var out string
	var items []string
	if !onlyCommands {
		buf := new(bytes.Buffer)
		// if the command found is at the end of given path
		if cmdStrict {
			// if command has a list sub command, execute it
			if lsCmd := findCommand(cmd, "list"); lsCmd != nil {
				if len(flags) == 0 {
					flags = []string{"-q"}
				}
				lsCmd.ParseFlags(flags)
				lsCmd.SetOutput(buf)
				lsCmd.Run(lsCmd, s.getArgsFromPathForCmd(path, lsCmd))
				if buf.Len() > 0 {
					for _, v := range strings.Split(buf.String(), "\n") {
						if v != "" {
							items = append(items, v)
						}
					}
				}
			}
		} else { // try show command
			if showCmd := findCommand(cmd, "show"); showCmd != nil {
				showCmd.ParseFlags(flags)
				showCmd.SetOutput(buf)
				showCmd.Run(showCmd, s.getArgsFromPathForCmd(path, showCmd))
			}
		}
		out = buf.String()
	}

	// compute list sub-menus and commands
	var submenus []string
	for _, c := range cmd.Commands() {
		// list only command with sub commands
		if len(c.Commands()) > 0 && c.Run == nil {
			var hasShowOrListCdm bool
			allContextValid := true
			for _, sub := range c.Commands() {
				if !s.isCtxOK(path, sub) {
					allContextValid = false
					continue
				}
				if sub.Name() == "show" || sub.Name() == "list" {
					hasShowOrListCdm = true
					break
				}
			}
			if hasShowOrListCdm || allContextValid {
				submenus = append(submenus, c.Name())
			}
		}
	}

	var cmds []string
	var cmdsCobra []*cobra.Command
	for _, c := range cmd.Commands() {
		if len(c.Commands()) == 0 && c.Name() != "list" && c.Name() != "show" {
			if (!s.hasCtx(path, c) && cmdStrict) ||
				(s.hasCtx(path, c) && s.isCtxOK(path, c) && !s.isCtxOK(strings.Join(spath[:len(spath)-2], "/"), c)) {
				cmds = append(cmds, c.Name())
				cmdsCobra = append(cmdsCobra, c)
			}
		}
	}

	if onlyCommands {
		return "", nil, nil, cmds, cmdsCobra
	}

	return out, items, submenus, cmds, nil
}

func (s *shellCurrent) hasCtx(path string, cmd *cobra.Command) bool {
	if _, withContext := s.extractArg(path, cmd, _ProjectKey); withContext {
		return true
	}
	if _, withContext := s.extractArg(path, cmd, _ApplicationName); withContext {
		return true
	}
	if _, withContext := s.extractArg(path, cmd, _WorkflowName); withContext {
		return true
	}
	return false
}

func (s *shellCurrent) isCtxOK(path string, cmd *cobra.Command) bool {
	if a, withContext := s.extractArg(path, cmd, _ProjectKey); withContext && a == "" {
		return false
	}
	if a, withContext := s.extractArg(path, cmd, _ApplicationName); withContext && a == "" {
		return false
	}
	if a, withContext := s.extractArg(path, cmd, _WorkflowName); withContext && a == "" {
		return false
	}
	return true
}

// key: _ProjectKey, _ApplicationName, _WorkflowName
// pos: position to extract
func (s *shellCurrent) extractArg(path string, cmd *cobra.Command, key string) (string, bool) {
	var inpath string
	switch key {
	case _ApplicationName:
		inpath = "application"
	case _WorkflowName:
		inpath = "workflow"
	}

	// check is ctx key is in cmd use, ex: [ PROJECT-KEY ]
	split := strings.Split(cmd.Use, "[")
	if len(split) < 2 {
		return "", false
	}
	split = strings.Split(split[1], "]")

	cmdWithContext := strings.Contains(split[0], strings.ToUpper(key))
	if cmdWithContext {
		if strings.HasPrefix(path, "/project/") {
			t := strings.Split(path, "/")
			if inpath == "" {
				return t[2], cmdWithContext
			} else if inpath != "" && len(t) >= 5 && t[3] == inpath {
				return t[4], cmdWithContext
			}
		}
	}
	return "", cmdWithContext
}

func (s *shellCurrent) getArgsFromPathForCmd(path string, cmd *cobra.Command) []string {
	args := []string{}
	if a, _ := s.extractArg(path, cmd, _ProjectKey); a != "" {
		args = append(args, a)
	}
	if a, _ := s.extractArg(path, cmd, _ApplicationName); a != "" {
		args = append(args, a)
	}
	if a, _ := s.extractArg(path, cmd, _WorkflowName); a != "" {
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

func (s *shellCurrent) openBrowser() {
	configUser, err := client.ConfigUser()
	if err != nil {
		fmt.Printf("Error while getting URL UI: %s", err)
		return
	}

	if configUser.URLUI == "" {
		fmt.Println("Unable to retrieve webui uri")
		return
	}

	_ = browser.OpenURL(configUser.URLUI + s.path)
}

func shellASCII() {
	fmt.Printf(`
   ___ ___  ___
  / __|   \/ __|
 | (__| |) \__ \   connecting to cds api %s...
  \___|___/|___/     > `, client.APIURL())
}
