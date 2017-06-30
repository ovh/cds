package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"regexp"

	"os"

	"github.com/howeyc/gopass"
	"github.com/naoina/toml"
	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/keychain"
)

var loginCmd = cli.Command{
	Name:  "login",
	Short: "Login to CDS",
	Flags: []cli.Flag{
		{
			Name:      "host",
			ShortHand: "H",
			Usage:     "CDS API Url",
			IsValid: func(s string) bool {
				match, _ := regexp.MatchString(`http[s]?:\/\/(.*)`, s)
				return match
			},
		}, {
			Name:      "username",
			ShortHand: "u",
			Usage:     "CDS Username",
		}, {
			Name:      "password",
			ShortHand: "p",
			Usage:     "CDS Password",
		},
	},
}

func loginRun(v cli.Values) error {
	url := v["host"]
	username := v["username"]
	password := v["password"]

	fmt.Println("CDS API Url:", url)

	//Take the user from flags or ask for on command line
	if username == "" {
		fmt.Printf("Username: ")
		username = cli.ReadLine()
	} else {
		fmt.Println("Username:", username)

	}

	//Take the password from flags or ask for on command line
	if password == "" {
		//Ask for the password
		fmt.Printf("Password: ")
		b, err := gopass.GetPasswd()
		password = string(b)
		if err != nil {
			cli.ExitOnError(err)
		}
	} else {
		fmt.Println("Password: ********")
	}

	conf := cdsclient.Config{
		Host:    url,
		Verbose: os.Getenv("CDS_VERBOSE") == "true",
	}

	client = cdsclient.New(conf)
	ok, token, err := client.UserLogin(username, password)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("login failed")
	}

	if configFile != "" {
		//Check if file exists
		if _, err := os.Stat(configFile); err == nil {
			fmt.Printf("File %s exists, do you want to overwrite? [y/N]: ", configFile)
			if !cli.AskForConfirmation(fmt.Sprintf("File %s exists, do you want to overwrite ? ", configFile)) {
				return fmt.Errorf("aborted")
			}
		}

		tomlConf := config{
			Host: url,
			InsecureSkipVerifyTLS: insecureSkipVerifyTLS,
		}
		var buf = new(bytes.Buffer)
		e := toml.NewEncoder(buf)
		if err := e.Encode(tomlConf); err != nil {
			return err
		}
		if err := ioutil.WriteFile(configFile, buf.Bytes(), os.FileMode(0644)); err != nil {
			return err
		}
	}

	if err := keychain.StoreSecret(url, username, token); err != nil {
		return err
	}

	fmt.Println("Login successfull")

	return nil
}
