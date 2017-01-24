package export

import "github.com/spf13/cobra"
import "github.com/ovh/cds/sdk"
import "gopkg.in/yaml.v2"
import "fmt"
import "io/ioutil"
import "os"

var (
	rootCmd = &cobra.Command{
		Use:   "export",
		Short: "CDS Admin Export (admin only)",
	}

	exportGroupsCmd = &cobra.Command{
		Use:   "groups",
		Short: "cds admin export groups",
		Run: func(cmd *cobra.Command, args []string) {
			if ok, err := sdk.IsAdmin(); !ok {
				if err != nil {
					fmt.Printf("Error : %v\n", err)
				}
				sdk.Exit("You are not allowed to run this command")
			}

			groups, err := sdk.ListGroups()
			if err != nil {
				sdk.Exit("Error: %s", err)
			}

			b, err := yaml.Marshal(groups)
			if err != nil {
				sdk.Exit("Error: %s", err)
			}

			if exportGroupsCmdOutputFlag == "" {
				fmt.Println(string(b))
				return
			}

			if err := ioutil.WriteFile(exportGroupsCmdOutputFlag, b, os.FileMode(0644)); err != nil {
				sdk.Exit("Error: %s", err)
			}
		},
	}

	exportGroupsCmdOutputFlag string

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

	exportProjectsCmd = &cobra.Command{
		Use:   "projects",
		Short: "cds admin export projects",
		Run: func(cmd *cobra.Command, args []string) {
			if ok, err := sdk.IsAdmin(); !ok {
				if err != nil {
					fmt.Printf("Error : %v\n", err)
				}
				sdk.Exit("You are not allowed to run this command")
			}

			projects, err := sdk.ListProject()
			if err != nil {
				sdk.Exit("Error: %s", err)
			}

			//Export without applications, pipelines and env
			for i := range projects {
				p, err := sdk.GetProject(projects[i].Key)
				if err != nil {
					sdk.Exit("Error: %s", err)
				}
				projects[i] = p
				projects[i].Applications = nil
				projects[i].Pipelines = nil
				projects[i].Environments = nil
				projects[i].Variable = nil
			}

			b, err := yaml.Marshal(projects)
			if err != nil {
				sdk.Exit("Error: %s", err)
			}

			if exportProjectsCmdOutputFlag == "" {
				fmt.Println(string(b))
				return
			}

			if err := ioutil.WriteFile(exportProjectsCmdOutputFlag, b, os.FileMode(0644)); err != nil {
				sdk.Exit("Error: %s", err)
			}
		},
	}

	exportProjectsCmdOutputFlag string
)

func init() {
	rootCmd.AddCommand(exportGroupsCmd)
	rootCmd.AddCommand(exportProjectsCmd)
	rootCmd.AddCommand(exportUsersCmd)

	exportGroupsCmd.Flags().StringVarP(&exportGroupsCmdOutputFlag, "output", "o", "", "cds admin export groups -o <filename>")
	exportProjectsCmd.Flags().StringVarP(&exportProjectsCmdOutputFlag, "output", "o", "", "cds admin export projects -o <filename>")
	exportUsersCmd.Flags().StringVarP(&exportUsersCmdOutputFlag, "output", "o", "", "cds admin export users -o <filename>")

}

//Cmd returns the root command
func Cmd() *cobra.Command {
	return rootCmd
}
