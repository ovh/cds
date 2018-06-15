package main

import (
	"crypto/tls"
	"net/http"
	"os"
	"path"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli/cds/admin"
	"github.com/ovh/cds/cli/cds/artifact"
	"github.com/ovh/cds/cli/cds/environment"
	"github.com/ovh/cds/cli/cds/generate"
	"github.com/ovh/cds/cli/cds/internal"
	"github.com/ovh/cds/cli/cds/login"
	"github.com/ovh/cds/cli/cds/pipeline"
	"github.com/ovh/cds/cli/cds/project"
	"github.com/ovh/cds/cli/cds/track"
	"github.com/ovh/cds/cli/cds/trigger"
	"github.com/ovh/cds/cli/cds/update"
	"github.com/ovh/cds/cli/cds/user"
	"github.com/ovh/cds/cli/cds/version"
	"github.com/ovh/cds/cli/cds/workflow"
	"github.com/ovh/cds/sdk"
)

var rootCmd = &cobra.Command{
	Use:   "cds",
	Short: "CDS - Command Line Tool",
}

func main() {
	rootCmd.PersistentFlags().StringVarP(&internal.ConfigFile, "file", "f", "", "set configuration file")
	rootCmd.PersistentFlags().BoolVarP(&internal.Verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&internal.NoWarnings, "no-warnings", "w", false, "do not display warnings")
	rootCmd.PersistentFlags().BoolVarP(&internal.InsecureSkipVerifyTLS, "insecure", "k", false, `(SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.`)

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		//ConfigFile default file
		if internal.ConfigFile == "" {
			internal.ConfigFile = path.Join(os.Getenv("HOME"), ".cds", "config.json")
		}

		//Set the config file
		sdk.CDSConfigFile = internal.ConfigFile

		//Set http client
		c := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: internal.InsecureSkipVerifyTLS},
			}}
		sdk.SetHTTPClient(c)

		//On login command: do nothing
		if cmd == login.CmdLogin || cmd == login.CmdSignup {
			return
		}

		//If file doesn't exist, stop here
		if _, err := os.Stat(internal.ConfigFile); os.IsNotExist(err) {
			sdk.Exit("File %s doesn't exists", internal.ConfigFile)
			return
		}

		//Read the config file
		if err := sdk.ReadConfig(); err != nil {
			sdk.Exit("Config error %s", err)
		}

		//Just one try
		sdk.SetRetry(1)

	}

	rootCmd.AddCommand(login.CmdLogin)
	rootCmd.AddCommand(login.CmdSignup)
	rootCmd.AddCommand(artifact.Cmd)
	rootCmd.AddCommand(environment.Cmd())
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(pipeline.Cmd())
	rootCmd.AddCommand(workflow.Cmd())
	rootCmd.AddCommand(project.Cmd)
	rootCmd.AddCommand(user.Cmd)
	rootCmd.AddCommand(update.Cmd)
	rootCmd.AddCommand(version.Cmd)
	rootCmd.AddCommand(trigger.Cmd())
	rootCmd.AddCommand(track.Cmd)
	rootCmd.AddCommand(generate.Cmd())
	rootCmd.AddCommand(admin.Cmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
