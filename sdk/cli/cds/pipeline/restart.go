package pipeline

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

func pipelineRestartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restart",
		Short: "cds pipeline run <projectKey> <appName> <pipelineName> [envName] <buildNumber>",
		Long:  ``,
		Run:   restartPipeline,
	}

	cmd.Flags().BoolVarP(&batch, "batch", "", false, "Do not stream logs")

	return cmd
}

func restartPipeline(cmd *cobra.Command, args []string) {

	if len(args) < 4 {
		sdk.Exit("Wrong usage: see %s\n", cmd.Short)
	}

	pk := args[0]
	app := args[1]
	name := args[2]
	var env string
	var bnS string
	if len(args) > 4 {
		env = args[3]
		bnS = args[4]
	} else {
		bnS = args[3]
	}

	bn, err := strconv.Atoi(bnS)
	if err != nil {
		sdk.Exit("%s is not a valid build number (%s)\n", bnS, err)
	}

	ch, err := sdk.RestartPipeline(pk, app, name, env, bn)
	if err != nil {
		sdk.Exit("Cannot restart pipeline (%s)\n", err)
	}

	if batch {
		return
	}

	streamResponse(ch)
}
