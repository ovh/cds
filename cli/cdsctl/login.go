package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/howeyc/gopass"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/jws"
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
		}, {
			Name:      "username",
			ShortHand: "u",
			Usage:     "CDS Username",
		}, {
			Name:      "password",
			ShortHand: "p",
			Usage:     "CDS Password",
		}, {
			Name:  "env",
			Usage: "Display the commands to set up the environment for the cds client",
			Type:  cli.FlagBool,
		},
	},
}

func login() *cobra.Command {
	return cli.NewCommand(loginCmd, loginRun, nil, cli.CommandWithoutExtraFlags)
}

func loginExperimental() *cobra.Command {
	c := cli.NewCommand(loginCmd, loginJWTRun, nil, cli.CommandWithoutExtraFlags)
	c.Use = "x" + c.Use
	c.Short = c.Short + " [EXPERIMENTAL]"
	return c
}

func loginJWTRun(v cli.Values) error {
	var apiURL = v.GetString("api-url")
	if strings.HasSuffix(apiURL, "/") {
		fmt.Fprintf(os.Stderr, "Invalid URL. Remove trailing '/'\n")
	}

	conf := cdsclient.Config{
		Host:    apiURL,
		Verbose: os.Getenv("CDS_VERBOSE") == "true",
	}

	client = cdsclient.New(conf)
	config, err := client.ConfigUser()
	if err != nil {
		return fmt.Errorf("unable get CDS UI URL: %v", err)
	}

	// prepare an accessTokenRequest with a short
	var accessTokenRequest = sdk.AccessTokenRequest{
		Description:           "cdsctl-login-" + time.Now().Format(time.RFC3339),
		Origin:                "cdsctl",
		ExpirationDelaySecond: 10 * 60, // ten minutes
	}

	privateKey, err := jws.NewRandomRSAKey()
	if err != nil {
		return fmt.Errorf("unable to prepare private key: %v", err)
	}

	pubKey, err := jws.ExportPublicKey(privateKey)
	if err != nil {
		return fmt.Errorf("unable to prepare publick key: %v", err)
	}

	signer, err := jws.NewSigner(privateKey)
	if err != nil {
		return fmt.Errorf("unable to prepare JWS signer: %v", err)
	}

	content, err := jws.Sign(signer, accessTokenRequest)
	if err != nil {
		return fmt.Errorf("unable to sign access token request: %v", err)
	}

	uiURL, err := url.Parse(config[sdk.ConfigURLUIKey])
	if err != nil {
		return fmt.Errorf("unable to parse UI URL %s: %v", config[sdk.ConfigURLUIKey], err)
	}

	uiURL.Path = "/account/login"
	q := uiURL.Query()
	q.Add("request", content)
	uiURL.RawQuery = q.Encode()

	fmt.Println("cdsctl: Opening the browser to login or control-c to abort")
	fmt.Println(" >\tWarning: If browser does not open, vist")
	fmt.Println(" >\t" + cli.Green("%s", uiURL.String()))
	browser.OpenURL(uiURL.String()) // nolint
	// wait for something
	fmt.Println("cdsctl: Waiting for login...")

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	accessToken, jwt, err := client.UserLoginCallback(ctx, content, pubKey)
	if err != nil {
		return fmt.Errorf("unable to get login data: %v", err)
	}

	// Now use the new JWT token to make the call
	conf = cdsclient.Config{
		Host:        apiURL,
		Verbose:     os.Getenv("CDS_VERBOSE") == "true",
		User:        accessToken.User.Username,
		AccessToken: jwt,
	}
	client = cdsclient.New(conf)

	u, err := client.UserGet(accessToken.User.Username)
	if err != nil {
		return fmt.Errorf("unable to get user %s: %v", accessToken.User.Username, err)
	}

	ids := sdk.GroupsToIDs(u.Groups)

	// Create a new token with a long expiration delay
	newAccessToken, jwt, err := client.AccessTokenCreate(sdk.AccessTokenRequest{
		Description:           "cdsctl-login-" + time.Now().Format(time.RFC3339),
		Origin:                "cdsctl",
		ExpirationDelaySecond: 604800, // one week
		GroupsIDs:             ids,
	})
	if err != nil {
		return fmt.Errorf("unable to create access token: %v", err)
	}

	// Delete the first token
	if err := client.AccessTokenDelete(accessToken.ID); err != nil {
		return fmt.Errorf("unable to delete login access token %s: %v", accessToken.Description, err)
	}

	fmt.Println("cdsctl: Login successful")
	fmt.Println("cdsctl: Logged in as", newAccessToken.User.Username)

	return doAfterLogin(apiURL, newAccessToken.User.Username, jwt, v.GetBool("env"), v.GetBool("insecure"))
}

func loginRun(v cli.Values) error {
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
		if sdk.ErrorIs(err, sdk.ErrInvalidUser) {
			return fmt.Errorf(sdk.ErrInvalidUser.Error())
		}
		return fmt.Errorf("Please check CDS API URL")
	}
	if !ok {
		return fmt.Errorf("login failed")
	}

	return doAfterLogin(url, username, token, env, insecureSkipVerifyTLS)
}

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
