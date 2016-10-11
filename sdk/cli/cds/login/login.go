package login

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/howeyc/gopass"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

// Cmd login
var Cmd = &cobra.Command{
	Use:   "login",
	Short: "Ease up creation of ~/.cds/config.json",
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
	defaultEndpoint := ""

	fmt.Printf("Username: ")
	username := readline()
	conf.User = username
	fmt.Printf("Password: ")
	password, err := gopass.GetPasswd()
	if err != nil {
		sdk.Exit("Error: wrong usage (%s)\n", err)
	}
	fmt.Printf("CDS endpoint [%s]: ", defaultEndpoint)
	conf.Host = readline()
	if conf.Host == "" {
		conf.Host = defaultEndpoint
	}

	home := os.Getenv("HOME")
	err = os.Mkdir(path.Join(home, ".cds"), 0700)
	if err != nil && !os.IsExist(err) {
		sdk.Exit("Error: Cannot create config folder (%s)\n", err)
	}

	sdk.InitEndpoint(conf.Host)

	loginOK, res, err := sdk.LoginUser(username, string(password))
	if !loginOK {
		if err != nil {
			sdk.Exit("Error: Login failed (%s)\n", err)
		}
	}
	if res.Token != "" {
		conf.Token = res.Token
	} else {
		conf.Password = string(password)
	}

	data, err := json.MarshalIndent(conf, " ", " ")
	if err != nil {
		sdk.Exit("Error: Cannot create config file (%s)\n", err)
	}

	err = ioutil.WriteFile(path.Join(home, ".cds", "config.json"), data, 0640)
	if err != nil {
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
