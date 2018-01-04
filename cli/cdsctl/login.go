package main

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"regexp"
	"runtime"

	"github.com/howeyc/gopass"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/cdsclient"
)

var loginCmd = cli.Command{
	Name:  "login",
	Short: "Login to CDS",
	Flags: []cli.Flag{
		{
			Name:      "api-url",
			ShortHand: "H",
			Usage:     "CDS API URL",
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
	url := v.GetString("api-url")
	username := v.GetString("username")
	password := v.GetString("password")
	env := v.GetBool("env")

	if env &&
		(url == "" || username == "" || password == "") {
		return fmt.Errorf("Please set flags to use --env option")
	}

	if !env {
		fmt.Println("CDS API URL:", url)
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
		fmt.Printf("export CDS_API_URL=%s\n", url)
		fmt.Printf("export CDS_USER=%s\n", username)
		fmt.Printf("export CDS_TOKEN=%s\n", token)
		fmt.Println("# Run this command to configure your shell:")
		fmt.Println(`# eval $(cds login -H API_URL -u USERNAME -p PASSWORD --env)`)
		return nil
	}

	if configFile == "" {
		homedir := userHomeDir()
		configFile = path.Join(homedir, ".cdsrc")
		fmt.Printf("You didn't specify config file location; %s will be used.\n", configFile)
	}

	var errfi error
	var fi *os.File

	//Check if file exists
	if _, err := os.Stat(configFile); err == nil {
		if !cli.AskForConfirmation(fmt.Sprintf("File %s exists, do you want to overwrite?", configFile)) {
			return fmt.Errorf("aborted")
		}
		fi, errfi = os.OpenFile(configFile, os.O_RDWR, os.FileMode(0600))
	} else {
		fi, errfi = os.Create(configFile)
	}

	if errfi != nil {
		return errfi
	}

	tomlConf := config{
		Host: url,
		InsecureSkipVerifyTLS: insecureSkipVerifyTLS,
		User:  username,
		Token: token,
	}

	defer fi.Close()

	if err := storeSecret(fi, &tomlConf); err != nil {
		return err
	}

	fmt.Println("Login successful")
	return nil
}
