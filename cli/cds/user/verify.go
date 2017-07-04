package user

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/spf13/cobra"

	"os"
	"path"

	"github.com/ovh/cds/cli/cds/login"
	"github.com/ovh/cds/sdk"
)

func cmdUserVerify() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "cds user verify <username> <token>",
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

	userData, errv := sdk.VerifyUser(name, token)
	if errv != nil {
		sdk.Exit("%s\n", errv)
	}

	userDataString, errm := json.MarshalIndent(userData, " ", " ")
	if errm != nil {
		fmt.Printf("VerifyUser: Cannot marshal userData: %s\n", errm)
		sdk.Exit("%s\n", errm)
	}
	fmt.Printf("Account informations : %s\n", userDataString)

	fileContent := &login.Config{
		User: userData.User.Username,

		Token: userData.Token,
		Host:  sdk.Host,
	}

	jsonStr, errm := json.MarshalIndent(fileContent, "", "  ")
	if errm != nil {
		sdk.Exit("%s\n", errm)
	}
	jsonStr = append(jsonStr, '\n')

	dir := path.Dir(sdk.CDSConfigFile)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.Mkdir(dir, 0740)
		if err != nil {
			sdk.Exit("%s\n", err)
		}
	}

	if err := ioutil.WriteFile(sdk.CDSConfigFile, jsonStr, 0600); err != nil {
		sdk.Exit("%s\n", err)
	}

}
