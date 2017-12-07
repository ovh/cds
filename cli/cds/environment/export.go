package environment

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

var exportFormat, exportOutput string

func exportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "cds environment export <projectKey> <environmentName>",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				sdk.Exit("Wrong usage: see %s\n", cmd.Short)
			}
			projectKey := args[0]
			name := args[1]

			env, err := sdk.GetEnvironment(projectKey, name)
			if err != nil {
				sdk.Exit("Error %s\n", err)
			}

			e := exportentities.NewEnvironment(*env, false, nil)

			f, err := exportentities.GetFormat(exportFormat)
			if err != nil {
				sdk.Exit("Error %s\n", err)
			}

			btes, errMarshal := exportentities.Marshal(e, f)
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

	return cmd
}
