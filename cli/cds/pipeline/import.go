package pipeline

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"
)

var importFormat string

func importCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "cds pipeline import <projectKey> <file>",
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

			url = fmt.Sprintf("/project/%s/pipeline/import?format=%s", projectKey, importFormat)

			data, _, err := sdk.Request("POST", url, btes)
			json.Unmarshal(data, &msg)
			if err != nil {
				sdk.Exit("Error: %s\n", err)
			}

			for _, s := range msg {
				fmt.Println(s)
			}
		},
	}

	return cmd
}
