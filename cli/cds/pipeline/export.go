package pipeline

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

var (
	exportFormat, exportOutput string
	exportWithPermissions      bool
)

func exportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "cds pipeline export <projectKey> <pipeline>",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				sdk.Exit("Wrong usage: see %s\n", cmd.Short)
			}
			projectKey := args[0]
			name := args[1]

			pip, err := sdk.GetPipeline(projectKey, name)
			if err != nil {
				sdk.Exit("Error %s\n", err)
			}

			p := exportentities.NewPipeline(pip, exportWithPermissions)

			f, err := exportentities.GetFormat(exportFormat)
			if err != nil {
				sdk.Exit("Error %s\n", err)
			}

			btes, errMarshal := exportentities.Marshal(p, f)
			if errMarshal != nil {
				sdk.Exit("Error %s\n", errMarshal)
			}

			if exportOutput == "" {
				fmt.Println(string(btes))
			} else {
				if err := ioutil.WriteFile(exportOutput, btes, os.FileMode(0644)); err != nil {
					sdk.Exit("Error %s\n", err)
				}
			}

		},
	}

	cmd.Flags().StringVarP(&exportFormat, "format", "", "yaml", "Format: json|yaml|hcl")
	cmd.Flags().StringVarP(&exportOutput, "output", "", "", "Output filename")
	cmd.Flags().BoolVarP(&exportWithPermissions, "withPermissions", "", false, "Export pipeline configuration with permission")

	return cmd
}
