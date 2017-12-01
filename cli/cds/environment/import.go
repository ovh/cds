package environment

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

var importFormat, importInto string

func importCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "cds environment import <projectKey> <file> [--env environmentName]",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 {
				sdk.Exit("Wrong usage: see %s\n", cmd.Short)
			}
			projectKey := args[0]
			name := args[1]
			msg := []string{}

			importFormat = "yaml"
			if strings.HasSuffix(name, ".json") {
				importFormat = "json"
			}

			var payload = &exportentities.Environment{}

			f, errF := exportentities.GetFormat(importFormat)
			if errF != nil {
				sdk.Exit("Error: %s\n", errF)
			}

			btes, err := ioutil.ReadFile(name)
			if err != nil {
				sdk.Exit("Error: %s\n", err)
			}

			var errorParse error
			switch f {
			case exportentities.FormatJSON:
				errorParse = json.Unmarshal(btes, payload)
			case exportentities.FormatYAML:
				errorParse = yaml.Unmarshal(btes, payload)
			}

			if errorParse != nil {
				sdk.Exit("Error: %s\n", errorParse)
			}

			var url string

			if importInto == "" {
				url = fmt.Sprintf("/project/%s/environment/import?format=%s", projectKey, importFormat)
			} else {
				p, errP := sdk.GetProject(projectKey, sdk.WithEnvs())
				if errP != nil {
					sdk.Exit("Error: %s\n", errP)
				}

				envExist := false
				for _, e := range p.Environments {
					if e.Name == importInto {
						envExist = true
						break
					}
				}

				//If user import into a non existing env, lets change the name existing in the file and call the import handler
				if !envExist {
					fmt.Println("Environment doesn't exist. It will be created.")
					payload.Name = importInto
					btes, err = yaml.Marshal(payload)
					if err != nil {
						sdk.Exit("Error: %s\n", err)
					}
					url = fmt.Sprintf("/project/%s/environment/import?format=%v", projectKey, "yaml")
				} else {
					url = fmt.Sprintf("/project/%s/environment/import/%s?format=%s", projectKey, importInto, importFormat)
				}
			}

			data, _, err := sdk.Request("POST", url, btes)
			if err != nil {
				sdk.Exit("Error: %s\n", err)
			}

			if err := json.Unmarshal(data, &msg); err != nil {
				sdk.Exit("Error: %s\n", err)
			}

			for _, s := range msg {
				fmt.Println(s)
			}
		},
	}

	cmd.Flags().StringVarP(&importInto, "env", "", "", "Import environment variables into an existing environment")

	return cmd
}
