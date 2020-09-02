package main

import (
	"fmt"
	"os"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/spf13/cobra"
)

func reset() *cobra.Command {
	return cli.NewCommand(resetCmd, resetFunc, cli.SubCommands{resetConfirm()}, cli.CommandWithoutExtraFlags)
}

var resetCmd = cli.Command{
	Name:  "reset-password",
	Short: "Reset CDS user password",
	Flags: []cli.Flag{
		{
			Name: "email",
		},
		{
			Name:      "api-url",
			ShortHand: "H",
			Usage:     "Url to your CDS api.",
		},
	},
}

func resetFunc(v cli.Values) error {
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

	email := v.GetString("email")
	if email == "" && !noInteractive {
		email = cli.AskValue("Email")
	}

	req := sdk.AuthConsumerSigninRequest{
		"email": email,
	}
	if err := client.AuthConsumerLocalAskResetPassword(req); err != nil {
		return err
	}

	fmt.Println("Reset successful. Instuctions have been sent to your email address.")
	return nil
}

func resetConfirm() *cobra.Command {
	return cli.NewCommand(resetConfirmCmd, resetConfirmFunc, nil, cli.CommandWithoutExtraFlags)
}

var resetConfirmCmd = cli.Command{
	Name: "confirm",
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
		}, {
			Name:      "password",
			ShortHand: "p",
		},
	},
}

func resetConfirmFunc(v cli.Values) error {
	token := v.GetString("token")
	if token == "" {
		return fmt.Errorf("Invalid given token")
	}

	noInteractive := v.GetBool("no-interactive")

	apiURL, err := getAPIURL(v)
	if err != nil {
		return err
	}

	client := cdsclient.New(cdsclient.Config{
		Host:                  apiURL,
		Verbose:               os.Getenv("CDS_VERBOSE") == "true" || v.GetBool("verbose"),
		InsecureSkipVerifyTLS: os.Getenv("CDS_INSECURE") == "true" || v.GetBool("insecure"),
	})

	password := v.GetString("password")
	if password == "" && !noInteractive {
		password = cli.AskPassword("Password")
	}

	if password == "" {
		return fmt.Errorf("Invalid given password")
	}

	signupresponse, err := client.AuthConsumerLocalResetPassword(token, password)
	if err != nil {
		return err
	}

	return doAfterLogin(client, v, signupresponse.APIURL, sdk.ConsumerLocal, signupresponse)
}
