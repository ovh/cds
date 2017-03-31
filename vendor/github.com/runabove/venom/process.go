package venom

import (
	"errors"
	"strings"
	"sync"

	log "github.com/Sirupsen/logrus"
)

// Process runs tests suite and return a Tests result
func Process(path []string, variables map[string]string, exclude []string, parallel int, logLevel string, detailsLevel string) (*Tests, error) {

	switch logLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "error":
		log.SetLevel(log.WarnLevel)
	default:
		log.SetLevel(log.WarnLevel)
	}

	switch detailsLevel {
	case DetailsLow, DetailsMedium, DetailsHigh:
		log.Infof("Detail Level: %s", detailsLevel)
	default:
		return nil, errors.New("Invalid details. Must be low, medium or high")
	}

	chanEnd := make(chan TestSuite, 1)
	parallels := make(chan TestSuite, parallel)
	wg := sync.WaitGroup{}
	testsResult := &Tests{}

	filesPath := getFilesPath(path, exclude)
	wg.Add(len(filesPath))
	chanToRun := make(chan TestSuite, len(filesPath)+1)

	go computeStats(testsResult, chanEnd, &wg)

	bars := readFiles(variables, detailsLevel, filesPath, chanToRun)

	pool := initBars(detailsLevel, bars)

	go func() {
		for ts := range chanToRun {
			parallels <- ts
			go func(ts TestSuite) {
				runTestSuite(&ts, bars, detailsLevel)
				chanEnd <- ts
				<-parallels
			}(ts)
		}
	}()

	wg.Wait()

	endBars(detailsLevel, pool)

	return testsResult, nil
}

func computeStats(testsResult *Tests, chanEnd <-chan TestSuite, wg *sync.WaitGroup) {
	for t := range chanEnd {
		testsResult.TestSuites = append(testsResult.TestSuites, t)
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
