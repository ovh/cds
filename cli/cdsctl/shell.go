package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"
)

var shellCmd = cli.Command{
	Name:  "shell",
	Short: "cdsctl interactive shell",
	Long: `
CDS Shell Mode. Keywords:
- ls: display current list
- ls <KEY>: display current object, ls MY_PRJ is the same as cdsctl project show MY_PRJ
- version: same as cdsctl version command
- cd: go to an object, try to run "ls" after a cd

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
}

// isInProjects returns true if there is nothing selected
func (s shellCurrent) isInProjects() bool {
	return s.project == "" && s.workflow == "" && s.application == "" && s.pipeline == ""
}

type shellCommandFunc func(args []string, current *shellCurrent) *cobra.Command

var (
	shellCommands = map[string]shellCommandFunc{
		"ls":  lsCommand,
		"dir": lsCommand,
		"version": func(args []string, current *shellCurrent) *cobra.Command {
			return cli.NewCommand(versionCmd, versionRun, nil, cli.CommandWithoutExtraFlags)
		},
	}

	lsCommand = func(args []string, current *shellCurrent) *cobra.Command {
		if current.isInProjects() {
			if len(args) == 1 {
				return cli.NewGetCommand(projectShowCmd, projectShowRun, nil, withAllCommandModifiers()...)
			}
			return cli.NewListCommand(projectListCmd, projectListRun, nil, withAllCommandModifiers()...)
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
		fmt.Printf("Command %s", cmd.Short)
		if len(tuple[1:]) > 0 {
			fmt.Printf(" with args: %+v", tuple[1:])
		}
		fmt.Println()
		cmd.SetArgs(tuple[1:])
		if err := cmd.Execute(); err != nil {
			fmt.Printf("Error while executing command: %s\n", err)
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
