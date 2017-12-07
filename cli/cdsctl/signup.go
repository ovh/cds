package main

import (
	"fmt"
	"os"
	"reflect"
	"regexp"

	"github.com/howeyc/gopass"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/cdsclient"
)

var signupCmd = cli.Command{
	Name:  "signup",
	Short: "Signup on CDS",
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
			Name:  "username",
			Usage: "CDS Username",
			Kind:  reflect.String,
		}, {
			Name:  "fullname",
			Usage: "Fullname",
			Kind:  reflect.String,
		}, {
			Name:  "email",
			Usage: "Email",
			Kind:  reflect.String,
		},
	},
}

func signupRun(v cli.Values) error {
	url := v.GetString("api-url")
	username := v.GetString("username")
	fullname := v.GetString("fullname")
	email := v.GetString("email")

	fmt.Println("CDS API URL:", url)

	//Take the user from flags or ask for on command line
	if username == "" {
		fmt.Printf("Username: ")
		username = cli.ReadLine()
	} else {
		fmt.Println("Username:", username)
	}

	//Take fullname user from flags or ask for on command line
	if fullname == "" {
		fmt.Printf("Fullname: ")
		fullname = cli.ReadLine()
	} else {
		fmt.Println("Fullname:", fullname)
	}

	//Take email user from flags or ask for on command line
	if email == "" {
		fmt.Printf("Email: ")
		email = cli.ReadLine()
	} else {
		fmt.Println("Email:", email)
	}

	conf := cdsclient.Config{
		Host:    url,
		Verbose: os.Getenv("CDS_VERBOSE") == "true",
	}
	client = cdsclient.New(conf)

	if err := client.UserSignup(username, fullname, email, "your username:%s, your confirmation code:%s"); err != nil {
		return err
	}

	fmt.Println("Please check your mail box to activate your account...")

	return doConfirm(username)
}

func doConfirm(username string) error {
	fmt.Printf("Enter your verification code: ")
	b, err := gopass.GetPasswd()
	if err != nil {
		return err
	}
	token := string(b)

	ok, password, err := client.UserConfirm(username, token)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("verification failed")
	}

	fmt.Println("All is fine. Here is your new password:")
	fmt.Println(password)

	return doLogin(client.APIURL(), username, password, false)
}
