package main

import (
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var keysDirectory string

// DEPRECATED
const (
	gitsshscript = `#!/bin/sh
if [ -z "$PKEY" ]; then
	ssh "$@"
else
	ssh -oStrictHostKeyChecking=no -i "$PKEY" "$@"
fi`
)

// DEPRECATED
func (w *currentWorker) writeSSHKey(key []byte, keypath string) error {
	log.Debug("writeSSHKey> Writing key in %", keypath)
	err := ioutil.WriteFile(keypath, key, os.FileMode(0600))
	if err != nil {
		return err
	}

	w.currentJob.pkey = keypath
	return nil
}

// DEPRECATED
func (w *currentWorker) writeGitSSH(p string) error {
	p = path.Join(p, "gitssh.sh")
	err := ioutil.WriteFile(p, []byte(gitsshscript), os.FileMode(0770))
	if err != nil {
		return err
	}

	w.currentJob.gitsshPath = p
	log.Debug("writeGitSSH> gitssh.sh is  %s", p)
	return nil
}

// Setup SSH keys will chose from available keys in this order:
// Environment > Application > Project
// This is the DEPRECATED way to setup ssh key
func (w *currentWorker) setupSSHKey(vars []sdk.Variable, keypath string) error {
	log.Debug("setupSSHKey> setup key in %s", keypath)
	var key sdk.Variable
	var prio int

	for _, v := range vars {
		if v.Type != sdk.KeyVariable {
			continue
		}
		log.Info("SSH> Got %s !\n", v.Name)

		var keyprio int
		var prefix string
		if strings.Contains(v.Name, "cds.proj.") {
			keyprio = 1
			prefix = "cds.proj."
		} else if strings.Contains(v.Name, "cds.app.") {
			keyprio = 2
			prefix = "cds.app."
		} else if strings.Contains(v.Name, "cds.env.") {
			keyprio = 3
			prefix = "cds.env."
		}

		if keyprio > prio {
			key = v
			key.Name = strings.TrimPrefix(key.Name, prefix)
		}

	}

	if key.Name != "" {
		err := os.MkdirAll(keypath, 0755)
		if err != nil {
			return err
		}

		if err = w.writeGitSSH(keypath); err != nil {
			return err
		}

		return w.writeSSHKey([]byte(key.Value), path.Join(keypath, key.Name))
	}

	return nil
}
