package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/venom"
)

func cmdJunit(w *currentWorker) *cobra.Command {
	c := &cobra.Command{
		Use:   "junit",
		Short: "worker junit",
		Long: `
worker junit command helps you to parse junit files and print a summary. 

It displays the number of tests, the number of passed tests, the number of failed tests and the number of skipped tests.

Examples:
	$ ls 
	result1.xml		result2.xml
	$ worker junit result1.xml
	10 10 0 0
	$ worker junit *.xml
	20 20 0 0

`,
		RunE: junitCmd(w),
	}
	return c
}

func junitCmd(w *currentWorker) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var filepaths []string
		for _, arg := range args {
			matches, err := filepath.Glob(arg)
			if err != nil {
				return err
			}
			filepaths = append(filepaths, matches...)
		}

		var tests venom.Tests
		for _, f := range filepaths {
			var ftests venom.Tests
			data, err := ioutil.ReadFile(f)
			if err != nil {
				return fmt.Errorf("junit parser: cannot read file %s (%s)", f, err)
			}
			var vf venom.Tests
			if err := xml.Unmarshal(data, &vf); err != nil {
				// Check if file contains testsuite only (and no testsuites)
				if s, ok := parseTestsuiteAlone(data); ok {
					ftests.TestSuites = append(ftests.TestSuites, s)
				}
				tests.TestSuites = append(tests.TestSuites, ftests.TestSuites...)
			} else {
				tests.TestSuites = append(tests.TestSuites, vf.TestSuites...)
			}
		}

		var res sdk.Result
		_ = computeStats(&res, &tests)

		fmt.Println(tests.Total, tests.TotalOK, tests.TotalKO, tests.TotalSkipped)

		return nil
	}
}
