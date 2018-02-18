package main

import (
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
)

var (
	healthCmd = cli.Command{
		Name:  "health",
		Short: "Check CDS health",
	}

	health = cli.NewCommand(healthCmd, nil,
		[]*cobra.Command{
			cli.NewListCommand(healthStatusCmd, healthStatusRun, nil),
			cli.NewGetCommand(healthMonDBTimesCmd, healthMonDBTimesRun, nil),
			cli.NewListCommand(healthMonDBMigrateCmd, healthMonDBMigrateRun, nil),
		})
)

var healthStatusCmd = cli.Command{
	Name:  "status",
	Short: "Show CDS Status",
}

func healthStatusRun(v cli.Values) (cli.ListResult, error) {
	s, err := client.MonStatus()
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(s.Lines), nil
}

var healthMonDBTimesCmd = cli.Command{
	Name:  "dbtimes",
	Short: "Show some DB Times",
}

func healthMonDBTimesRun(v cli.Values) (interface{}, error) {
	s, err := client.MonDBTimes()
	return *s, err
}

var healthMonDBMigrateCmd = cli.Command{
	Name:  "dbmigrate",
	Short: "Show DB Migrate status",
}

func healthMonDBMigrateRun(v cli.Values) (cli.ListResult, error) {
	s, err := client.MonDBMigrate()
	if err != nil {
		return nil, err
	}
	return cli.AsListResult(s), nil
}
