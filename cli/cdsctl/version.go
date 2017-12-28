package main

import (
	"fmt"
	"runtime"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var versionCmd = cli.Command{
	Name:  "version",
	Short: "show cdsctl version",
}

func versionRun(v cli.Values) error {
	fmt.Printf("CDS cdsctl version: %s os:%s architecture:%s\n", sdk.VERSION, runtime.GOOS, runtime.GOARCH)
	version, err := client.Version()
	if err != nil {
		return err
	}
	fmt.Printf("CDS api version: %s\n", version.Version)
	return nil
}
