package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/spf13/cobra"
)

func signup() *cobra.Command {
	return cli.NewCommand(signupCmd, signupFunc, cli.SubCommands{signupVerify()}, cli.CommandWithoutExtraFlags)
}

func signupVerify() *cobra.Command {
	return cli.NewCommand(signupVerifyCmd, signupVerifyFunc, nil, cli.CommandWithoutExtraFlags)
}

var signupCmd = cli.Command{
	Name:  "signup",
	Short: "Signup on CDS",
	Flags: []cli.Flag{
		{
			Name:      "api-url",
			ShortHand: "H",
			Usage:     "Url to your CDS api.",
		},
		{
			Name: "username",
		},
		{
			Name: "fullname",
		},
		{
			Name: "email",
		},
		{
			Name: "password",
		},
	},
}

func getAPIURL(v cli.Values) (string, error) {
	noInteractive := v.GetBool("no-interactive")

	// Checks that an URL is given
	apiURL := v.GetString("api-url")
	if apiURL == "" && !noInteractive {
		apiURL = cli.AskValue("api-url")
	}
	if apiURL == "" {
		return "", fmt.Errorf("Please set api url")
	}
	match, _ := regexp.MatchString(`http[s]?:\/\/(.*)`, apiURL)
	if !match {
		return "", fmt.Errorf("Invalid given api url")
	}
	apiURL = strings.TrimSuffix(apiURL, "/")
	return apiURL, nil
}

func signupFunc(v cli.Values) error {
	noInteractive := v.GetBool("no-interactive")

	apiURL, err := getAPIURL(v)
	if err != nil {
		return err
	}

	// Load all drivers from given CDS instance
	client := cdsclient.New(cdsclient.Config{
		Host:                  apiURL,
		Verbose:               os.Getenv("CDS_VERBOSE") == "true" || v.GetBool("verbose"),
		InsecureSkipVerifyTLS: os.Getenv("CDS_INSECURE") == "true" || v.GetBool("insecure"),
	})
	drivers, err := client.AuthDriverList()
	if err != nil {
		return fmt.Errorf("Cannot list auth drivers: %v", err)
	}
	if len(drivers.Drivers) == 0 {
		return fmt.Errorf("No authentication driver configured")
	}

	var localConsumerDriverEnable bool
	for _, d := range drivers.Drivers {
		if d.Type == sdk.ConsumerLocal {
			localConsumerDriverEnable = true
			break
		}
	}

	if !localConsumerDriverEnable {
		return fmt.Errorf("No authentication driver configured")
	}

	signupRequest, err := loginRunLocal(v)
	if err != nil {
		return err
	}

	fullname := v.GetString("fullname")
	if fullname == "" && !noInteractive {
		fullname = cli.AskValue("Fullname")
	}

	email := v.GetString("email")
	if email == "" && !noInteractive {
		email = cli.AskValue("Email")
	}

	signupRequest["email"] = email
	signupRequest["fullname"] = fullname

	if err := client.AuthConsumerLocalSignup(signupRequest); err != nil {
		return err
	}

	fmt.Println("Signup successful. Instuctions have been sent to your email address.")
	return nil
}

var signupVerifyCmd = cli.Command{
	Name:  "verify",
	Short: "Verify local CDS signup.",
	Long:  "For admin signup INIT_TOKEN environment variable must be set.",
	Args: []cli.Arg{
		{
			Name:       "token",
			AllowEmpty: false,
		},
	},
	Flags: []cli.Flag{
		{
			Name:      "api-url",
			ShortHand: "H",
			Usage:     "Url to your CDS api.",
		},
		{
			Name:  "env",
			Usage: "Display the commands to set up the environment for the cds client.",
			Type:  cli.FlagBool,
		},
	},
}

func signupVerifyFunc(v cli.Values) error {
	apiURL, err := getAPIURL(v)
	if err != nil {
		return err
	}

	client := cdsclient.New(cdsclient.Config{
		Verbose:               os.Getenv("CDS_VERBOSE") == "true" || v.GetBool("verbose"),
		InsecureSkipVerifyTLS: os.Getenv("CDS_INSECURE") == "true" || v.GetBool("insecure"),
		Host:                  apiURL,
	})

	signupresponse, err := client.AuthConsumerLocalSignupVerify(v.GetString("token"),
		os.Getenv("INIT_TOKEN"))
	if err != nil {
		return err
	}

	if apiURL != signupresponse.APIURL {
		fmt.Println("WARNING: The advertised API URL differs from the provided URL")
	}
	return doAfterLogin(client, v, apiURL, sdk.ConsumerLocal, signupresponse)
}
