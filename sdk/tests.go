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
	XMLName    xml.Name   `xml:"testsuite" json:"xmlName" yaml:"-"`
	Disabled   int        `xml:"disabled,attr,omitempty" json:"disabled" yaml:"-"`
	Errors     int        `xml:"errors,attr,omitempty" json:"errors" yaml:"-"`
	Failures   int        `xml:"failures,attr,omitempty" json:"failures" yaml:"-"`
	Hostname   string     `xml:"hostname,attr,omitempty" json:"hostname" yaml:"-"`
	ID         string     `xml:"id,attr,omitempty" json:"id" yaml:"-"`
	Name       string     `xml:"name,attr" json:"name" yaml:"name"`
	Package    string     `xml:"package,attr,omitempty" json:"package" yaml:"-"`
	Properties []Property `xml:"properties,attr" json:"properties" yaml:"-"`
	Skipped    int        `xml:"skipped,attr,omitempty" json:"skipped" yaml:"skipped,omitempty"`
	Total      int        `xml:"tests,attr" json:"total" yaml:"total,omitempty"`
	TestCases  []TestCase `xml:"testcase" json:"tests" yaml:"testcases"`
	Time       string     `xml:"time,attr,omitempty" json:"time" yaml:"-"`
	Timestamp  string     `xml:"timestamp,attr,omitempty" json:"timestamp" yaml:"-"`
}

// Property represents a key/value pair used to define properties.
type Property struct {
	XMLName xml.Name `xml:"property" json:"xmlName" yaml:"-"`
	Name    string   `xml:"name,attr" json:"name" yaml:"-"`
	Value   string   `xml:"value,attr" json:"value" yaml:"-"`
}

// TestCase is a single test case with its result.
type TestCase struct {
	XMLName    xml.Name    `xml:"testcase" json:"xmlName" yaml:"-"`
	Assertions string      `xml:"assertions,attr,omitempty" json:"assertions" yaml:"-"`
	Classname  string      `xml:"classname,attr,omitempty" json:"classname" yaml:"-"`
	Errors     []Failure   `xml:"error,omitempty" json:"errors" yaml:"errors,omitempty"`
	Failures   []Failure   `xml:"failure,omitempty" json:"failures" yaml:"failures,omitempty"`
	Name       string      `xml:"name,attr" json:"name" yaml:"name"`
	Skipped    int         `xml:"skipped,attr,omitempty" json:"skipped" yaml:"skipped,omitempty"`
	Status     string      `xml:"status,attr,omitempty" json:"status" yaml:"status,omitempty"`
	Systemout  InnerResult `xml:"system-out,omitempty" json:"systemout" yaml:"systemout,omitempty"`
	Systemerr  InnerResult `xml:"system-err,omitempty" json:"systemerr" yaml:"systemerr,omitempty"`
	Time       string      `xml:"time,attr,omitempty" json:"time" yaml:"time,omitempty"`
	TestSteps  []TestStep  `xml:"-" json:"steps" yaml:"steps"`
}

// TestStep contains command to execute, type: exec, http, etc...
type TestStep struct {
	Type string `xml:"-" json:"type,omitempty" yaml:"type,omitempty"`

	// binary
	Command string   `xml:"-" json:"command,omitempty" yaml:"command,omitempty"`
	Args    []string `xml:"-" json:"args,omitempty" yaml:"args,omitempty"`
	StdIn   string   `xml:"-" json:"stdin,omitempty" yaml:"stdin,omitempty"`

	// HTTP
	Method     string         `xml:"-" json:"method,omitempty" yaml:"method,omitempty"`
	URL        string         `xml:"-" json:"url,omitempty" yaml:"url,omitempty"`
	Payload    string         `xml:"-" json:"playload,omitempty" yaml:"playload,omitempty"`
	Assertions []string       `xml:"-" json:"assertions,omitempty" yaml:"assertions,omitempty"`
	Result     TestStepResult `xml:"-" json:"result,omitempty" yaml:"result,omitempty"`
}

// DetectType returns type of testStep if testStep.Type is not empty.
// If testStep.Type is empty, return http is URL is not empty
// of return exec is command is not empty
func (t *TestStep) DetectType() (string, error) {
	if t.Type != "" {
		return t.Type, nil
	}
	if t.URL != "" {
		return "http", nil
	}
	if t.Command != "" {
		return "exec", nil
	}

	return "", fmt.Errorf("Type is invalid")
}

// TestStepResult represents a step result
type TestStepResult struct {
	StdOut string `xml:"-" json:"stdout,omitempty" yaml:"stdout,omitempty"`
	StdErr string `xml:"-" json:"stderr,omitempty" yaml:"stderr,omitempty"`
	Err    error  `xml:"-" json:"error,omitempty" yaml:"error,omitempty"`
	Code   string `xml:"-" json:"code,omitempty" yaml:"code,omitempty"`
}

// Failure contains data related to a failed test.
type Failure struct {
	Value   string `xml:",innerxml" json:"value" yaml:"value,omitempty"`
	Type    string `xml:"type,attr,omitempty" json:"type" yaml:"type,omitempty"`
	Message string `xml:"message,attr,omitempty" json:"message" yaml:"message,omitempty"`
}

// InnerResult is used by TestCase
type InnerResult struct {
	Value string `xml:",innerxml" json:"value" yaml:"value"`
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
