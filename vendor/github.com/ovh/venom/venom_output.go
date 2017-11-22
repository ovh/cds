package venom

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"time"

	tap "github.com/mndrix/tap-go"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

// OutputResult output result to sdtout, files...
func (v *Venom) OutputResult(tests Tests, elapsed time.Duration) error {
	var data []byte
	var err error
	switch v.OutputFormat {
	case "json":
		data, err = json.MarshalIndent(tests, "", "  ")
		if err != nil {
			log.Fatalf("Error: cannot format output json (%s)", err)
		}
	case "tap":
		data, err = outputTapFormat(tests)
		if err != nil {
			log.Fatalf("Error: cannot format output tap (%s)", err)
		}
	case "yml", "yaml":
		data, err = yaml.Marshal(tests)
		if err != nil {
			log.Fatalf("Error: cannot format output yaml (%s)", err)
		}
	default:
		dataxml, errm := xml.MarshalIndent(tests, "", "  ")
		if errm != nil {
			log.Fatalf("Error: cannot format xml output: %s", errm)
		}
		data = append([]byte(`<?xml version="1.0" encoding="utf-8"?>\n`), dataxml...)
	}

	if v.OutputDetails == "high" {
		v.PrintFunc(string(data))
	}

	if v.OutputResume {
		v.outputResume(tests, elapsed)
	}

	if v.OutputDir != "" {
		filename := v.OutputDir + "/" + "test_results" + "." + v.OutputFormat
		if err := ioutil.WriteFile(filename, data, 0644); err != nil {
			return fmt.Errorf("Error while creating file %s: %v", filename, err)
		}
	}
	return nil
}

func outputTapFormat(tests Tests) ([]byte, error) {
	t := tap.New()
	buf := new(bytes.Buffer)
	t.Writer = buf
	t.Header(tests.Total)
	for _, ts := range tests.TestSuites {
		for _, tc := range ts.TestCases {
			name := ts.Name + " / " + tc.Name
			if len(tc.Skipped) > 0 {
				t.Skip(1, name)
				continue
			}

			if len(tc.Errors) > 0 {
				t.Fail(name)
				for _, e := range tc.Errors {
					t.Diagnosticf("Error: %s", e.Value)
				}
				continue
			}

			if len(tc.Failures) > 0 {
				t.Fail(name)
				for _, e := range tc.Failures {
					t.Diagnosticf("Failure: %s", e.Value)
				}
				continue
			}

			t.Pass(name)
		}
	}

	return buf.Bytes(), nil
}

func (v *Venom) outputResume(tests Tests, elapsed time.Duration) {
	if v.OutputResumeFailures {
		for _, t := range tests.TestSuites {
			if t.Failures > 0 || t.Errors > 0 {
				v.PrintFunc("FAILED %s\n", t.Name)
				v.PrintFunc("--------------\n")

				for _, tc := range t.TestCases {
					for _, f := range tc.Failures {
						v.PrintFunc("%s\n", f.Value)
					}
					for _, f := range tc.Errors {
						v.PrintFunc("%s\n", f.Value)
					}
				}
				v.PrintFunc("-=-=-=-=-=-=-=-=-\n\n")
			}
		}
	}

	totalTestCases := 0
	totalTestSteps := 0
	for _, t := range tests.TestSuites {
		if t.Failures > 0 || t.Errors > 0 {
			v.PrintFunc("FAILED %s\n", t.Name)
		}
		totalTestCases += len(t.TestCases)
		for _, tc := range t.TestCases {
			totalTestSteps += len(tc.TestSteps)
		}
	}

	v.PrintFunc("Total:%d TotalOK:%d TotalKO:%d TotalSkipped:%d TotalTestSuite:%d TotalTestCase:%d TotalTestStep:%d Duration:%s\n",
		tests.Total,
		tests.TotalOK,
		tests.TotalKO,
		tests.TotalSkipped,
		len(tests.TestSuites),
		totalTestCases,
		totalTestSteps,
		elapsed,
	)
}
