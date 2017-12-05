package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"reflect"
	"regexp"
	"runtime"

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
			Kind: reflect.String,
		}, {
			Name:      "username",
			ShortHand: "u",
			Usage:     "CDS Username",
			Kind:      reflect.String,
		}, {
			Name:      "password",
			ShortHand: "p",
			Usage:     "CDS Password",
			Kind:      reflect.String,
		}, {
			Name:  "env",
			Usage: "Display the commands to set up the environment for the cds client",
			Kind:  reflect.Bool,
		},
	},
}

func loginRun(v cli.Values) error {
	url := v.GetString("host")
	username := v.GetString("username")
	password := v.GetString("password")
	env := v.GetBool("env")

	if env &&
		(url == "" || username == "" || password == "") {
		return fmt.Errorf("Please set flags to use --env option")
	}

	if !env {
		fmt.Println("CDS API Url:", url)
	}

	//Take the user from flags or ask for on command line
	if username == "" {
		fmt.Printf("Username: ")
		username = cli.ReadLine()
	} else if !env {
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
	} else if !env {
		fmt.Println("Password: ********")
	}

	return doLogin(url, username, password, env)
}

func doLogin(url, username, password string, env bool) error {
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

	if env && runtime.GOOS == "windows" {
		fmt.Println("env option is not supported on windows yet")
		os.Exit(1)
	}

	if env {
		fmt.Printf("export CDS_API=%s\n", url)
		fmt.Printf("export CDS_USER=%s\n", username)
		fmt.Printf("export CDS_TOKEN=%s\n", token)
		fmt.Println("# Run this command to configure your shell:")
		fmt.Println(`# eval $(cds login -H HOST -u USERNAME -p PASSWORD --env)`)
		return nil
	}

	if configFile == "" {
		u, err := user.Current()
		if err != nil {
			return err
		}
		configFile = path.Join(u.HomeDir, ".cdsrc")
		fmt.Printf("You didn't specify config file location; %s will be used.\n", configFile)
	}

	//Check if file exists
	if _, err := os.Stat(configFile); err == nil {
		if !cli.AskForConfirmation(fmt.Sprintf("File %s exists, do you want to overwrite?", configFile)) {
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

	if err := keychain.StoreSecret(url, username, token); err != nil {
		return err
	}

	fmt.Println("Login successfull")
	return nil
}
