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
	mainCmd.AddCommand(docCmd) // hidden command
}

func main() {
	mainCmd.Execute()
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

You will find lastest release of CDS ` + "`engine`" + ` on [Github Releases](https://github.com/ovh/cds/releases/latest).
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
