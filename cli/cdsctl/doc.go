package main

import (
	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/doc"
)

var docCmd = cli.Command{
	Name:  "doc",
	Short: "generate hugo doc for building http://ovh.github.com/cds",
	Long: `With generation-path, you can comment Handler as:
// @title A title
// @description the description
// @params AA=valA
// @params BB=valB
// @body {"mykey": "myval"}
	`,
	Hidden: true,
	Args: []cli.Arg{
		{Name: "generation-path"},
	},
}

func docRun(v cli.Values) error {
	if err := doc.GenerateDocumentation(root, v.GetString("generation-path"), ""); err != nil {
		sdk.Exit(err.Error())
	}
	return nil
}
