package sdk

import (
	"encoding/json"
	"fmt"
)

// Tests contains all informations about tests in a pipeline build
type Tests struct {
	PipelineBuildID int64       `json:"pipeline_build_id"`
	Total           int         `json:"total"`
	TotalOK         int         `json:"ok"`
	TotalKO         int         `json:"ko"`
	TotalSkipped    int         `json:"skipped"`
	TestSuites      []TestSuite `xml:"testsuite" json:"test_suites"`
}

// TestSuite defines the result of a group of tests
type TestSuite struct {
	Name     string `xml:"name,attr" json:"name"`
	Total    int    `xml:"tests,attr" json:"total"`
	Failures int    `xml:"failures,attr" json:"failures"`
	Errors   int    `xml:"errors,attr" json:"errors"`
	Skip     int    `xml:"skip,attr" json:"skipped"`
	Tests    []Test `xml:"testcase" json:"tests"`
}

// Test define a single test
type Test struct {
	Name    string  `xml:"name,attr" json:"name"`
	Time    string  `xml:"time,attr" json:"time"`
	Failure string  `xml:"failure" json:"failure"`
	Error   string  `xml:"error" json:"error"`
	Skip    *string `xml:"skipped" json:"skipped"`
}

// GetTestResults retrieves tests results for a specific build
func GetTestResults(proj, app, pip, env string, bn int) (Tests, error) {
	if env == "" {
		env = DefaultEnv.Name
	}
	uri := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/build/%d/test?env=%s", proj, app, pip, bn, env)
	var t Tests

	data, code, err := Request("GET", uri, nil)
	if err != nil {
		return t, err
	}
	if code > 300 {
		return t, fmt.Errorf("HTTP %d", code)
	}

	err = json.Unmarshal([]byte(data), &t)
	if err != nil {
		return t, err
	}

	return t, nil
}
