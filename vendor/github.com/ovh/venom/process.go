package venom

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

func (v *Venom) init() error {
	v.testsuites = []TestSuite{}
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
		log.Debug("Detail Level: ", v.OutputDetails)
	default:
		return errors.New("Invalid details. Must be low, medium or high")
	}

	return nil
}

// Parse parses tests suite to check context and variables
func (v *Venom) Parse(path []string, exclude []string) error {
	if err := v.init(); err != nil {
		return err
	}

	filesPath, err := getFilesPath(path, exclude)
	if err != nil {
		return err
	}

	if err := v.readFiles(filesPath); err != nil {
		return err
	}

	missingVars := []string{}
	extractedVars := []string{}
	for i := range v.testsuites {
		ts := &v.testsuites[i]
		log.Info("Parsing testsuite", ts.Package)

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
		var varExtracted bool
		for _, e := range extractedVars {
			if strings.HasPrefix(k, e) {
				varExtracted = true
			}
		}
		if !varExtracted {
			var ignored bool
			for _, i := range v.IgnoreVariables {
				if strings.HasPrefix(k, i) {
					ignored = true
				}
			}
			if !ignored {
				reallyMissingVars = append(reallyMissingVars, k)
			}
		}
	}

	if len(reallyMissingVars) > 0 {
		return fmt.Errorf("Missing variables %v", reallyMissingVars)
	}

	return nil
}

// Process runs tests suite and return a Tests result
func (v *Venom) Process(path []string, exclude []string) (*Tests, error) {
	if err := v.init(); err != nil {
		return nil, err
	}

	filesPath, err := getFilesPath(path, exclude)
	if err != nil {
		return nil, err
	}

	if err := v.readFiles(filesPath); err != nil {
		return nil, err
	}

	if v.OutputDetails != DetailsLow {
		pool := v.initBars()
		defer endBars(v.OutputDetails, pool)
	}

	chanEnd := make(chan *TestSuite, 1)
	parallels := make(chan *TestSuite, v.Parallel) //Run testsuite in parrallel
	wg := sync.WaitGroup{}
	testsResult := &Tests{}

	wg.Add(len(filesPath))
	chanToRun := make(chan *TestSuite, len(filesPath)+1)

	go v.computeStats(testsResult, chanEnd, &wg)
	go func() {
		for ts := range chanToRun {
			parallels <- ts
			go func(ts *TestSuite) {
				v.runTestSuite(ts)
				chanEnd <- ts
				<-parallels
			}(ts)
		}
	}()

	for i := range v.testsuites {
		chanToRun <- &v.testsuites[i]
	}

	wg.Wait()

	return testsResult, nil
}

func (v *Venom) computeStats(testsResult *Tests, chanEnd <-chan *TestSuite, wg *sync.WaitGroup) {
	for t := range chanEnd {
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
		wg.Done()
	}
}

func rightPad(s string, padStr string, pLen int) string {
	o := s + strings.Repeat(padStr, pLen)
	return o[0:pLen]
}
