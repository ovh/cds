package artifact

import (
	"github.com/ovh/cds/sdk"

	"github.com/spf13/cobra"
)

var environment string

func cmdArtifactDownload() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "download",
		Short:   "cds artifact download <projectName> <applicationName> <pipelineName> <tag> [artifactName]",
		Long:    ``,
		Run:     downloadArtifacts,
		Aliases: []string{"dl"},
	}
	cmd.Flags().StringVarP(&environment, "env", "", "", "environment name")
	return cmd
}

func downloadArtifacts(cmd *cobra.Command, args []string) {
	if len(args) == 5 {
		downloadArtifact(args)
		return
	}

	if len(args) != 4 {
		sdk.Exit("Wrong usage: %s\n", cmd.Short)
	}

	basedir := "."
	project := args[0]
	appName := args[1]
	pipeline := args[2]
	tag := args[3]

	err := sdk.DownloadArtifacts(project, appName, pipeline, tag, basedir, environment)
	if err != nil {
		sdk.Exit("Error: Cannot download artifacts in %s-%s-%s/%s (%s)\n", project, appName, pipeline, tag, err)
	}
}

func downloadArtifact(args []string) {
	basedir := "."
	project := args[0]
	appName := args[1]
	pipeline := args[2]
	tag := args[3]
	filename := args[4]

	err := sdk.DownloadArtifact(project, appName, pipeline, tag, basedir, environment, filename)
	if err != nil {
		sdk.Exit("Error: Cannot download artifact %s (%s)\n", filename, err)
	}
}
