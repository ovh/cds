package main

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	survey "gopkg.in/AlecAivazis/survey.v1"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

var loginCmd = cli.Command{
	Name:    "login",
	Short:   "Login to CDS",
	Example: `Use it with 'eval' and 'env' flag to set environment variable: eval $(cds login -H API_URL -u USERNAME -p PASSWORD --env)`,
	Flags: []cli.Flag{
		{
			Name:      "api-url",
			ShortHand: "H",
			Usage:     "Url to your CDS api.",
			IsValid: func(s string) bool {
				match, _ := regexp.MatchString(`http[s]?:\/\/(.*)`, s)
				return match
			},
		},
		{
			Name:  "consumer-type",
			Usage: "CDS auth consumer type (default: local).",
		},
		{
			Name:  "env",
			Usage: "Display the commands to set up the environment for the cds client.",
			Type:  cli.FlagBool,
		},
		{
			Name:      "username",
			ShortHand: "u",
		},
		{
			Name:      "password",
			ShortHand: "p",
		},
	},
}

func login() *cobra.Command {
	return cli.NewCommand(loginCmd, loginRun, nil, cli.CommandWithoutExtraFlags)
}

func loginRun(v cli.Values) error {
	env := v.GetBool("env")
	if env && sdk.GOOS == "windows" {
		return fmt.Errorf("Env option is not supported on windows yet")
	}

	apiURL := v.GetString("api-url")
	if apiURL == "" {
		return fmt.Errorf("Please set api url")
	}
	if strings.HasSuffix(apiURL, "/") {
		return fmt.Errorf("Invalid given api url, remove trailing '/'")
	}

	consumerType := sdk.AuthConsumerType(v.GetString("consumer-type"))
	if !consumerType.IsValid() {
		return fmt.Errorf("Invalid given consumer type")
	}

	switch consumerType {
	case sdk.ConsumerLocal:
		return loginRunLocal(v)
	case sdk.ConsumerBuiltin:
		return loginRunBuiltin(v)
	default:
		return loginRunExternal(v)
	}
}

func loginRunLocal(v cli.Values) error {
	apiURL := v.GetString("api-url")

	username := v.GetString("username")
	password := v.GetString("password")
	env := v.GetBool("env")
	if env && (username == "" || password == "") {
		return fmt.Errorf("Please set username and password flags to use --env option")
	}

	if username == "" {
		username = cli.AskValueChoice("Username")
	} else if !env {
		fmt.Printf("Username: %s", username)
	}
	if password == "" {
		if err := survey.AskOne(&survey.Password{Message: "Password"}, &password, nil); err != nil {
			return err
		}
	} else if !env {
		fmt.Println("Password: ********")
	}

	conf := cdsclient.Config{
		Host:    apiURL,
		Verbose: os.Getenv("CDS_VERBOSE") == "true",
	}

	client = cdsclient.New(conf)

	res, err := client.AuthConsumerSignin(sdk.ConsumerLocal, sdk.AuthConsumerSigninRequest{
		"username": username,
		"password": password,
	})
	if err != nil {
		return fmt.Errorf("cannot signin: %v", err)
	}

	return doAfterLogin(apiURL, res.User.Username, res.Token, env, v.GetBool("insecure"))
}

func loginRunBuiltin(v cli.Values) error {
	apiURL := v.GetString("api-url")

	signinToken := v.GetString("signin-token")
	env := v.GetBool("env")
	if env && signinToken == "" {
		return fmt.Errorf("Please set signin-token flag to use --env option")
	}

	if signinToken == "" {
		if err := survey.AskOne(&survey.Password{Message: "Sign in token"}, &signinToken, nil); err != nil {
			return err
		}
	} else if !env {
		fmt.Println("Sign in token: ********")
	}

	conf := cdsclient.Config{
		Host:    apiURL,
		Verbose: os.Getenv("CDS_VERBOSE") == "true",
	}

	client = cdsclient.New(conf)

	res, err := client.AuthConsumerSignin(sdk.ConsumerBuiltin, sdk.AuthConsumerSigninRequest{
		"token": signinToken,
	})
	if err != nil {
		return fmt.Errorf("cannot signin: %s", err)
	}

	return doAfterLogin(apiURL, res.User.Username, res.Token, env, v.GetBool("insecure"))
}

func loginRunExternal(v cli.Values) error {
	apiURL := v.GetString("api-url")
	consumerType := sdk.AuthConsumerType(v.GetString("consumer-type"))

	conf := cdsclient.Config{
		Host:    apiURL,
		Verbose: os.Getenv("CDS_VERBOSE") == "true",
	}

	client = cdsclient.New(conf)

	askSigninURI, err := url.Parse(apiURL + "/auth/consumer/" + string(consumerType) + "/askSignin?origin=cdsctl")
	if err != nil {
		return fmt.Errorf("cannot parse given api uri: %v", err)
	}

	fmt.Println("cdsctl: Opening the browser to login or control-c to abort")
	fmt.Println(" >\tWarning: If browser does not open, visit")
	fmt.Println(" >\t" + cli.Green("%s", askSigninURI.String()))
	browser.OpenURL(askSigninURI.String()) // nolint

	token := cli.AskValueChoice("Enter 'token' value:")
	splittedToken := strings.Split(token, ":")
	if len(splittedToken) != 2 {
		return fmt.Errorf("invalid given 'token' value")
	}
	state, code := splittedToken[0], splittedToken[1]

	res, err := client.AuthConsumerSignin(consumerType, sdk.AuthConsumerSigninRequest{
		"state": state,
		"code":  code,
	})
	if err != nil {
		return fmt.Errorf("cannot signin: %v", err)
	}

	return doAfterLogin(apiURL, res.User.Username, res.Token, v.GetBool("env"), v.GetBool("insecure"))
}

func doAfterLogin(url, username, token string, env bool, insecureSkipVerifyTLS bool) error {
	if insecureSkipVerifyTLS {
		fmt.Println("Using insecure TLS connection...")
	}

	if env {
		fmt.Printf("export CDS_API_URL=%s\n", url)
		fmt.Printf("export CDS_USER=%s\n", username)
		fmt.Printf("export CDS_TOKEN=%s\n", token)
		fmt.Println("# Run this command to configure your shell:")
		fmt.Println(`# eval $(cds login -H API_URL -u USERNAME -p PASSWORD --env)`)
		return nil
	} else {
		fmt.Println("cdsctl: Login successful")
		fmt.Println("cdsctl: Logged in as", username)
	}

	if configFile == "" {
		homedir := userHomeDir()
		configFile = path.Join(homedir, ".cdsrc")
		fmt.Printf("cdsctl: You didn't specify config file location; %s will be used.\n", configFile)
	}

	var errfi error
	var fi *os.File

	//Check if file exists
	if _, err := os.Stat(configFile); err == nil {
		if !cli.AskForConfirmation(fmt.Sprintf("File %s exists, do you want to overwrite?", configFile)) {
			return fmt.Errorf("aborted")
		}
		if errre := os.Remove(configFile); errre != nil {
			return fmt.Errorf("Error while removing old file %s: %s", configFile, errre)
		}
	}

	fi, errfi = os.Create(configFile)
	if errfi != nil {
		return fmt.Errorf("Error while creating file %s: %s", configFile, errfi)
	}

	tomlConf := config{
		Host:                  url,
		InsecureSkipVerifyTLS: insecureSkipVerifyTLS,
		User:                  username,
		Token:                 token,
	}

	defer fi.Close()

	if err := storeSecret(fi, &tomlConf); err != nil {
		return err
	}

	if errm := fi.Chmod(os.FileMode(0600)); errm != nil {
		return fmt.Errorf("Error while chmod 600 file %s: %s", configFile, errm)
	}

	return nil
}
