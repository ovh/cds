package sdk

import (
	"encoding/json"
	"encoding/xml"
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

// TestSuite is a single JUnit test suite which may contain many
// testcases.
type TestSuite struct {
	XMLName    xml.Name   `xml:"testsuite" json:"xmlName"`
	Disabled   int        `xml:"disabled,attr,omitempty" json:"disabled"`
	Errors     int        `xml:"errors,attr,omitempty" json:"errors"`
	Failures   int        `xml:"failures,attr,omitempty" json:"failures"`
	Hostname   string     `xml:"hostname,attr,omitempty" json:"hostname"`
	ID         string     `xml:"id,attr,omitempty" json:"iIDd"`
	Name       string     `xml:"name,attr" json:"name"`
	Package    string     `xml:"package,attr,omitempty" json:"package"`
	Properties []Property `xml:"properties,attr" json:"properties"`
	Skipped    int        `xml:"skipped,attr,omitempty" json:"skipped"`
	Total      int        `xml:"tests,attr" json:"total"`
	TestCases  []TestCase `xml:"testcase" json:"tests"`
	Time       string     `xml:"time,attr,omitempty" json:"time"`
	Timestamp  string     `xml:"timestamp,attr,omitempty" json:"timestamp"`
}

// Property represents a key/value pair used to define properties.
type Property struct {
	XMLName xml.Name `xml:"property" json:"xmlName"`
	Name    string   `xml:"name,attr" json:"name"`
	Value   string   `xml:"value,attr" json:"value"`
}

// TestCase is a single test case with its result.
type TestCase struct {
	XMLName    xml.Name    `xml:"testcase" json:"xmlName"`
	Assertions string      `xml:"assertions,attr,omitempty" json:"assertions"`
	Classname  string      `xml:"classname,attr,omitempty" json:"classname"`
	Errors     []Failure   `xml:"error,omitempty" json:"errors"`
	Failures   []Failure   `xml:"failure,omitempty" json:"failures"`
	Name       string      `xml:"name,attr" json:"name"`
	Skipped    int         `xml:"skipped,attr,omitempty" json:"skipped"`
	Status     string      `xml:"status,attr,omitempty" json:"status"`
	Systemout  InnerResult `xml:"system-out,omitempty" json:"systemout"`
	Systemerr  InnerResult `xml:"system-err,omitempty" json:"systemerr"`
	Time       string      `xml:"time,attr,omitempty" json:"time"`
}

// Failure contains data related to a failed test.
type Failure struct {
	Value   string `xml:",innerxml" json:"value"`
	Type    string `xml:"type,attr,omitempty" json:"type"`
	Message string `xml:"message,attr,omitempty" json:"message"`
}

// InnerResult is used by TestCase
type InnerResult struct {
	Value string `xml:",innerxml" json:"value"`
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
