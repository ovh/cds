package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/ovh/venom/cli/venom/run"
	"github.com/ovh/venom/cli/venom/template"
	"github.com/ovh/venom/cli/venom/update"
	"github.com/ovh/venom/cli/venom/version"
)

var rootCmd = &cobra.Command{
	Use:   "venom",
	Short: "Venom - RUN Integration Tests",
}

func main() {
	addCommands()

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Err:%s", err)
	}
}

//AddCommands adds child commands to the root command rootCmd.
func addCommands() {
	rootCmd.AddCommand(run.Cmd)
	rootCmd.AddCommand(template.Cmd)
	rootCmd.AddCommand(version.Cmd)
	rootCmd.AddCommand(update.Cmd)
}
