package coverage

import "encoding/xml"

type CoberturaCoverage struct {
	XMLName         xml.Name          `xml:"coverage"`
	LineRate        string            `xml:"line-rate,attr"`
	BranchRate      string            `xml:"branch-rate,attr"`
	LinesCovered    string            `xml:"lines-covered,attr"`
	LinesValid      string            `xml:"lines-valid,attr"`
	BranchesCovered string            `xml:"branches-covered,attr"`
	BranchesValid   string            `xml:"branches-valid,attr"`
	Complexity      string            `xml:"complexity,attr"`
	Version         string            `xml:"version,attr"`
	Timestamp       string            `xml:"timestamp,attr"`
	Sources         CoberturaSources  `xml:"sources"`
	Packages        CoberturaPackages `xml:"packages"`
}

type CoberturaSources struct {
	Source []string `xml:"source"`
}

type CoberturaPackages struct {
	Package []CoberturaPackage `xml:"package"`
}

type CoberturaPackage struct {
	Name       string           `xml:"name,attr"`
	LineRate   string           `xml:"line-rate,attr"`
	BranchRate string           `xml:"branch-rate,attr"`
	Complexity string           `xml:"complexity,attr"`
	Classes    CoberturaClasses `xml:"classes"`
}

type CoberturaClasses struct {
	Class []CoberturaClass `xml:"class"`
}

type CoberturaClass struct {
	Name       string           `xml:"name,attr"`
	FileName   string           `xml:"filename,attr"`
	LineRate   string           `xml:"line-rate,attr"`
	BranchRate string           `xml:"branch-rate,attr"`
	Complexity string           `xml:"complexity,attr"`
	Methods    CoberturaMethods `xml:"methods"`
	Lines      CoberturaLines   `xml:"lines"`
}

type CoberturaMethods struct {
	Method []CoberturaMethod `xml:"method"`
}

type CoberturaMethod struct {
	Name       string         `xml:"name,attr"`
	Signature  string         `xml:"signature,attr"`
	LineRate   string         `xml:"line-rate,attr"`
	BranchRate string         `xml:"branch-rate,attr"`
	Lines      CoberturaLines `xml:"lines"`
}

type CoberturaLines struct {
	Line []CoberturaLine `xml:"line"`
}

type CoberturaLine struct {
	Number            string `xml:"number,attr"`
	Hits              string `xml:"hits,attr"`
	Branch            string `xml:"branch,attr"`
	ConditionCoverage string `xml:"condition-coverage,attr"`
	Conditions        []CoberturaConditions
}

type CoberturaConditions struct {
	Condition []CoberturaCondition `xml:"conditions,attr"`
}

type CoberturaCondition struct {
	Number   string `xml:"number,attr"`
	Type     string `xml:"type,attr"`
	Coverage string `xml:"coverage,attr"`
}
