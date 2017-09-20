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
)

// TagOpts represents options for git tag command
type TagOpts struct {
	Message  string
	SignKey  string
	SignID   string
	Name     string
	Path     string
	Username string
}

// Tag makes git tag action
func Tag(repo string, auth *AuthOpts, opts *TagOpts, output *OutputOpts) error {
	if strings.HasPrefix(repo, "http://") || strings.HasPrefix(repo, "ftp://") || strings.HasPrefix(repo, "ftps://") {
		return fmt.Errorf("Git protocol not supported")
	}
	if strings.HasPrefix(repo, "https://") {
		return gitTagOverHTTPS(repo, auth, opts, output)
	}
	return gitTagOverSSH(auth, opts, output)
}

func gitTagOverHTTPS(repo string, auth *AuthOpts, opts *TagOpts, output *OutputOpts) error {
	if auth == nil {
		cmd := gitTagCommand(opts)
		return runCommand(cmd, output)
	}
	u, err := url.Parse(repo)
	if err != nil {
		return err
	}

	u.User = url.UserPassword(auth.Username, auth.Password)

	cmd := gitTagCommand(opts)
	return runCommand(cmd, output)
}

func gitTagOverSSH(auth *AuthOpts, opts *TagOpts, output *OutputOpts) error {
	if auth == nil {
		return fmt.Errorf("Authentication is required for git over ssh")
	}

	keyDir := filepath.Dir(auth.PrivateKey.Filename)
	allCmd := gitTagCommand(opts)

	gitSSHCmd := exec.Command("ssh").Path
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

func gitTagCommand(opts *TagOpts) cmds {
	allCmd := []cmd{}

	if opts != nil && opts.SignKey != "" {
		// Create command to import key
		importpubCmd := cmd{
			cmd:  "gpg",
			args: []string{"--import", "pgp.pub.key"},
		}
		allCmd = append(allCmd, importpubCmd)

		importcmd := cmd{
			cmd:  "gpg",
			args: []string{"--import", "pgp.key"},
		}

		allCmd = append(allCmd, importcmd)
	}

	allCmd = append(allCmd, gitConfigCommand("user.name", opts.Username))
	allCmd = append(allCmd, gitConfigCommand("user.email", "cds@localhost"))

	gitcmd := cmd{
		cmd:  "git",
		args: []string{"tag"},
	}

	// Option for git push after tagging
	optPush := &PushOpts{}

	if opts != nil {
		if opts.Path != "" {
			gitcmd.dir = opts.Path
			optPush.Directory = opts.Path
		}
		gitcmd.args = append(gitcmd.args, "-a", opts.Name)

		if opts.Message != "" {
			gitcmd.args = append(gitcmd.args, "-m", fmt.Sprintf("\"%s\"", opts.Message))
		}
		if opts.SignKey != "" {
			gitcmd.args = append(gitcmd.args, "-u", opts.SignID)
		}
		optPush.Branch = opts.Name
	}

	allCmd = append(allCmd, gitcmd)
	allCmd = append(allCmd, gitPushCommand(optPush)...)
	return cmds(allCmd)
}

// TagList List tag from given git directory
func TagList(dir string, output *OutputOpts) error {
	return runCommand(gitTagListCommand(dir), output)
}

func gitTagListCommand(dir string) cmds {
	allCmd := []cmd{}

	gitcmd := cmd{
		cmd:  "git",
		args: []string{"ls-remote", "--tags", "--refs", "origin"},
		dir:  dir,
	}

	allCmd = append(allCmd, gitcmd)
	return cmds(allCmd)
}
