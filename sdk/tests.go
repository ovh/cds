package sdk

import (
	"encoding/xml"
	"fmt"
)

type TestsResults struct {
	JUnitTestsSuites
	TestsStats
}

type TestsStats struct {
	Total        int `json:"total,omitempty" mapstructure:"total"`
	TotalOK      int `json:"ok,omitempty" mapstructure:"ok"`
	TotalKO      int `json:"ko,omitempty" mapstructure:"ko"`
	TotalSkipped int `json:"skipped,omitempty" mapstructure:"skipped"`
}

type JUnitTestsSuites struct {
	XMLName    xml.Name         `xml:"testsuites" json:"-"`
	TestSuites []JUnitTestSuite `xml:"testsuite" json:"test_suites" mapstructure:"test_suites"`
}

// EnsureData add missing names on test cases and suites also compute
// test suites total values from test cases data.
func (s JUnitTestsSuites) EnsureData() JUnitTestsSuites {
	cleaned := s

	// Add names if missing
	for i := range cleaned.TestSuites {
		if cleaned.TestSuites[i].Name == "" {
			cleaned.TestSuites[i].Name = fmt.Sprintf("TestSuite.%d", i)
		}
		for j := range cleaned.TestSuites[i].TestCases {
			if cleaned.TestSuites[i].TestCases[j].Name == "" {
				cleaned.TestSuites[i].TestCases[j].Name = fmt.Sprintf("TestCase.%d", j)
			}
		}
	}

	// Validate total values for test suites
	for i, ts := range cleaned.TestSuites {
		var nSkipped, nFailures, nErrors int
		for _, tc := range cleaned.TestSuites[i].TestCases {
			// For a test case we should only increment one counter
			if len(tc.Errors) > 0 {
				nErrors++
			} else if len(tc.Failures) > 0 {
				nFailures++
			} else if len(tc.Skipped) > 0 {
				nSkipped++
			}
		}
		cleaned.TestSuites[i].Skipped = nSkipped
		cleaned.TestSuites[i].Failures = nFailures
		cleaned.TestSuites[i].Errors = nErrors
		cleaned.TestSuites[i].Total = len(ts.TestCases)
	}

	return cleaned
}

func (s JUnitTestsSuites) ComputeStats() TestsStats {
	var stats TestsStats
	for _, ts := range s.TestSuites {
		stats.Total += ts.Total
		stats.TotalKO += ts.Failures + ts.Errors
		stats.TotalSkipped += ts.Skipped
		stats.TotalOK += ts.Total - (ts.Failures + ts.Errors + ts.Skipped)
	}
	return stats
}

type JUnitTestSuite struct {
	XMLName   xml.Name        `xml:"testsuite" json:"-"`
	Disabled  int             `xml:"disabled,attr,omitempty" json:"disabled,omitempty" mapstructure:"disabled"`
	Errors    int             `xml:"errors,attr,omitempty" json:"errors,omitempty" mapstructure:"errors"`
	Failures  int             `xml:"failures,attr,omitempty" json:"failures,omitempty" mapstructure:"failures"`
	ID        string          `xml:"id,attr" json:"id,omitempty" mapstructure:"id"`
	Name      string          `xml:"name,attr" json:"name,omitempty" mapstructure:"name"`
	Package   string          `xml:"package,attr,omitempty" json:"package,omitempty" mapstructure:"package"`
	Skipped   int             `xml:"skipped,attr,omitempty" json:"skipped,omitempty" mapstructure:"skipped"`
	TestCases []JUnitTestCase `xml:"testcase" json:"tests,omitempty" mapstructure:"tests"`
	Time      string          `xml:"time,attr,omitempty" json:"time,omitempty" mapstructure:"time"`
	Timestamp string          `xml:"timestamp,attr,omitempty" json:"timestamp,omitempty" mapstructure:"timestamp"`
	Total     int             `xml:"tests,attr" json:"total,omitempty" mapstructure:"total"`
}

type JUnitTestCase struct {
	XMLName   xml.Name           `xml:"testcase" json:"-"`
	Classname string             `xml:"classname,attr,omitempty" json:"classname,omitempty" mapstructure:"classname"`
	Errors    []JUnitTestFailure `xml:"error,omitempty" json:"errors,omitempty" mapstructure:"errors"`
	Failures  []JUnitTestFailure `xml:"failure,omitempty" json:"failures,omitempty" mapstructure:"failures"`
	Name      string             `xml:"name,attr" json:"name,omitempty" mapstructure:"name"`
	Skipped   []JUnitTestSkipped `xml:"skipped,omitempty" json:"skipped,omitempty" mapstructure:"skipped"`
	Status    string             `xml:"status,attr,omitempty" json:"status,omitempty" mapstructure:"status"`
	Systemerr JUnitInnerResult   `xml:"system-err,omitempty" json:"systemerr,omitempty" mapstructure:"systemerr"`
	Systemout JUnitInnerResult   `xml:"system-out,omitempty" json:"systemout,omitempty" mapstructure:"systemout"`
	Time      string             `xml:"time,attr,omitempty" json:"time,omitempty" mapstructure:"time"`
}

type JUnitTestSkipped struct {
	Message string `xml:"message,attr,omitempty" json:"message,omitempty" mapstructure:"message"`
	Value   string `xml:",cdata" json:"value,omitempty" mapstructure:"value"`
}

type JUnitTestFailure struct {
	Message string `xml:"message,attr,omitempty" json:"message,omitempty" mapstructure:"message"`
	Type    string `xml:"type,attr,omitempty" json:"type,omitempty" mapstructure:"type"`
	Value   string `xml:",cdata" json:"value,omitempty" mapstructure:"value"`
}

type JUnitInnerResult struct {
	Value string `xml:",cdata" json:"value,omitempty" mapstructure:"value"`
}
