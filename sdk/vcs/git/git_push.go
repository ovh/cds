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

// PushOpts represents options for git push command
type PushOpts struct {
	Directory string
	Remote    string
	Branch    string
}

// Push make git push action
func Push(repo string, auth *AuthOpts, opts *PushOpts, output *OutputOpts) error {
	if strings.HasPrefix(repo, "http://") || strings.HasPrefix(repo, "ftp://") || strings.HasPrefix(repo, "ftps://") {
		return fmt.Errorf("Git protocol not supported")
	}
	if strings.HasPrefix(repo, "https://") {
		return gitPushOverHTTPS(repo, auth, opts, output)
	}
	return gitPushOverSSH(auth, opts, output)
}

func gitPushOverHTTPS(repo string, auth *AuthOpts, opts *PushOpts, output *OutputOpts) error {
	if auth == nil {
		cmd := gitPushCommand(opts)
		return runCommand(cmd, output)
	}
	u, err := url.Parse(repo)
	if err != nil {
		return err
	}

	u.User = url.UserPassword(auth.Username, auth.Password)

	cmd := gitPushCommand(opts)
	return runCommand(cmd, output)
}

func gitPushOverSSH(auth *AuthOpts, opts *PushOpts, output *OutputOpts) error {
	if auth == nil {
		return fmt.Errorf("Authentication is required for git over ssh")
	}

	keyDir := filepath.Dir(auth.PrivateKey.Filename)
	allCmd := gitPushCommand(opts)

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

func gitPushCommand(opts *PushOpts) cmds {
	allCmd := []cmd{}
	gitcmd := cmd{
		cmd:  "git",
		args: []string{"push"},
	}

	if opts != nil {
		if opts.Directory != "" {
			gitcmd.dir = opts.Directory
		}
		var remote string
		if opts.Remote == "" {
			remote = "origin"
		}
		gitcmd.args = append(gitcmd.args, remote, opts.Branch)
	}
	allCmd = append(allCmd, gitcmd)
	return cmds(allCmd)
}
