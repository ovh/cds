package vcs

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ovh/cds/sdk"
)

// SetupSSHKey writes all the keys in the path, or just the specified key it not nil
func SetupSSHKey(vars []sdk.Variable, path string, key *sdk.Variable) error {
	if key == nil {
		for _, v := range vars {
			if v.Type != sdk.KeyVariable && v.Type != sdk.KeySSHParameter {
				continue
			}
			if err := write(path, v.Name, v.Value); err != nil {
				return err
			}
		}
		return nil
	}
	return write(path, key.Name, key.Value)
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

	p := filepath.Join(path, key.Name)
	b, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}

	return &SSHKey{Filename: p, Content: b}, nil
}

func write(path, name, content string) error {
	path = filepath.Join(path, name)

	if err := ioutil.WriteFile(path, []byte(content), os.FileMode(0600)); err != nil {
		return err
	}

	return nil
}
