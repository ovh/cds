package importer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/sdk"
)

var (
	rootCmd = &cobra.Command{
		Use:   "import",
		Short: "CDS Admin Import (admin only)",
	}

	importUsersCmd = &cobra.Command{
		Use:   "users",
		Short: "cds admin import users <file>",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				sdk.Exit("Wrong usage.")
			}
			b, err := ioutil.ReadFile(args[0])
			if err != nil {
				sdk.Exit("Error: %s", err)
			}

			var users = []sdk.User{}
			if err := yaml.Unmarshal(b, &users); err != nil {
				sdk.Exit("Error: %s", err)
			}

			jsonB, err := json.Marshal(users)
			if err != nil {
				sdk.Exit("Error: %s", err)
			}

			data, _, err := sdk.Request("POST", "/user/import", jsonB)
			if err != nil {
				sdk.Exit("Error: %s", err)
			}
			errors := map[string]string{}
			if err := json.Unmarshal(data, &errors); err != nil {
				sdk.Exit("Error: %s", err)
			}

			for k, v := range errors {
				fmt.Printf(" - %s : %s\n", k, v)
			}
		},
	}

	importProjectsCmd = &cobra.Command{
		Use:   "projects",
		Short: "cds admin import projects <file>",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				sdk.Exit("Wrong usage.")
			}
			b, err := ioutil.ReadFile(args[0])
			if err != nil {
				sdk.Exit("Error: %s", err)
			}

			var projects = []sdk.Project{}
			if err := yaml.Unmarshal(b, &projects); err != nil {
				sdk.Exit("Error: %s", err)
			}

			for _, p := range projects {
				p.Key = strings.ToUpper(p.Key)
				b, err := json.Marshal(p)
				if err != nil {
					fmt.Printf(" - [%s] %s : %s\n", p.Key, p.Name, err)
					continue
				}
				data, _, err := sdk.Request("POST", "/project", b)
				if err != nil {
					fmt.Printf(" - [%s] %s : %s\n", p.Key, p.Name, err)
					continue
				}

				if e := sdk.DecodeError(data); e != nil {
					fmt.Printf(" - [%s] %s : %s\n", p.Key, p.Name, err)
				}
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(importUsersCmd)
	rootCmd.AddCommand(importProjectsCmd)
}

//Cmd returns the root command
func Cmd() *cobra.Command {
	return rootCmd
}
