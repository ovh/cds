package venom

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/mndrix/tap-go"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// Version of Venom
// One Line for this, used by release.sh script
// Keep "const Version on one line"
const Version = "0.15.0"

// PrintFunc used by venom to print output
var PrintFunc = fmt.Printf

var (
	executors = map[string]Executor{}
	contexts  = map[string]TestCaseContext{}
)

const (
	// ContextKey is key for Test Case Context. this
	// can be used by executors for getting context
	ContextKey = "tcContext"
)

// RegisterExecutor register Test Executors
func RegisterExecutor(name string, e Executor) {
	executors[name] = e
}

// WrapExecutor initializes a test by name
// no type -> exec is default
func WrapExecutor(t map[string]interface{}, tcc TestCaseContext) (*ExecutorWrap, error) {
	var name string
	var retry, delay, timeout int

	if itype, ok := t["type"]; ok {
		name = fmt.Sprintf("%s", itype)
	}

	if name == "" && tcc.GetName() != "default" {
		name = tcc.GetName()
	} else if name == "" {
		name = "exec"
	}

	retry, errRetry := getAttrInt(t, "retry")
	if errRetry != nil {
		return nil, errRetry
	}
	delay, errDelay := getAttrInt(t, "delay")
	if errDelay != nil {
		return nil, errDelay
	}
	timeout, errTimeout := getAttrInt(t, "timeout")
	if errTimeout != nil {
		return nil, errTimeout
	}

	if e, ok := executors[name]; ok {
		ew := &ExecutorWrap{
			executor: e,
			retry:    retry,
			delay:    delay,
			timeout:  timeout,
		}
		return ew, nil
	}

	return nil, fmt.Errorf("[%s] type '%s' is not implemented", tcc.GetName(), name)
}

// RegisterTestCaseContext new register TestCaseContext
func RegisterTestCaseContext(name string, tcc TestCaseContext) {
	contexts[name] = tcc
}

// ContextWrap initializes a context for a testcase
// no type -> parent context
func ContextWrap(tc *TestCase) (TestCaseContext, error) {
	if tc.Context == nil {
		return contexts["default"], nil
	}
	var typeName string
	if itype, ok := tc.Context["type"]; ok {
		typeName = fmt.Sprintf("%s", itype)
	}

	if typeName == "" {
		return contexts["default"], nil
	}
	contexts[typeName].SetTestCase(*tc)
	return contexts[typeName], nil
}

func getAttrInt(t map[string]interface{}, name string) (int, error) {
	var out int
	if i, ok := t[name]; ok {
		var ok bool
		out, ok = i.(int)
		if !ok {
			return -1, fmt.Errorf("attribute %s '%s' is not an integer", name, i)
		}
	}
	if out < 0 {
		out = 0
	}
	return out, nil
}

// Exit func display an error message on stderr and exit 1
func Exit(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

// OutputResult output result to sdtout, files...
func OutputResult(format string, resume, resumeFailures bool, outputDir string, tests Tests, elapsed time.Duration, detailsLevel string) error {
	var data []byte
	var err error
	switch format {
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

	if detailsLevel == "high" {
		PrintFunc(string(data))
	}

	if resume {
		outputResume(tests, elapsed, resumeFailures)
	}

	if outputDir != "" {
		filename := outputDir + "/" + "test_results" + "." + format
		if err := ioutil.WriteFile(filename, data, 0644); err != nil {
			return fmt.Errorf("Error while creating file %s, err:%s", filename, err)
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

func outputResume(tests Tests, elapsed time.Duration, resumeFailures bool) {

	if resumeFailures {
		for _, t := range tests.TestSuites {
			if t.Failures > 0 || t.Errors > 0 {
				PrintFunc("FAILED %s\n", t.Name)
				PrintFunc("--------------\n")

				for _, tc := range t.TestCases {
					for _, f := range tc.Failures {
						PrintFunc("%s\n", f.Value)
					}
					for _, f := range tc.Errors {
						PrintFunc("%s\n", f.Value)
					}
				}
				PrintFunc("-=-=-=-=-=-=-=-=-\n")
			}
		}
	}

	totalTestCases := 0
	totalTestSteps := 0
	for _, t := range tests.TestSuites {
		if t.Failures > 0 || t.Errors > 0 {
			PrintFunc("FAILED %s\n", t.Name)
		}
		totalTestCases += len(t.TestCases)
		for _, tc := range t.TestCases {
			totalTestSteps += len(tc.TestSteps)
		}
	}

	PrintFunc("Total:%d TotalOK:%d TotalKO:%d TotalSkipped:%d TotalTestSuite:%d TotalTestCase:%d TotalTestStep:%d Duration:%s\n",
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
