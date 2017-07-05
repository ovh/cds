package main

import "github.com/ovh/cds/cli"

var healthCmd = cli.Command{
	Name:  "health",
	Short: "Check CDS health",
}

func healthRun(v cli.Values) (interface{}, error) {
	s, err := client.MonStatus()
	return s, err
}
