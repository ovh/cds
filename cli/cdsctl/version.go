package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var versionCmd = cli.Command{
	Name:  "version",
	Short: "show cdsctl version",
}

func version() *cobra.Command {
	return cli.NewCommand(versionCmd, versionRun, nil, cli.CommandWithoutExtraFlags)
}

func versionRun(v cli.Values) error {
	fmt.Println(sdk.VersionString())
	version, err := client.Version()
	if err != nil {
		return err
	}
	fmt.Printf("CDS api version: %s\n", version.Version)
	fmt.Printf("CDS URL: %s\n", client.APIURL())
	return nil
}
