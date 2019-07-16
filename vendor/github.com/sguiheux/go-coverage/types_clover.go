package coverage

import "encoding/xml"

type CloverCoverage struct {
	XMLName   xml.Name      `xml:"coverage"`
	Clover    string        `xml:"clover,attr"`
	Generated int64         `xml:"generated,attr"`
	Project   CloverProject `xml:"project"`
}

type CloverProject struct {
	Name      string          `xml:"name,attr"`
	Timestamp int64           `xml:"timestamp,attr"`
	Metrics   CloverMetric    `xml:"metrics"`
	Package   []CloverPackage `xml:"package"`
}

type CloverPackage struct {
	Name    string               `xml:"name,attr"`
	Metrics CloverPackageMetrics `xml:"metrics"`
	File    []CloverFile         `xml:"file"`
}

type CloverFile struct {
	Name    string            `xml:"name,attr"`
	Path    string            `xml:"path,attr"`
	Metrics CloverFileMetrics `xml:"metrics"`
	Class   []CloverClass     `xml:"class"`
	Line    []CloverLine      `xml:"line"`
}

type CloverClass struct {
	Name    string             `xml:"name,attr"`
	Metrics CloverClassMetrics `xml:"metrics"`
}

type CloverLine struct {
	Num          int64   `xml:"num,attr"`
	Type         string  `xml:"type,attr"`
	Complexity   int64   `xml:"complexity,attr"`
	Count        int64   `xml:"count,attr"`
	FalseCount   int64   `xml:"falsecount,attr"`
	TrueCount    int64   `xml:"truecount,attr"`
	Signature    int64   `xml:"signature,attr"`
	TestDuration float64 `xml:"testduration,attr"`
	TestSuccess  bool    `xml:"testsuccess,attr"`
	Visibility   string  `xml:"visibility,attr"`
}

type CloverMetric struct {
	CloverPackageMetrics
	Packages int64 `xml:"packages,attr"`
}

type CloverPackageMetrics struct {
	CloverFileMetrics
	Files int64 `xml:"files,attr"`
}

type CloverFileMetrics struct {
	CloverClassMetrics
	Classes int64 `xml:"classes,attr"`
	Loc     int64 `xml:"loc,attr"`
	Ncloc   int64 `xml:"ncloc,attr"`
}

type CloverClassMetrics struct {
	Complexity          int64   `xml:"complexity,attr"`
	Elements            int64   `xml:"elements,attr"`
	CoveredElements     int64   `xml:"coveredelements,attr"`
	Conditionnals       int64   `xml:"conditionals,attr"`
	CoveredConditionals int64   `xml:"coveredconditionals,attr"`
	Statements          int64   `xml:"statements,attr"`
	CoveredStatements   int64   `xml:"coveredstatements,attr"`
	CoveredMethods      int64   `xml:"coveredmethods,attr"`
	Methods             int64   `xml:"methods,attr"`
	TestDuration        float64 `xml:"testduration,attr"`
	TestFailures        int64   `xml:"testfailures,attr"`
	TestPasses          int64   `xml:"testpasses,attr"`
	TestRuns            int64   `xml:"testruns,attr"`
}
