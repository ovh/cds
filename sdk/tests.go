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
	Total        int `json:"total,omitempty"`
	TotalOK      int `json:"ok,omitempty"`
	TotalKO      int `json:"ko,omitempty"`
	TotalSkipped int `json:"skipped,omitempty"`
}

type JUnitTestsSuites struct {
	XMLName    xml.Name         `xml:"testsuites" json:"-"`
	TestSuites []JUnitTestSuite `xml:"testsuite" json:"test_suites"`
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
	Disabled  int             `xml:"disabled,attr,omitempty" json:"disabled,omitempty"`
	Errors    int             `xml:"errors,attr,omitempty" json:"errors,omitempty"`
	Failures  int             `xml:"failures,attr,omitempty" json:"failures,omitempty"`
	ID        string          `xml:"id,attr" json:"id,omitempty"`
	Name      string          `xml:"name,attr" json:"name,omitempty"`
	Package   string          `xml:"package,attr,omitempty" json:"package,omitempty"`
	Skipped   int             `xml:"skipped,attr,omitempty" json:"skipped,omitempty"`
	TestCases []JUnitTestCase `xml:"testcase" json:"tests,omitempty"`
	Time      string          `xml:"time,attr,omitempty" json:"time,omitempty"`
	Timestamp string          `xml:"timestamp,attr,omitempty" json:"timestamp,omitempty"`
	Total     int             `xml:"tests,attr" json:"total,omitempty"`
}

type JUnitTestCase struct {
	XMLName   xml.Name           `xml:"testcase" json:"-"`
	Classname string             `xml:"classname,attr,omitempty" json:"classname,omitempty"`
	Errors    []JUnitTestFailure `xml:"error,omitempty" json:"errors,omitempty"`
	Failures  []JUnitTestFailure `xml:"failure,omitempty" json:"failures,omitempty"`
	Name      string             `xml:"name,attr" json:"name,omitempty"`
	Skipped   []JUnitTestSkipped `xml:"skipped,omitempty" json:"skipped,omitempty"`
	Status    string             `xml:"status,attr,omitempty" json:"status,omitempty"`
	Systemerr JUnitInnerResult   `xml:"system-err,omitempty" json:"systemerr,omitempty"`
	Systemout JUnitInnerResult   `xml:"system-out,omitempty" json:"systemout,omitempty"`
	Time      string             `xml:"time,attr,omitempty" json:"time,omitempty"`
}

type JUnitTestSkipped struct {
	Message string `xml:"message,attr,omitempty" json:"message,omitempty"`
	Value   string `xml:",cdata" json:"value,omitempty"`
}

type JUnitTestFailure struct {
	Message string `xml:"message,attr,omitempty" json:"message,omitempty"`
	Type    string `xml:"type,attr,omitempty" json:"type,omitempty"`
	Value   string `xml:",cdata" json:"value,omitempty"`
}

type JUnitInnerResult struct {
	Value string `xml:",cdata" json:"value,omitempty"`
}
