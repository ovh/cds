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
	"strings"
	"syscall"

	"github.com/ovh/cds/sdk"
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
	SignKey    vcs.PGPKey
}

// OutputOpts is a optional structs for git clone command
type OutputOpts struct {
	Stdout io.Writer
	Stderr io.Writer
}

type cmds []cmd

func (c cmds) Strings() []string {
	res := make([]string, len(c))
	for i := range c {
		res[i] = c[i].String()
	}
	return res
}

type cmd struct {
	workdir string
	cmd     string
	args    []string
}

func (c cmd) String() string {
	return c.cmd + " " + strings.Join(c.args, " ")
}

func getRepoURL(repo string, auth *AuthOpts) (string, error) {
	if strings.HasPrefix(repo, "http://") || strings.HasPrefix(repo, "ftp://") || strings.HasPrefix(repo, "ftps://") {
		return "", sdk.WithStack(fmt.Errorf("Git protocol not supported"))
	}
	if auth != nil && strings.HasPrefix(repo, "https://") {
		u, err := url.Parse(repo)
		if err != nil {
			return "", sdk.WithStack(err)
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
		return sdk.WithStack(fmt.Errorf("Authentication is required for git over ssh"))
	}

	pkAbsFileName, err := filepath.Abs(auth.PrivateKey.Filename)
	if err != nil {
		return sdk.WithStack(err)
	}
	keyDir := filepath.Dir(pkAbsFileName)

	gitSSHCmd := exec.Command("ssh").Path
	gitSSHCmd += " -F /dev/null -o IdentitiesOnly=yes -o StrictHostKeyChecking=no"
	gitSSHCmd += " -i " + pkAbsFileName

	var wrapper, wrapperPath string
	if sdk.GOOS == "windows" {
		gitSSHCmd += ` %*`
		wrapper = `@echo off
` + gitSSHCmd
		wrapperPath = filepath.Join(keyDir, "gitwrapper.bat")
	} else {
		gitSSHCmd += ` "$@"`
		wrapper = `#!/bin/sh
` + gitSSHCmd
		wrapperPath = filepath.Join(keyDir, "gitwrapper")
	}

	if err := ioutil.WriteFile(wrapperPath, []byte(wrapper), os.FileMode(0700)); err != nil {
		return sdk.WithStack(err)
	}

	return runGitCommandRaw(commands, output, "GIT_SSH="+wrapperPath)
}

func runGitCommandRaw(cmds cmds, output *OutputOpts, envs ...string) error {
	osEnv := os.Environ()
	osEnv = append(osEnv, envs...)
	for _, c := range cmds {
		for i, arg := range c.args {
			c.args[i] = os.ExpandEnv(arg)
		}
		cmd := exec.Command(c.cmd, c.args...)
		cmd.Dir = c.workdir
		cmd.Env = osEnv

		if verbose {
			LogFunc("Executing Command %s - %v", c, envs)
		}

		if output != nil {
			cmd.Stdout = output.Stdout
			cmd.Stderr = output.Stderr
		}

		if err := cmd.Start(); err != nil {
			return sdk.WithStack(err)
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
					return fmt.Errorf("Command fail: %d", status.ExitStatus())
				}
				return sdk.WithStack(exiterr)
			}
			return sdk.WithStack(err)
		}
	}
	return nil
}
