package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/engine/venom/run"
	"github.com/ovh/cds/engine/venom/template"
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
}
