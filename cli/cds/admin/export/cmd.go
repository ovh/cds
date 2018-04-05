package export

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/sdk"
)

var (
	rootCmd = &cobra.Command{
		Use:   "export",
		Short: "CDS Admin Export (admin only)",
	}

	exportUsersCmd = &cobra.Command{
		Use:   "users",
		Short: "cds admin export users",
		Run: func(cmd *cobra.Command, args []string) {
			if ok, err := sdk.IsAdmin(); !ok {
				if err != nil {
					fmt.Printf("Error : %v\n", err)
				}
				sdk.Exit("You are not allowed to run this command")
			}

			users, err := sdk.ListUsers()
			if err != nil {
				sdk.Exit("Error: %s", err)
			}

			b, err := yaml.Marshal(users)
			if err != nil {
				sdk.Exit("Error: %s", err)
			}

			if exportUsersCmdOutputFlag == "" {
				fmt.Println(string(b))
				return
			}

			if err := ioutil.WriteFile(exportUsersCmdOutputFlag, b, os.FileMode(0644)); err != nil {
				sdk.Exit("Error: %s", err)
			}
		},
	}

	exportUsersCmdOutputFlag string
)

func init() {
	rootCmd.AddCommand(exportUsersCmd)
	exportUsersCmd.Flags().StringVarP(&exportUsersCmdOutputFlag, "output", "o", "", "cds admin export users -o <filename>")
}

//Cmd returns the root command
func Cmd() *cobra.Command {
	return rootCmd
}
