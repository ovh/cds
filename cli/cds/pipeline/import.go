package pipeline

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

var importFormat, importGit, importURL string

func importCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "cds pipeline import <projectKey> [file] [--git <your-repository> --format json|yaml] [--url <url> --format json|yaml]",
		Long:  "See documentation on https://github.com/ovh/cds/tree/master/doc/tutorials",
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
				} else if strings.HasSuffix(name, ".hcl") {
					importFormat = "hcl"
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
			} else if importGit != "" {
				var err error
				var format exportentities.Format
				btes, format, err = exportentities.ReadGit(importGit)
				if err != nil {
					sdk.Exit("Error: %s\n", err)
				}
			} else {
				sdk.Exit("Wrong usage: see %s\n", cmd.Short)
			}

			var url string
			url = fmt.Sprintf("/project/%s/pipeline/import?format=%s", projectKey, importFormat)

			data, code, err := sdk.Request("POST", url, btes)
			json.Unmarshal(data, &msg)
			if err != nil {
				sdk.Exit("Error: %s\n", err)
			}

			for _, s := range msg {
				fmt.Println(s)
			}

			if code >= 400 {
				sdk.Exit("Error while importing pipeline\n")
			}
		},
	}

	cmd.Flags().StringVarP(&importGit, "git", "", "", "Import pipeline from a git repository. Default filename if .cds.pip.yml")
	cmd.Flags().StringVarP(&importURL, "url", "", "", "Import pipeline from an URL")
	cmd.Flags().StringVarP(&importFormat, "format", "", "yaml", "Configuration file format")

	return cmd
}
