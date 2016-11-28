package track

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

// Cmd cobra command for artifact subtree
var Cmd = &cobra.Command{
	Use:   "track",
	Short: "track <commit>",
	Long:  "Track CDS pipeline building given commit",
	Run:   trackCmd,
}

func trackCmd(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		sdk.Exit("Wrong usage: See %s\n", cmd.Short)
	}

	track(args[0])
}
