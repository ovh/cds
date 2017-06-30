package main

import "github.com/ovh/cds/cli"

var healthCmd = cli.Command{
	Name:  "health",
	Short: "Check CDS health",
}

func healthRun(v cli.Values, filter map[string]string) (cli.ListResult, error) {
	s, err := client.MonStatus()
	return cli.AsListResult(s, filter, ""), err
}
