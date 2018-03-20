package pipeline

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func pipelineHistoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "cds pipeline history <projectKey> <applicationName> <pipelineName> [envName]",
		Long:  ``,
		Run:   historyPipeline,
	}

	return cmd
}

func historyPipeline(cmd *cobra.Command, args []string) {

	if len(args) < 3 || len(args) > 4 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	projectKey := args[0]
	appName := args[1]
	pipelineName := args[2]
	var envName string
	if len(args) == 4 {
		envName = args[3]
	}
	builds, err := sdk.GetPipelineBuildHistory(projectKey, appName, pipelineName, envName, "")
	if err != nil {
		sdk.Exit("Error: cannot retrieve build history (%s)\n", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 27, 1, 2, ' ', 0)
	titles := []string{"BUILD", "VERSION", "STATUS", "BRANCH"}
	fmt.Fprintln(w, strings.Join(titles, "\t"))

	for _, b := range builds {
		fmt.Fprintf(w, "#%d\t%d\t%s\t%s\n",
			b.BuildNumber,
			b.Version,
			b.Status,
			b.Trigger.VCSChangesBranch,
		)

		w.Flush()
	}
}
