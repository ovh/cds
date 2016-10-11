package user

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/spf13/cobra"

	"os"
	"path"
	"github.com/ovh/cds/sdk"
)

type configFile struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Host     string `json:"host"`
}

func cmdUserVerify() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "cds user verify <username> <token>",
		Long:  ``,
		Run:   verifyUser,
	}

	return cmd
}

func verifyUser(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	name := args[0]
	token := args[1]

	userData, err := sdk.VerifyUser(name, token)
	if err != nil {
		sdk.Exit("%s\n", err)
	}

	userDataString, err := json.MarshalIndent(userData, " ", " ")
	if err != nil {
		fmt.Printf("VerifyUser: Cannot marshal userData: %s\n", err)
		sdk.Exit("%s\n", err)
	}
	fmt.Printf("Account informations : %s\n", userDataString)

	fileContent := &configFile{
		User:     userData.User.Username,
		Password: userData.Password,
		Host:     sdk.Host,
	}

	jsonStr, err := json.MarshalIndent(fileContent, "", "  ")
	if err != nil {
		sdk.Exit("%s\n", err)
	}
	jsonStr = append(jsonStr, '\n')

	dir := path.Dir(sdk.CDSConfigFile)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.Mkdir(dir, 0740)
		if err != nil {
			sdk.Exit("%s\n", err)
		}
	}
	err = ioutil.WriteFile(sdk.CDSConfigFile, jsonStr, 0600)
	if err != nil {
		sdk.Exit("%s\n", err)
	}

}
