package environment

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
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
			} else if strings.HasSuffix(name, ".hcl") {
				importFormat = "hcl"
			}

			btes, err := ioutil.ReadFile(name)
			if err != nil {
				sdk.Exit("Error: %s\n", err)
			}

			var url string

			if importInto == "" {
				url = fmt.Sprintf("/project/%s/environment/import?format=%s", projectKey, importFormat)
			} else {
				url = fmt.Sprintf("/project/%s/environment/import/%s?format=%s", projectKey, importInto, importFormat)
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
