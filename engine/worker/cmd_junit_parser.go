package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/engine/worker/internal/action"
	"github.com/ovh/cds/sdk"
)

func cmdJunitParser() *cobra.Command {
	c := &cobra.Command{
		Use:   "junit-parser",
		Short: "worker junit-parser",
		Long: `
worker junit-parser command helps you to parse junit files and print a summary.

It displays the number of tests, the number of passed tests, the number of failed tests and the number of skipped tests.

Examples:
	$ ls
	result1.xml		result2.xml
	$ worker junit-parser result1.xml
	10 10 0 0
	$ worker junit-parser *.xml
	20 20 0 0
`,
		RunE: junitParserCmd(),
	}
	return c
}

func junitParserCmd() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var filepaths []string
		for _, arg := range args {
			matches, err := filepath.Glob(arg)
			if err != nil {
				return err
			}
			filepaths = append(filepaths, matches...)
		}

		var tests sdk.JUnitTestsSuites
		for _, f := range filepaths {
			data, err := os.ReadFile(f)
			if err != nil {
				return fmt.Errorf("junit parser: cannot read file %s (%s)", f, err)
			}
			var ftests sdk.JUnitTestsSuites
			if err := xml.Unmarshal(data, &ftests); err != nil {
				// Check if file contains testsuite only (and no testsuites)
				if s, ok := action.ParseTestsuiteAlone(data); ok {
					ftests.TestSuites = append(ftests.TestSuites, s)
				}
			}
			tests.TestSuites = append(tests.TestSuites, ftests.TestSuites...)
		}

		tests = tests.EnsureData()
		stats := tests.ComputeStats()

		fmt.Println(stats.Total, stats.TotalOK, stats.TotalKO, stats.TotalSkipped)

		return nil
	}
}
