package main

import (
	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/doc"
)

var docCmd = cli.Command{
	Name:   "doc",
	Short:  "generate hugo doc for building http://ovh.github.com/cds",
	Hidden: true,
}

func docRun(v cli.Values) error {
	if err := doc.GenerateDocumentation(root); err != nil {
		sdk.Exit(err.Error())
	}
	return nil
}
