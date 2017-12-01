package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"strconv"

	"github.com/BurntSushi/toml"

	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/keychain"
)

type config struct {
	Host                  string
	user                  string
	token                 string
	InsecureSkipVerifyTLS bool
}

func loadConfig(configFile string) (*cdsclient.Config, error) {
	var verbose = os.Getenv("CDS_VERBOSE") == "true"

	c := &config{}
	c.Host = os.Getenv("CDS_API")
	c.user = os.Getenv("CDS_USER")
	c.token = os.Getenv("CDS_TOKEN")
	c.InsecureSkipVerifyTLS, _ = strconv.ParseBool(os.Getenv("CDS_INSECURE"))

	if c.Host != "" && c.user != "" {
		if verbose {
			fmt.Println("Configuration loaded from environment variables")
		}
	}

	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	u, err := user.Current()
	if err != nil {
		return nil, err
	}

	var configFiles []string
	if configFile != "" {
		configFiles = []string{configFile}
	} else {
		configFiles = []string{
			path.Join(dir, ".cdsrc"),
			path.Join(u.HomeDir, ".cdsrc"),
		}
	}

	var i int
	for c.Host == "" && i < len(configFiles) {
		if _, err := os.Stat(configFiles[i]); err == nil {
			b, err := ioutil.ReadFile(configFiles[i])
			if err != nil {
				if verbose {
					fmt.Printf("Unable to read %s \n", configFiles[i])
				}
				return nil, err
			}
			if _, err := toml.Decode(string(b), c); err != nil {
				return nil, err
			}
			if verbose {
				fmt.Println("Configuration loaded from", configFiles[i])
			}
		}
		i++
	}

	if c.Host == "" {
		return nil, fmt.Errorf("unable to load configuration, you should try to login first")
	}

	conf := &cdsclient.Config{
		Host:    c.Host,
		User:    c.user,
		Token:   c.token,
		Verbose: verbose,
	}

	return conf, nil
}

func loadClient(c *cdsclient.Config) (cdsclient.Interface, error) {
	user, secret, err := keychain.GetSecret(c.Host)
	if err != nil {
		return nil, err
	}
	c.User = user
	c.Token = secret
	return cdsclient.New(*c), nil
}
