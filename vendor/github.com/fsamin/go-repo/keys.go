package repo

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// sshKey is a type for a ssh key
type sshKey struct {
	filename string
	content  []byte
}

// pgpKey is a type for a pgp key
type pgpKey struct {
	name    string
	public  string
	private string
	id      string
}

func (r Repo) setuoSSHKey() ([]string, error) {
	if r.sshKey == nil {
		return nil, fmt.Errorf("no ssh keys to setup")
	}

	gitSSHCmd := exec.Command("ssh").Path
	gitSSHCmd += " -i " + r.sshKey.filename
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

	keyDir := filepath.Dir(r.sshKey.filename)
	wrapperPath := filepath.Join(keyDir, "gitwrapper")
	if err := ioutil.WriteFile(wrapperPath, []byte(wrapper), os.FileMode(0700)); err != nil {
		return nil, err
	}

	return []string{"GIT_SSH=" + wrapperPath, "PKEY=" + r.sshKey.filename}, nil
}

func (r Repo) instalGPGKey() error {
	return nil
}
