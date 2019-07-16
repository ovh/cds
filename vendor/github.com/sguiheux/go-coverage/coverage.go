package coverage

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

// New creates a new lcov parser
func New(filePath string, mode CoverageMode) Parser {
	return Parser{
		path: filePath,
		mode: mode,
	}
}

// Parse parses the lcov file
func (l Parser) Parse() (Report, error) {
	switch l.mode {
	case LCOV:
		return l.processLcov()
	case COBERTURA:
		return l.processCobertura()
	case CLOVER:
		return l.processClover()
	}
	return Report{}, fmt.Errorf("coverage.parse> Unknown mode %s", l.mode)
}

func (l Parser) processClover() (Report, error) {
	file, errF := os.Open(l.path)
	if errF != nil {
		return Report{}, fmt.Errorf("coverage.processClover> Unable to open file: %v", errF)
	}
	defer file.Close()

	b, errR := ioutil.ReadAll(file)
	if errR != nil {
		return Report{}, fmt.Errorf("coverage.processClover> Unable to read file: %v", errR)
	}

	var cloReport CloverCoverage
	if err := xml.Unmarshal(b, &cloReport); err != nil {
		return Report{}, fmt.Errorf("coverage.processClover> Unable to unmarshal content: %v", err)
	}

	report := Report{
		CoveredBranches:  int(cloReport.Project.Metrics.CoveredConditionals),
		CoveredFunctions: int(cloReport.Project.Metrics.CoveredMethods),
		CoveredLines:     int(cloReport.Project.Metrics.CoveredStatements),
		TotalBranches:    int(cloReport.Project.Metrics.Conditionnals),
		TotalFunctions:   int(cloReport.Project.Metrics.Methods),
		TotalLines:       int(cloReport.Project.Metrics.Statements),
		Files:            make([]FileReport, 0, cloReport.Project.Metrics.Files),
	}

	for _, p := range cloReport.Project.Package {
		for _, f := range p.File {
			fp := FileReport{
				TotalLines:       int(f.Metrics.Statements),
				TotalFunctions:   int(f.Metrics.Methods),
				TotalBranches:    int(f.Metrics.Conditionnals),
				CoveredLines:     int(f.Metrics.CoveredStatements),
				CoveredFunctions: int(f.Metrics.CoveredMethods),
				CoveredBranches:  int(f.Metrics.CoveredConditionals),
				Path:             f.Path,
			}
			report.Files = append(report.Files, fp)
		}
	}
	return report, nil

}

func (l Parser) processCobertura() (Report, error) {
	file, errF := os.Open(l.path)
	if errF != nil {
		return Report{}, fmt.Errorf("coverage.processCobertura> Unable to open file: %v", errF)
	}
	defer file.Close()

	b, errR := ioutil.ReadAll(file)
	if errR != nil {
		return Report{}, fmt.Errorf("coverage.processCobertura> Unable to read file: %v", errR)
	}

	var cobReport CoberturaCoverage
	if err := xml.Unmarshal(b, &cobReport); err != nil {
		return Report{}, fmt.Errorf("coverage.processCobertura> Unable to unmarshal content: %v", err)
	}

	report := Report{
		TotalLines:      getInt(cobReport.LinesValid),
		TotalBranches:   getInt(cobReport.BranchesValid),
		CoveredLines:    getInt(cobReport.LinesCovered),
		CoveredBranches: getInt(cobReport.BranchesCovered),
	}
	return report, nil
}

func (l Parser) processLcov() (Report, error) {
	file, errF := os.Open(l.path)
	if errF != nil {
		return Report{}, fmt.Errorf("coverage.processLcov> Unable to open lcov file: %v", errF)
	}
	defer file.Close()

	r := bufio.NewReader(file)

	report := Report{
		Files: make([]FileReport, 0),
	}
	fileReport := FileReport{}
	for {
		line, errR := r.ReadString('\n')
		if errR != nil && errR != io.EOF {
			return report, fmt.Errorf("coverage.processLcov> Unable to read line: %v", errR)
		}
		if errR == io.EOF {
			break
		}
		line = strings.Replace(line, "\n", "", -1)

		// Test new file
		if strings.HasPrefix(line, "SF:") {
			if fileReport.Path != "" {
				report.Files = append(report.Files, fileReport)
			}
			fileReport = FileReport{
				Path: strings.Replace(line, "SF:", "", 1),
			}
		} else {
			l.processLcovLine(line, &report, &fileReport)
		}

	}
	return report, nil
}

func (l Parser) processLcovLine(line string, report *Report, fileReport *FileReport) {
	switch {
	case strings.HasPrefix(line, "FNF:"):
		nb := getInt(strings.Replace(line, "FNF:", "", -1))
		fileReport.TotalFunctions = nb
		report.TotalFunctions += nb
	case strings.HasPrefix(line, "FNH:"):
		nb := getInt(strings.Replace(line, "FNH:", "", -1))
		fileReport.CoveredFunctions = nb
		report.CoveredFunctions += nb
	case strings.HasPrefix(line, "BRF:"):
		nb := getInt(strings.Replace(line, "BRF:", "", -1))
		fileReport.TotalBranches = nb
		report.TotalBranches += nb
	case strings.HasPrefix(line, "BRH:"):
		nb := getInt(strings.Replace(line, "BRH:", "", -1))
		fileReport.CoveredBranches = nb
		report.CoveredBranches += nb
	case strings.HasPrefix(line, "LF:"):
		nb := getInt(strings.Replace(line, "LF:", "", -1))
		fileReport.TotalLines = nb
		report.TotalLines += nb
	case strings.HasPrefix(line, "LH:"):
		nb := getInt(strings.Replace(line, "LH:", "", -1))
		fileReport.CoveredLines = nb
		report.CoveredLines += nb
	}
}

func getInt(s string) int {
	i := 0
	i, _ = strconv.Atoi(s)
	return i
}
