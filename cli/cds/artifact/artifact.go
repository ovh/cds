package artifact

import (
	"github.com/spf13/cobra"
)

// Cmd cobra command for artifact subtree
var Cmd = &cobra.Command{
	Use:   "artifact",
	Short: "Manage build artifacts",
	Long:  ``,
}

func init() {
	//Cmd.AddCommand(cmdArtifactUpload())
	Cmd.AddCommand(cmdArtifactDownload())
	Cmd.AddCommand(cmdArtifactList())
}
