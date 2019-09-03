package main

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/ovh/cds/cli/cdsctl/internal"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"

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
		},
		{
			Name:      "driver",
			Usage:     "An enabled auth driver to login with. This should be local, github, gitlab, ldap, builtin or corporate-sso",
			ShortHand: "d",
		},
		{
			Name:  "env",
			Usage: "Display the commands to set up the environment for the cds client.",
			Type:  cli.FlagBool,
		},
		{
			Name:      "username",
			Usage:     "The identifier name needed by selected auth driver",
			ShortHand: "u",
		},
		{
			Name:      "password",
			ShortHand: "p",
		},
		{
			Name:      "init-token",
			Usage:     "A CDS init token that can be used for first connection",
			ShortHand: "i",
		},
		{
			Name:  "context-name",
			Usage: "A cdsctl context name",
		},
	},
}

func login() *cobra.Command {
	return cli.NewCommand(loginCmd, loginRun, nil, cli.CommandWithoutExtraFlags)
}

func loginRun(v cli.Values) error {
	noInteractive := v.GetBool("no-interactive")

	// Env param is not valid for windows users
	if v.GetBool("env") && sdk.GOOS == "windows" {
		return fmt.Errorf("Env option is not supported on windows yet")
	}

	// Checks that an URL is given
	apiURL, err := getAPIURL(v)
	if err != nil {
		return err
	}

	// Load all drivers from given CDS instance
	client := cdsclient.New(cdsclient.Config{
		Host:    apiURL,
		Verbose: os.Getenv("CDS_VERBOSE") == "true",
	})
	drivers, err := client.AuthDriverList()
	if err != nil {
		return fmt.Errorf("Cannot list auth drivers: %v", err)
	}
	if len(drivers.Drivers) == 0 {
		return fmt.Errorf("No authentication driver configured")
	}

	// Check driver type validity or ask for one
	driverType := sdk.AuthConsumerType(v.GetString("driver"))
	if driverType == "" && !v.GetBool("no-interactive") {
		opts := make([]string, len(drivers.Drivers))
		for i, d := range drivers.Drivers {
			opts[i] = string(d.Type)
		}
		selected := cli.AskChoice("Select the type of driver that will be used to login", opts...)
		driverType = drivers.Drivers[selected].Type
	}
	if !driverType.IsValid() || !drivers.Drivers.ExistsConsumerType(driverType) {
		return fmt.Errorf("Invalid given consumer type")
	}

	var req sdk.AuthConsumerSigninRequest
	switch driverType {
	case sdk.ConsumerLocal:
		req, err = loginRunLocal(v)
	case sdk.ConsumerLDAP:
		req, err = loginRunLDAP(v)
	case sdk.ConsumerBuiltin:
		req, err = loginRunBuiltin(v)
	default:
		if noInteractive {
			return fmt.Errorf("Cannot signin with %s driver in no interactive mode", driverType)
		}
		req, err = loginRunExternal(v, driverType)
	}
	if err != nil {
		return err
	}

	// For first connection ask for an optional init token
	if drivers.IsFirstConnection {
		initToken := v.GetString("init-token")
		if initToken != "" {
			req["init_token"] = initToken
		}
	}

	// Send signin request
	res, err := client.AuthConsumerSignin(driverType, req)
	if err != nil {
		return fmt.Errorf("cannot signin: %v", err)
	}

	return doAfterLogin(v, apiURL, res)
}

func loginRunLocal(v cli.Values) (sdk.AuthConsumerSigninRequest, error) {
	req := sdk.AuthConsumerSigninRequest{
		"username": v.GetString("username"),
		"password": v.GetString("password"),
	}

	noInteractive := v.GetBool("no-interactive")

	if req["username"] == "" && !noInteractive {
		req["username"] = cli.AskValue("Username")
	}
	if req["password"] == "" && !noInteractive {
		req["password"] = cli.AskPassword("Password")
	}
	if req["username"] == "" || req["password"] == "" {
		return req, fmt.Errorf("Invalid given username or password")
	}

	return req, nil
}

func loginRunLDAP(v cli.Values) (sdk.AuthConsumerSigninRequest, error) {
	req := sdk.AuthConsumerSigninRequest{
		"bind":     v.GetString("username"),
		"password": v.GetString("password"),
	}

	noInteractive := v.GetBool("no-interactive")

	if req["bind"] == "" && !noInteractive {
		req["bind"] = cli.AskValue("LDAP bind")
	}
	if req["password"] == "" && !noInteractive {
		req["password"] = cli.AskPassword("Password")
	}
	if req["bind"] == "" || req["password"] == "" {
		return req, fmt.Errorf("Invalid given LDAP bind or password")
	}

	return req, nil
}

func loginRunBuiltin(v cli.Values) (sdk.AuthConsumerSigninRequest, error) {
	req := sdk.AuthConsumerSigninRequest{
		"token": v.GetString("signin-token"),
	}

	if req["token"] == "" && !v.GetBool("no-interactive") {
		req["token"] = cli.AskPassword("Sign in token")
	}
	if req["token"] == "" {
		return req, fmt.Errorf("Invalid given signin token")
	}

	return req, nil
}

func loginRunExternal(v cli.Values, consumerType sdk.AuthConsumerType) (sdk.AuthConsumerSigninRequest, error) {
	req := sdk.AuthConsumerSigninRequest{}

	apiURL := v.GetString("api-url")

	client := cdsclient.New(cdsclient.Config{
		Host:    apiURL,
		Verbose: v.GetBool("verbose"),
	})
	config, err := client.ConfigUser()
	if err != nil {
		return req, err
	}

	askSigninURI, err := url.Parse(config.URLUI + "/auth/ask-signin/" + string(consumerType) + "?origin=cdsctl")
	if err != nil {
		return req, fmt.Errorf("cannot parse given api uri: %v", err)
	}

	fmt.Println("cdsctl: Opening the browser to login or control-c to abort")
	fmt.Println(" >\tWarning: If browser does not open, visit")
	fmt.Println(" >\t" + cli.Green("%s", askSigninURI.String()))
	browser.OpenURL(askSigninURI.String()) // nolint

	token := cli.AskPassword("Token")
	splittedToken := strings.Split(token, ":")
	if len(splittedToken) != 2 {
		return req, fmt.Errorf("Invalid given token")
	}
	req["state"], req["code"] = splittedToken[0], splittedToken[1]

	return req, nil
}

func doAfterLogin(v cli.Values, apiURL string, res sdk.AuthConsumerSigninResponse) error {
	noInteractive := v.GetBool("no-interactive")
	insecureSkipVerifyTLS := v.GetBool("insecure")
	if insecureSkipVerifyTLS {
		fmt.Println("Using insecure TLS connection...")
	}

	env := v.GetBool("env")
	if env {
		fmt.Printf("export CDS_API_URL=%s\n", apiURL)
		fmt.Printf("export CDS_USER=%s\n", res.User.Username)
		fmt.Printf("export CDS_SESSION_TOKEN=%s\n", res.Token)
		return nil
	}

	fmt.Println("cdsctl: Login successful")
	fmt.Println("cdsctl: Logged in as", res.User.Username)

	// If no config file path is given, use a default one $HOME/.cdsrc
	configFile := v.GetString("file")
	if configFile == "" {
		homedir := userHomeDir()
		configFile = path.Join(homedir, ".cdsrc")
		fmt.Printf("cdsctl: You didn't specify config file location; %s will be used.\n", configFile)
	}

	contextName := v.GetString("context-name")
	// create file if not exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		fi, err := os.Create(configFile)
		if err != nil {
			return fmt.Errorf("Error while creating file %s: %v", configFile, err)
		}
		fi.Close()
		if !noInteractive {
			contextName = cli.AskValue("Enter a context name for this login (default):")
		}
	} else {
		fi, err := os.Open(configFile)
		if err != nil {
			return fmt.Errorf("Error while opening file %s: %v", configFile, err)
		}
		cdsConfigFile, err := internal.GetConfigFile(fi)
		if err != nil {
			fmt.Printf("Error while reading config file %s: %v\n", configFile, err)
		} else if !noInteractive {
			opts := []string{}
			for _, c := range cdsConfigFile.Contexts {
				line := c.Context
				if c.Context == cdsConfigFile.Current {
					line = fmt.Sprintf("%s - current", line)
				}
				opts = append(opts, line)
			}
			other := "Enter another name"
			opts = append(opts, other)

			selected := cli.AskChoice("Choose a context for this login", opts...)
			if opts[selected] == other {
				contextName = cli.AskValue("Enter a context name for this login (default):")
			} else {
				contextName = strings.TrimPrefix(opts[selected], " - current")
			}
		}
		fi.Close()
	}

	fi, err := os.OpenFile(configFile, os.O_RDONLY, 0600)
	if err != nil {
		return fmt.Errorf("Error while opening file %s: %v", configFile, err)
	}
	defer fi.Close()

	if contextName == "" {
		contextName = "default"
	}

	cdsContext := internal.CDSContext{
		Context:               contextName,
		Host:                  apiURL,
		InsecureSkipVerifyTLS: insecureSkipVerifyTLS,
		SessionToken:          res.Token,
	}

	wdata := &bytes.Buffer{}
	if err := internal.StoreContext(fi, wdata, cdsContext); err != nil {
		return err
	}

	if err := fi.Close(); err != nil {
		return fmt.Errorf("Error while closing file %s: %v", configFile, err)
	}
	if err := writeConfigFile(configFile, wdata); err != nil {
		return err
	}
	return nil
}

func writeConfigFile(configFile string, content *bytes.Buffer) error {
	if errre := os.Remove(configFile); errre != nil {
		return fmt.Errorf("Error while removing old file %s: %s", configFile, errre)
	}
	fi, err := os.OpenFile(configFile, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("Error while opening file %s: %v", configFile, err)
	}
	defer fi.Close()
	if _, err := fi.Write(content.Bytes()); err != nil {
		return fmt.Errorf("Error while writing file %s: %v", configFile, err)
	}
	return nil
}
