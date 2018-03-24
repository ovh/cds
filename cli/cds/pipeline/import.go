package pipeline

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

var (
	importFormat, importGit, importURL string
	importForce                        bool
)

func importCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "cds pipeline import <projectKey> [file] [--url <url> --format json|yaml] [--force]",
		Long:  "See documentation on https://ovh.github.io/cds/workflows/pipelines/configuration-file/",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				sdk.Exit("Wrong usage: see %s\n", cmd.Short)
			}

			projectKey := args[0]
			msg := []string{}
			btes := []byte{}

			if len(args) == 2 {
				name := args[1]
				importFormat = "yaml"
				if strings.HasSuffix(name, ".json") {
					importFormat = "json"
				}
				var err error
				btes, _, err = exportentities.ReadFile(name)
				if err != nil {
					sdk.Exit("Error: %s\n", err)
				}
			} else if importURL != "" {
				var err error
				btes, _, err = exportentities.ReadURL(importURL, importFormat)
				if err != nil {
					sdk.Exit("Error: %s\n", err)
				}
			} else {
				sdk.Exit("Wrong usage: see %s\n", cmd.Short)
			}

			var url string
			url = fmt.Sprintf("/project/%s/import/pipeline?format=%s", projectKey, importFormat)

			if importForce {
				url += "&forceUpdate=true"
			}

			data, code, err := sdk.Request("POST", url, btes)
			if sdk.ErrorIs(err, sdk.ErrPipelineAlreadyExists) {
				fmt.Print("Pipline already exists. ")
				if cli.AskForConfirmation("Do you want to override ?") {
					url = fmt.Sprintf("/project/%s/import/pipeline?format=%s&forceUpdate=true", projectKey, importFormat)
					data, code, err = sdk.Request("POST", url, btes)
				} else {
					sdk.Exit("Aborted\n")
				}
			}

			if code > 400 {
				sdk.Exit("Error: %d - %s\n", code, err)
			}
			if err != nil {
				sdk.Exit("Error: %s\n", err)
			}

			if err := json.Unmarshal(data, &msg); err != nil {
				sdk.Exit("Error: %s\n", err)
			}

			for _, s := range msg {
				fmt.Println(s)
			}

			if code == 400 {
				sdk.Exit("Error while importing pipeline\n")
			}
		},
	}

	cmd.Flags().StringVarP(&importURL, "url", "", "", "Import pipeline from an URL")
	cmd.Flags().StringVarP(&importFormat, "format", "", "yaml", "Configuration file format")
	cmd.Flags().BoolVarP(&importForce, "force", "", false, "Use force flag to update your pipeline")

	return cmd
}
