package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cli/cds/action"
	"github.com/ovh/cds/sdk/cli/cds/application"
	"github.com/ovh/cds/sdk/cli/cds/artifact"
	"github.com/ovh/cds/sdk/cli/cds/dashboard"
	"github.com/ovh/cds/sdk/cli/cds/environment"
	"github.com/ovh/cds/sdk/cli/cds/generate"
	"github.com/ovh/cds/sdk/cli/cds/group"
	"github.com/ovh/cds/sdk/cli/cds/internal"
	"github.com/ovh/cds/sdk/cli/cds/login"
	"github.com/ovh/cds/sdk/cli/cds/pipeline"
	"github.com/ovh/cds/sdk/cli/cds/plugin"
	"github.com/ovh/cds/sdk/cli/cds/project"
	"github.com/ovh/cds/sdk/cli/cds/repositoriesmanager"
	"github.com/ovh/cds/sdk/cli/cds/track"
	"github.com/ovh/cds/sdk/cli/cds/trigger"
	"github.com/ovh/cds/sdk/cli/cds/update"
	"github.com/ovh/cds/sdk/cli/cds/user"
	"github.com/ovh/cds/sdk/cli/cds/version"
	"github.com/ovh/cds/sdk/cli/cds/wizard"
	"github.com/ovh/cds/sdk/cli/cds/worker"
)

var rootCmd = &cobra.Command{
	Use:   "cds",
	Short: "CDS - Command Line Tool",
	Long:  `CDS - Command Line Tool`,
}

func main() {
	rootCmd.PersistentFlags().StringVarP(&sdk.Host, "host", "H", "", "")
	rootCmd.PersistentFlags().BoolVarP(&internal.Verbose, "verbose", "v", false, "verbose output")

	viper.BindPFlag("host", rootCmd.PersistentFlags().Lookup("host"))

	// Display warnings, on failure, fail silently
	warnings, err := sdk.GetWarnings()
	if err == nil && len(warnings) > 0 {
		fmt.Printf("/!\\ %d warnings found in your CDS configuration:\n", len(warnings))
		for _, w := range warnings {
			fmt.Printf("- %s\n", w.Message)
		}
		fmt.Printf("\n")
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
	rootCmd.AddCommand(repositoriesmanager.Cmd())
	rootCmd.AddCommand(plugin.Cmd())
	rootCmd.AddCommand(generate.Cmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
