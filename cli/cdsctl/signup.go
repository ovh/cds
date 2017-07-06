package main

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/howeyc/gopass"
	"github.com/ovh/cds/cli"
)

var signupCmd = cli.Command{
	Name:  "signup",
	Short: "Signup on CDS",
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
	url := v["host"]
	username := v["username"]
	fullname := v["fullname"]
	email := v["email"]

	fmt.Println("CDS API Url:", url)

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

	if err := client.UserSignup(username, fullname, email, "cdscli"); err != nil {
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

	return doLogin(client.APIURL(), username, password)
}
