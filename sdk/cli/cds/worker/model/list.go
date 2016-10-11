package model

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/ovh/cds/sdk"

	"github.com/spf13/cobra"
)

func cmdWorkerModelList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "cds worker model list",
		Long:  ``,
		Run:   listWorkerModel,
	}

	return cmd
}

func listWorkerModel(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}

	models, err := sdk.GetWorkerModels()
	if err != nil {
		sdk.Exit("Error: cannot get worker models (%s)\n", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 27, 1, 2, ' ', 0)
	titles := []string{"NAME", "TYPE", "IMAGE"}
	fmt.Fprintln(w, strings.Join(titles, "\t"))

	for _, m := range models {
		if len(m.Image) > 100 {
			m.Image = m.Image[:97] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\n",
			m.Name,
			m.Type,
			m.Image,
		)

		w.Flush()
	}
}
