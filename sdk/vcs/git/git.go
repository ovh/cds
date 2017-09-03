package git

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/ovh/cds/sdk/vcs"
)

var (
	verbose bool
	//LogFunc can be overrided
	LogFunc = log.Printf
)

func init() {
	if os.Getenv("CDS_VERBOSE") == "true" {
		verbose = true
	}
}

// AuthOpts is a optional structs for git command
type AuthOpts struct {
	Username   string
	Password   string
	PrivateKey vcs.SSHKey
}

// OutputOpts is a optional structs for git clone command
type OutputOpts struct {
	Stdout io.Writer
	Stderr io.Writer
}

type cmds []cmd

func (c cmds) Strings() []string {
	res := []string{}
	for i := range c {
		res = append(res, c[i].String())
	}
	return res
}

type cmd struct {
	dir  string
	cmd  string
	args []string
}

func (c cmd) String() string {
	return c.cmd + " " + strings.Join(c.args, " ")
}

func runCommand(cmds cmds, output *OutputOpts, envs ...string) error {
	osEnv := os.Environ()
	for _, e := range envs {
		osEnv = append(osEnv, e)
	}
	for _, c := range cmds {
		cmd := exec.Command(c.cmd, c.args...)
		if c.dir != "" {
			cmd.Dir = c.dir
		}
		cmd.Env = osEnv

		if verbose {
			LogFunc("Executing Command %s - %v", c, envs)
		}

		if output != nil {
			cmd.Stdout = output.Stdout
			cmd.Stderr = output.Stderr
		}

		if err := cmd.Start(); err != nil {
			return err
		}

		//close stdin
		stdin, _ := cmd.StdinPipe()
		if stdin != nil {
			stdin.Close()
		}

		if err := cmd.Wait(); err != nil {
			if exiterr, ok := err.(*exec.ExitError); ok {
				if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
					if verbose {
						LogFunc("Command status code %d", status.ExitStatus())
					}
					return fmt.Errorf("Command fail : %d", status.ExitStatus())
				}
				return exiterr
			}
			return err
		}
	}
	return nil
}
