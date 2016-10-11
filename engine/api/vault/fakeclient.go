package vault

import (
	"bufio"
	"log"
	"os"
	"path"
	"strings"
)

//LocalInsecureClient is for dev purpose only
type LocalInsecureClient struct {
	Secrets map[string]string
}

//Load Loads secrets from a directory
func (c *LocalInsecureClient) Load(dirname string) error {
	c.Secrets = map[string]string{}
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
					log.Fatal(err)
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
				c.Secrets[key] = value
			}
		}
	}
	return nil
}

//GetSecrets is for dev purpose only
func (c *LocalInsecureClient) GetSecrets() (map[string]string, error) {
	return c.Secrets, nil
}
