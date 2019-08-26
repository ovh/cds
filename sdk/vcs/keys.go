package vcs

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ovh/cds/sdk"
)

// SetupSSHKey write the key in the specific path
func SetupSSHKey(path string, key sdk.Variable) error {
	path = filepath.Join(path, key.Name)
	return WriteKey(path, key.Value)
}

// CleanAllSSHKeys erase all the keys in the path
func CleanAllSSHKeys(path string) error {
	return os.RemoveAll(path)
}

// CleanSSHKey erase the specified key
func CleanSSHKey(path string, key sdk.Variable) error {
	return os.RemoveAll(filepath.Join(path, "cds.key."+key.Name+".priv"))
}

// SSHKey is a type for a ssh key
type SSHKey struct {
	Filename string
	Content  []byte
}

// PGPKey is a type for a pgp key
type PGPKey struct {
	Name    string
	Public  string
	Private string
	ID      string
}

// GetSSHKey get a key in the path. If the key is nil, it will choose a default key among project, application and env variables
func GetSSHKey(vars []sdk.Variable, path string, key *sdk.Variable) (*SSHKey, error) {
	var k sdk.Variable
	if key == nil {
		var prio int
		for _, v := range vars {
			if !strings.HasPrefix(v.Name, "cds.key.") {
				continue
			}
			var keyprio int
			if strings.HasPrefix(v.Name, "cds.key.proj") {
				keyprio = 1
			} else if strings.HasPrefix(v.Name, "cds.key.app") {
				keyprio = 2
			} else if strings.HasPrefix(v.Name, "cds.key.env") {
				keyprio = 3
			}
			if keyprio > prio {
				k = v
			}
		}
	} else {
		k.Name = key.Name
		k.Value = key.Value
	}

	if k.Name == "" {
		return nil, sdk.ErrKeyNotFound
	}

	p := filepath.Join(path, k.Name)
	b, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}

	return &SSHKey{Filename: p, Content: b}, nil
}

func WriteKey(path, content string) error {
	if err := ioutil.WriteFile(path, []byte(content), os.FileMode(0600)); err != nil {
		return err
	}
	return nil
}
