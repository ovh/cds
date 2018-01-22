package version

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ovh/tat"
	"github.com/ovh/tat/tatcli/internal"
)

var versionNewLine bool

func init() {
	Cmd.Flags().BoolVarP(&versionNewLine, "versionNewLine", "", true, "New line after version number. If true, display Version Engine too")
}

// Cmd version
var Cmd = &cobra.Command{
	Use:     "version",
	Short:   "Display Version of tatcli and tat engine if configured: tatcli version",
	Long:    `tatcli version`,
	Aliases: []string{"v"},
	Run: func(cmd *cobra.Command, args []string) {
		if versionNewLine {
			fmt.Printf("Version tatcli: %s\n", tat.Version)

			internal.ReadConfig()
			if viper.GetString("url") == "" {
				internal.Exit("Version Engine: No Engine Configured. See tatcli config --help\n")
			} else {
				out, err := internal.Client().Version()
				internal.Check(err)
				internal.Print(out)
			}
		} else {
			fmt.Print(tat.Version)
		}
	},
}
