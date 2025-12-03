package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	_ "github.com/spf13/viper/remote"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/doc"
)

func init() {
	mainCmd.AddCommand(versionCmd)
	mainCmd.AddCommand(updateCmd)
	mainCmd.AddCommand(uptodateCmd)
	mainCmd.AddCommand(databaseCmd)
	mainCmd.AddCommand(startCmd)
	mainCmd.AddCommand(configCmd)
	mainCmd.AddCommand(downloadCmd)
	mainCmd.AddCommand(docCmd)    // hidden command
	mainCmd.AddCommand(schemaCmd) // hidden command TODO: use for doc generation
}

func main() {
	if err := mainCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var mainCmd = &cobra.Command{
	Use:   "engine",
	Short: "CDS Engine",
	Long: `
CDS

Continuous Delivery Service

Enterprise-Grade Continuous Delivery & DevOps Automation Open Source Platform

https://ovh.github.io/cds/

## Download

You will find latest release of CDS ` + "`engine`" + ` on [Github Releases](https://github.com/ovh/cds/releases/latest).
`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display CDS version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(sdk.VersionString())
	},
}

var docCmd = &cobra.Command{
	Use:    "doc <generation-path> <git-directory>",
	Short:  "generate hugo doc for building http://ovh.github.com/cds",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			cmd.Usage()
			os.Exit(1)
		}
		if err := doc.GenerateDocumentation(mainCmd, args[0], args[1]); err != nil {
			sdk.Exit(err.Error())
		}
	},
}

var schemaCmd = &cobra.Command{
	Use:    "schema <directory>",
	Short:  "generate CDS resources yaml file",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {

		var directory string
		if len(args) == 1 {
			directory = args[0]
		}

		if err := sdk.GetYamlFromJsonSchema(sdk.EntityTypeWorkerModel, directory); err != nil {
			sdk.Exit(err.Error())
		}
		if err := sdk.GetYamlFromJsonSchema(sdk.EntityTypeAction, directory); err != nil {
			sdk.Exit(err.Error())
		}
		if err := sdk.GetYamlFromJsonSchema(sdk.EntityTypeWorkflow, directory); err != nil {
			sdk.Exit(err.Error())
		}
		if err := sdk.GetYamlFromJsonSchema(sdk.EntityTypeWorkflowTemplate, directory); err != nil {
			sdk.Exit(err.Error())
		}
	},
}
