package venom

import (
	"errors"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Process runs tests suite and return a Tests result
func (v *Venom) Process(path []string, exclude []string) (*Tests, error) {
	switch v.LogLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "error":
		log.SetLevel(log.WarnLevel)
	default:
		log.SetLevel(log.WarnLevel)
	}

	log.SetOutput(v.LogOutput)
	switch v.OutputDetails {
	case DetailsLow, DetailsMedium, DetailsHigh:
		log.Infof("Detail Level: %s", v.OutputDetails)
	default:
		return nil, errors.New("Invalid details. Must be low, medium or high")
	}

	filesPath := getFilesPath(path, exclude)

	if err := v.readFiles(filesPath); err != nil {
		return nil, err
	}

	//First parse the testsuites to check if all testsuites are fine (context init + variable usages)
	if err := v.parse(); err != nil {
		return nil, err
	}

	//Then process
	if v.OutputDetails != DetailsLow {
		pool := v.initBars()
		defer endBars(v.OutputDetails, pool)
	}

	for i := range v.testsuites {
		ts := &v.testsuites[i]
		v.runTestSuite(ts)
	}

	testsResult := &Tests{}
	v.computeStats(testsResult)

	return testsResult, nil
}

func (v *Venom) parse() error {
	missingVars := []string{}
	extractedVars := []string{}
	for i := range v.testsuites {
		ts := &v.testsuites[i]
		log.Info("Parsing testsuite %s", ts.Package)

		tvars, textractedVars, err := v.parseTestSuite(ts)
		if err != nil {
			return err
		}
		for _, k := range tvars {
			var found bool
			for i := 0; i < len(missingVars); i++ {
				if missingVars[i] == k {
					found = true
					break
				}
			}
			if !found {
				missingVars = append(missingVars, k)
			}
		}
		for _, k := range textractedVars {
			var found bool
			for i := 0; i < len(extractedVars); i++ {
				if extractedVars[i] == k {
					found = true
					break
				}
			}
			if !found {
				extractedVars = append(extractedVars, k)
			}
		}
	}

	reallyMissingVars := []string{}
	for _, k := range missingVars {
		log.Debugf("Checking variable %s", k)
		var varExtracted bool
		for _, e := range extractedVars {
			if k == e {
				varExtracted = true
			}
		}
		if !varExtracted {
			reallyMissingVars = append(reallyMissingVars, k)
		}
	}

	if len(reallyMissingVars) > 0 {
		return fmt.Errorf("Missing variables %v", reallyMissingVars)
	}

	return nil
}

func (v *Venom) computeStats(testsResult *Tests) {
	for i := range v.testsuites {
		t := &v.testsuites[i]
		testsResult.TestSuites = append(testsResult.TestSuites, *t)
		if t.Failures > 0 {
			testsResult.TotalKO += t.Failures
		} else {
			testsResult.TotalOK += len(t.TestCases) - t.Failures
		}
		if t.Skipped > 0 {
			testsResult.TotalSkipped += t.Skipped
		}

		testsResult.Total = testsResult.TotalKO + testsResult.TotalOK + testsResult.TotalSkipped
	}
}

func rightPad(s string, padStr string, pLen int) string {
	o := s + strings.Repeat(padStr, pLen)
	return o[0:pLen]
}
