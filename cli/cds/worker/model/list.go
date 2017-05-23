package model

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func cmdWorkerModelList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "cds worker model list",
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
	titles := []string{"NAME", "TYPE", "PROTOCOL", "DISABLED", "IMAGE", "LAST_MODIFIED", "LAST_REGISTRATION", "NEED_REGISTRATION"}
	fmt.Fprintln(w, strings.Join(titles, "\t"))

	for _, m := range models {
		if len(m.Image) > 100 {
			m.Image = m.Image[:97] + "..."
		}

		fmt.Fprintf(w, "%-30s\t%-10s\t%-4s\t%t\t%-50s\t%s\t%s\t%t\n",
			m.Name,
			m.Type,
			m.Communication,
			m.Disabled,
			m.Image,
			m.UserLastModified,
			m.LastRegistration,
			m.NeedRegistration,
		)

		w.Flush()
	}
}
