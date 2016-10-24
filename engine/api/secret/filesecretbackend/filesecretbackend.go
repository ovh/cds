package filesecretbackend

import (
	"bufio"
	"os"
	"path"
	"strings"

	"github.com/ovh/cds/engine/api/secret/secretbackend"
)

type fileSecretBackend struct {
	secrets map[string]string
}

//Client return a SecretBackend
func Client(opts map[string]string) secretbackend.Driver {
	c := &fileSecretBackend{}
	c.Init(secretbackend.NewOptions(opts))
	return c
}

func (c *fileSecretBackend) Name() string {
	return "File Secret Backend - CDS Embbeded"
}

func (c *fileSecretBackend) Init(opts secretbackend.MapVar) error {
	dir := opts.Get("secret_directory")
	if dir == "" {
		dir = ".secrets"
	}
	return c.load(dir)
}

//Load Loads secrets from a directory
func (c *fileSecretBackend) load(dirname string) error {
	c.secrets = map[string]string{}
	d, err := os.Open(dirname)
	if err != nil {
		return err
	}
	defer d.Close()
	fi, err := d.Readdir(-1)
	if err != nil {
		return err
	}
	for _, fi := range fi {
		if fi.Mode().IsRegular() {
			if strings.HasSuffix(fi.Name(), ".key") {
				//Load the key, first line is the secret key, then the secret value
				file, err := os.Open(path.Join(dirname, fi.Name()))
				if err != nil {
					return err
				}
				defer file.Close()
				scanner := bufio.NewScanner(file)
				var i = 0
				var key, value string
				for scanner.Scan() {
					if i == 0 {
						key = scanner.Text()
					} else {
						value = value + scanner.Text() + "\n"
					}
					i++
				}
				c.secrets[key] = value
			}
		}
	}
	return nil
}

//GetSecrets is for dev purpose only
func (c *fileSecretBackend) GetSecrets() secretbackend.Secrets {
	return *secretbackend.NewSecrets(c.secrets)
}
