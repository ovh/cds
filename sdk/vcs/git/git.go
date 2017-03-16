package git

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"path/filepath"

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

// CloneOpts is a optional structs for git clone command
type CloneOpts struct {
	Depth                   int
	SingleBranch            bool
	Branch                  string
	Recursive               bool
	Verbose                 bool
	Quiet                   bool
	CheckoutCommit          string
	NoStrictHostKeyChecking bool
}

// OutputOpts is a optional structs for git clone command
type OutputOpts struct {
	Stdout io.Writer
	Stderr io.Writer
}

// Clone make a git clone
func Clone(repo string, path string, auth *AuthOpts, opts *CloneOpts, output *OutputOpts) error {
	if verbose {
		t1 := time.Now()
		defer LogFunc("Git clone %s (%v s)\n", path, int(time.Since(t1).Seconds()))
	}

	if strings.HasPrefix(repo, "http://") || strings.HasPrefix(repo, "ftp://") || strings.HasPrefix(repo, "ftps://") {
		return fmt.Errorf("Git protocol not supported")
	}

	if strings.HasPrefix(repo, "https://") {
		return gitCloneOverHTTPS(repo, path, auth, opts, output)
	}
	return gitCloneOverSSH(repo, path, auth, opts, output)
}

func gitCloneOverHTTPS(repo string, path string, auth *AuthOpts, opts *CloneOpts, output *OutputOpts) error {
	if auth == nil {
		cmd := gitCommand(repo, path, opts)
		return runCommand(cmd, output)
	}
	u, err := url.Parse(repo)
	if err != nil {
		return err
	}

	u.User = url.UserPassword(auth.Username, auth.Password)

	cmd := gitCommand(u.String(), path, opts)
	return runCommand(cmd, output)
}

func gitCloneOverSSH(repo string, path string, auth *AuthOpts, opts *CloneOpts, output *OutputOpts) error {
	if auth == nil {
		return fmt.Errorf("Authentication is required for git over ssh")
	}

	keyDir := filepath.Dir(auth.PrivateKey.Filename)
	allCmd := gitCommand(repo, path, opts)

	gitSSHCmd := exec.Command("ssh").Path
	if opts != nil && opts.NoStrictHostKeyChecking {
		gitSSHCmd += " -o StrictHostKeyChecking=no"
	}
	gitSSHCmd += " -i " + auth.PrivateKey.Filename

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

	return runCommand(allCmd, output, "GIT_SSH="+wrapperPath)
}

func gitCommand(repo string, path string, opts *CloneOpts) cmds {
	allCmd := []cmd{}
	gitcmd := cmd{
		cmd:  "git",
		args: []string{"clone"},
	}

	if opts != nil {
		if opts.Quiet {
			gitcmd.args = append(gitcmd.args, "--quiet")
		} else if opts.Verbose {
			gitcmd.args = append(gitcmd.args, "--verbose")
		}

		if opts.CheckoutCommit == "" {
			if opts.Depth != 0 {
				gitcmd.args = append(gitcmd.args, "--depth", fmt.Sprintf("%d", opts.Depth))
			}
		}

		if opts.Branch != "" {
			gitcmd.args = append(gitcmd.args, "--branch", opts.Branch)
		} else if opts.SingleBranch {
			gitcmd.args = append(gitcmd.args, "--single-branch")
		}

		if opts.Recursive {
			gitcmd.args = append(gitcmd.args, "--recursive")
		}
	}

	gitcmd.args = append(gitcmd.args, repo)

	if path != "" {
		gitcmd.args = append(gitcmd.args, path)
	}

	allCmd = append(allCmd, gitcmd)

	if opts != nil && opts.CheckoutCommit != "" {
		resetCmd := cmd{
			cmd:  "git",
			args: []string{"reset", "--hard", opts.CheckoutCommit},
		}
		//Locate the git reset cmd to the right directory
		if path == "" {
			t := strings.Split(repo, "/")
			resetCmd.dir = strings.TrimSuffix(t[len(t)-1], ".git")
		} else {
			resetCmd.dir = path
		}

		allCmd = append(allCmd, resetCmd)
	}

	return cmds(allCmd)
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
