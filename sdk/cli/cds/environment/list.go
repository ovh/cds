package environment

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func environmentListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "cds environment list <projectkey>",
		Long:    ``,
		Aliases: []string{"ls", "ps"},
		Run:     listEnvironments,
	}

	return cmd
}

func listEnvironments(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	apps, err := sdk.ListEnvironments(args[0])
	if err != nil {
		sdk.Exit("%s\n", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 27, 1, 2, ' ', 0)
	titles := []string{"NAME"}
	fmt.Fprintln(w, strings.Join(titles, "\t"))

	for _, a := range apps {
		fmt.Fprintf(w, "%s\n",
			a.Name,
		)

		w.Flush()
	}
}
