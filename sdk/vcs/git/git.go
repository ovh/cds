package git

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

func getRepoURL(repo string, auth *AuthOpts) (string, error) {
	if strings.HasPrefix(repo, "http://") || strings.HasPrefix(repo, "ftp://") || strings.HasPrefix(repo, "ftps://") {
		return "", fmt.Errorf("Git protocol not supported")
	}
	if auth != nil && strings.HasPrefix(repo, "https://") {
		u, err := url.Parse(repo)
		if err != nil {
			return "", err
		}
		u.User = url.UserPassword(auth.Username, auth.Password)
		return u.String(), nil
	}
	return repo, nil
}

func runGitCommands(repo string, commands []cmd, auth *AuthOpts, output *OutputOpts) error {
	if strings.HasPrefix(repo, "https://") {
		return runGitCommandRaw(commands, output)
	}
	return runGitCommandsOverSSH(commands, auth, output)
}

func runGitCommandsOverSSH(commands []cmd, auth *AuthOpts, output *OutputOpts) error {
	if auth == nil {
		return fmt.Errorf("Authentication is required for git over ssh")
	}

	keyDir := filepath.Dir(auth.PrivateKey.Filename)

	gitSSHCmd := exec.Command("ssh").Path
	gitSSHCmd += " -i " + auth.PrivateKey.Filename
	gitSSHCmd += " -o StrictHostKeyChecking=no"

	var wrapper string
	if runtime.GOOS == "windows" {
		gitSSHCmd += " %*"
		wrapper = gitSSHCmd
	} else {
		gitSSHCmd += ` "$@"`
		wrapper = `#!/bin/sh
` + gitSSHCmd
	}

	wrapperPath := filepath.Join(keyDir, "gitwrapper")
	if err := ioutil.WriteFile(wrapperPath, []byte(wrapper), os.FileMode(0700)); err != nil {
		return err
	}

	return runGitCommandRaw(commands, output, "GIT_SSH="+wrapperPath)
}

func runGitCommandRaw(cmds cmds, output *OutputOpts, envs ...string) error {
	osEnv := os.Environ()
	for _, e := range envs {
		osEnv = append(osEnv, e)
	}
	for _, c := range cmds {
		for i, arg := range c.args {
			c.args[i] = os.ExpandEnv(arg)
		}
		cmd := exec.Command(c.cmd, c.args...)
		if c.dir != "" {
			cmd.Dir = os.ExpandEnv(c.dir)
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
