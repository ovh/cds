package message

import (
	"github.com/ovh/tat/tatcli/internal"
	"github.com/spf13/cobra"
)

var cascade, cascadeForce bool

func init() {
	cmdMessageDelete.Flags().BoolVarP(&cascade, "cascade", "", false, "--cascade : delete message and its replies")
	cmdMessageDelete.Flags().BoolVarP(&cascadeForce, "cascadeForce", "", false, "--cascadeForce : delete message and its replies, event if it's in a Tasks Topic of one user")
}

var cmdMessageDelete = &cobra.Command{
	Use:     "delete",
	Aliases: []string{"rm"},
	Short:   "Delete a message: tatcli message delete <topic> <idMessage> [--cascade] [--cascadeForce]",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 2 {
			out, err := internal.Client().MessageDelete(args[1], args[0], cascade, cascadeForce)
			internal.Check(err)
			if internal.Verbose {
				internal.Print(out)
			}
		} else {
			internal.Exit("Invalid argument to delete message: tatcli message delete --help\n")
		}
	},
}
