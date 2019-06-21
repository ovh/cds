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

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
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
		},
		{
			Name:  "consumer-type",
			Usage: "CDS auth consumer type (default: local)",
		},
		{
			Name:  "env",
			Usage: "Display the commands to set up the environment for the cds client",
			Type:  cli.FlagBool,
		},
	},
}

func login() *cobra.Command {
	return cli.NewCommand(loginCmd, loginRun, nil, cli.CommandWithoutExtraFlags)
}

func loginRun(v cli.Values) error {
	var apiURL = v.GetString("api-url")
	if strings.HasSuffix(apiURL, "/") {
		fmt.Fprintf(os.Stderr, "Invalid URL. Remove trailing '/'\n")
	}

	consumerType := sdk.AuthConsumerType(v.GetString("consumer-type"))
	if !consumerType.IsValid() {
		return fmt.Errorf("invalid given consumer type")
	}

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

	state := cli.AskValueChoice("Enter 'state' value:")
	code := cli.AskValueChoice("Enter 'code' value:")

	res, err := client.AuthConsumerSignin(consumerType, sdk.AuthConsumerSigninRequest{
		"state": state,
		"code":  code,
	})
	if err != nil {
		return fmt.Errorf("cannot signin: %v", err)
	}

	fmt.Println("cdsctl: Login successful")
	fmt.Println("cdsctl: Logged in as", res.User.Username)

	return doAfterLogin(apiURL, res.User.Username, res.Token, v.GetBool("env"), v.GetBool("insecure"))
}

/*func loginRun(v cli.Values) error {
	url := v.GetString("api-url")
	username := v.GetString("username")
	password := v.GetString("password")
	env := v.GetBool("env")
	insecureSkipVerifyTLS := v.GetBool("insecure")

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

	return doLogin(url, username, password, env, insecureSkipVerifyTLS)
}

func doLogin(url, username, password string, env, insecureSkipVerifyTLS bool) error {
	conf := cdsclient.Config{
		Host:    url,
		Verbose: os.Getenv("CDS_VERBOSE") == "true",
	}

	client = cdsclient.New(conf)
	ok, token, err := client.UserLogin(username, password)
	if err != nil {
		if conf.Verbose {
			fmt.Fprintf(os.Stderr, "error:%s\n", err)
		}
		if strings.HasSuffix(url, "/") {
			fmt.Fprintf(os.Stderr, "Invalid URL. Remove trailing '/'\n")
		}
		return fmt.Errorf("Please check CDS API URL")
	}
	if !ok {
		return fmt.Errorf("login failed")
	}

	return doAfterLogin(url, username, token, env, insecureSkipVerifyTLS)
}*/

func doAfterLogin(url, username, token string, env bool, insecureSkipVerifyTLS bool) error {
	if insecureSkipVerifyTLS {
		fmt.Println("Using insecure TLS connection...")
	}

	if env && sdk.GOOS == "windows" {
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
