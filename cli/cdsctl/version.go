package main

import (
	"fmt"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var versionCmd = cli.Command{
	Name:  "version",
	Short: "show cdsctl version",
}

func versionRun(v cli.Values) error {
	fmt.Printf("CDS cdsctl version: %s os:%s architecture:%s\n", sdk.VERSION, sdk.OS, sdk.ARCH)
	version, err := client.Version()
	if err != nil {
		return err
	}
	fmt.Printf("CDS api version: %s\n", version.Version)
	return nil
}
