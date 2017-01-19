package main

import (
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/cli/cds/action"
	"github.com/ovh/cds/cli/cds/admin"
	"github.com/ovh/cds/cli/cds/application"
	"github.com/ovh/cds/cli/cds/artifact"
	"github.com/ovh/cds/cli/cds/dashboard"
	"github.com/ovh/cds/cli/cds/environment"
	"github.com/ovh/cds/cli/cds/generate"
	"github.com/ovh/cds/cli/cds/group"
	"github.com/ovh/cds/cli/cds/internal"
	"github.com/ovh/cds/cli/cds/login"
	"github.com/ovh/cds/cli/cds/pipeline"
	"github.com/ovh/cds/cli/cds/project"
	"github.com/ovh/cds/cli/cds/track"
	"github.com/ovh/cds/cli/cds/trigger"
	"github.com/ovh/cds/cli/cds/update"
	"github.com/ovh/cds/cli/cds/user"
	"github.com/ovh/cds/cli/cds/version"
	"github.com/ovh/cds/cli/cds/wizard"
	"github.com/ovh/cds/cli/cds/worker"
)

var rootCmd = &cobra.Command{
	Use:   "cds",
	Short: "CDS - Command Line Tool",
	Long:  `CDS - Command Line Tool`,
}

func displayWarnings() {
	// Display warnings, on failure, fail silently
	warnings, err := sdk.GetWarnings()
	if err == nil && len(warnings) > 0 {
		fmt.Printf("/!\\ %d warnings found in your CDS configuration:\n", len(warnings))
		for _, w := range warnings {
			fmt.Printf("- %s\n", w.Message)
		}
		fmt.Printf("\n")
	}
}

func main() {
	rootCmd.PersistentFlags().StringVarP(&internal.ConfigFile, "file", "f", "", "set configuration file")
	rootCmd.PersistentFlags().BoolVarP(&internal.Verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&internal.NoWarnings, "no-warnings", "w", false, "do not display warnings")

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		//ConfigFile default file
		if internal.ConfigFile == "" {
			internal.ConfigFile = path.Join(os.Getenv("HOME"), ".cds", "config.json")
		}

		//Set the config file
		sdk.CDSConfigFile = internal.ConfigFile

		//On login command: do nothing
		if cmd == login.Cmd {
			return
		}

		//If file doesn't exist, stop here
		if _, err := os.Stat(internal.ConfigFile); os.IsNotExist(err) {
			sdk.Exit("File %s doesn't exists", internal.ConfigFile)
			return
		}

		//Do not check warning on user command
		if cmd == user.Cmd {
			return
		}

		//Manage warnings
		if internal.NoWarnings {
			return
		}
		displayWarnings()
	}

	rootCmd.AddCommand(login.Cmd)
	rootCmd.AddCommand(action.Cmd)
	rootCmd.AddCommand(application.Cmd())
	rootCmd.AddCommand(artifact.Cmd)
	rootCmd.AddCommand(environment.Cmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(pipeline.Cmd())
	rootCmd.AddCommand(project.Cmd)
	rootCmd.AddCommand(group.Cmd)
	rootCmd.AddCommand(user.Cmd)
	rootCmd.AddCommand(worker.Cmd)
	rootCmd.AddCommand(update.Cmd)
	rootCmd.AddCommand(version.Cmd)
	rootCmd.AddCommand(trigger.Cmd())
	rootCmd.AddCommand(dashboard.Cmd)
	rootCmd.AddCommand(wizard.Cmd)
	rootCmd.AddCommand(track.Cmd)
	rootCmd.AddCommand(generate.Cmd())
	rootCmd.AddCommand(admin.Cmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
