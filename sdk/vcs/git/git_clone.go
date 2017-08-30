package git

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

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

// Clone make a git clone
func Clone(repo string, path string, auth *AuthOpts, opts *CloneOpts, output *OutputOpts) error {
	if verbose {
		t1 := time.Now()
		if opts != nil && opts.CheckoutCommit != "" {
			defer LogFunc("Checkout commit %s", opts.CheckoutCommit)
		}
		defer LogFunc("Git clone %s (%v s)", path, int(time.Since(t1).Seconds()))
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
		cmd := gitCloneCommand(repo, path, opts)
		return runCommand(cmd, output)
	}
	u, err := url.Parse(repo)
	if err != nil {
		return err
	}

	u.User = url.UserPassword(auth.Username, auth.Password)

	cmd := gitCloneCommand(u.String(), path, opts)
	return runCommand(cmd, output)
}

func gitCloneOverSSH(repo string, path string, auth *AuthOpts, opts *CloneOpts, output *OutputOpts) error {
	if auth == nil {
		return fmt.Errorf("Authentication is required for git over ssh")
	}

	keyDir := filepath.Dir(auth.PrivateKey.Filename)
	allCmd := gitCloneCommand(repo, path, opts)

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

func gitCloneCommand(repo string, path string, opts *CloneOpts) cmds {
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
