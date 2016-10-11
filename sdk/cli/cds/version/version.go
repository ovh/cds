package version

import (
	"fmt"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cli/cds/internal"

	"github.com/spf13/cobra"
)

// Cmd version
var Cmd = &cobra.Command{
	Use:     "version",
	Short:   "Display Version of cds : cds version",
	Long:    `cds version`,
	Aliases: []string{"v"},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version cds : %s\n", sdk.VERSION)
		fmt.Printf("Architecture : %s\n", internal.Architecture)
		fmt.Printf("Git Sha1 : %s\n", internal.Sha1)
		fmt.Printf("Binary Creation Date : %s\n", internal.DateCreation)
		fmt.Printf("Packaging Informations : %s\n", internal.PackagingInformations)
	},
}
