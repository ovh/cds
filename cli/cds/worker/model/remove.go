package model

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

var forceDelete bool

func cmdWorkerModelRemove() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove",
		Short:   "cds worker model remove <name>",
		Long:    ``,
		Run:     removeWorkerModel,
		Aliases: []string{"rm"},
	}

	cmd.Flags().BoolVarP(&forceDelete, "force", "", false, "delete worker model, exit 0 if worker model does not exist")
	return cmd
}

func removeWorkerModel(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}
	name := args[0]

	m, err := sdk.GetWorkerModel(name)
	if err != nil {
		if forceDelete && sdk.ErrorIs(err, sdk.ErrNoWorkerModel) {
			fmt.Printf("%s\n", err.Error())
			return
		}
		sdk.Exit("Error: cannot retrieve worker model (%s)\n", err)
	}

	err = sdk.DeleteWorkerModel(m.ID)
	if err != nil {
		sdk.Exit("Error: cannot remove worker model (%s)\n", err)
	}
}
