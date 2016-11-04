package login

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/howeyc/gopass"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

var (
	defaultEndPoint string
	defaultUser     string
)

func init() {
	Cmd.Flags().StringVarP(&defaultEndPoint, "host", "", "", "CDS API URL")
	Cmd.Flags().StringVarP(&defaultUser, "user", "", "", "CDS User")
}

// Cmd login
var Cmd = &cobra.Command{
	Use:   "login",
	Short: "Ease up creation of config file",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		runLogin()
	},
}

type config struct {
	User     string `json:"user"`
	Password string `json:"password,omitempty"`
	Token    string `json:"token,omitempty"`
	Host     string `json:"host"`
}

func runLogin() {
	conf := config{}

	//Take the endpoint from flags or ask for on command line
	if defaultEndPoint == "" {
		fmt.Printf("CDS endpoint: ")
		conf.Host = readline()
	} else {
		fmt.Printf("CDS endpoint: %s\n", defaultEndPoint)
		conf.Host = defaultEndPoint
	}

	//Take the user from flags or ask for on command line
	if defaultUser == "" {
		fmt.Printf("Username: ")
		conf.User = readline()
	} else {
		fmt.Printf("Username: %s\n", defaultUser)
		conf.User = defaultUser
	}

	//Ask for the password
	fmt.Printf("Password: ")
	password, err := gopass.GetPasswd()
	if err != nil {
		sdk.Exit("Error: wrong usage (%s)\n", err)
	}

	//Create the config directory
	if err := os.Mkdir(filepath.Base(sdk.CDSConfigFile), 0700); err != nil && !os.IsExist(err) {
		sdk.Exit("Error: Cannot create config folder (%s)\n", err)
	}

	//Configure sdk
	sdk.Options(conf.Host, "", "", "")

	//Login
	loginOK, res, err := sdk.LoginUser(conf.User, string(password))
	if !loginOK {
		if err != nil {
			sdk.Exit("Error: Login failed (%s)\n", err)
		}
	}

	//Store result in conf object
	if res.Token != "" {
		conf.Token = res.Token
	} else {
		conf.Password = string(password)
	}

	//Write conf in file
	data, err := json.MarshalIndent(conf, " ", " ")
	if err != nil {
		sdk.Exit("Error: Cannot create config file (%s)\n", err)
	}
	if err := ioutil.WriteFile(sdk.CDSConfigFile, data, 0640); err != nil {
		sdk.Exit("Error: Cannot write config file (%s)\n", err)
	}

	fmt.Printf("Done\n")
}

func readline() string {
	var all string
	var line []byte
	var err error

	hasMoreInLine := true
	bio := bufio.NewReader(os.Stdin)

	for hasMoreInLine {
		line, hasMoreInLine, err = bio.ReadLine()
		if err != nil {
			sdk.Exit("Error: cannot read from stdin (%s)\n", err)
		}
		all += string(line)
	}

	return strings.Replace(all, "\n", "", -1)
}
