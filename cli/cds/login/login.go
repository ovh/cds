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
	defaultPassword string
)

func init() {
	Cmd.Flags().StringVarP(&defaultEndPoint, "host", "", "", "CDS API URL")
	Cmd.Flags().StringVarP(&defaultUser, "user", "", "", "CDS User")
	Cmd.Flags().StringVarP(&defaultPassword, "password", "", "", "CDS Password")
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

	//Check if file exists
	if _, err := os.Stat(sdk.CDSConfigFile); err == nil {
		fmt.Printf("File %s exists, do you want to overwrite? [y/N]: ", sdk.CDSConfigFile)
		overwrite := readline()
		if overwrite != "y" && overwrite != "Y" {
			fmt.Println("Aborted")
			return
		}
	}

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

	//Take the password from flags or ask for on command line
	if defaultUser == "" {
		//Ask for the password
		fmt.Printf("Password: ")
		b, err := gopass.GetPasswd()
		conf.Password = string(b)
		if err != nil {
			sdk.Exit("Error: wrong usage (%s)\n", err)
		}
	} else {
		fmt.Printf("Password: ******** \n")
		conf.Password = defaultPassword
	}

	//Create the config directory
	if err := os.Mkdir(filepath.Dir(sdk.CDSConfigFile), 0700); err != nil && !os.IsExist(err) {
		sdk.Exit("Error: Cannot create config folder (%s)\n", err)
	}

	//Configure sdk
	sdk.Options(conf.Host, "", "", "")

	//Login
	loginOK, res, err := sdk.LoginUser(conf.User, conf.Password)
	if !loginOK {
		if err != nil {
			sdk.Exit("Error: Login failed (%s)\n", err)
		}
	}

	//Store result in conf object
	if res.Token != "" {
		conf.Token = res.Token
		conf.Password = ""
	} else {
		conf.Token = ""
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
