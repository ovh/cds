package pipeline

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func pipelineListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "cds pipeline list <projectkey>",
		Long:    ``,
		Aliases: []string{"ls", "ps"},
		Run:     listPipelines,
	}

	return cmd
}

func listPipelines(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	pip, err := sdk.ListPipelines(args[0])
	if err != nil {
		sdk.Exit("%s\n", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 27, 1, 2, ' ', 0)
	titles := []string{"NAME"}
	fmt.Fprintln(w, strings.Join(titles, "\t"))

	for _, p := range pip {
		fmt.Fprintf(w, "%s\n",
			p.Name,
		)

		w.Flush()
	}
}
