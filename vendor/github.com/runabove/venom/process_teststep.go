package venom

import (
	"context"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
)

func runTestStep(tcc TestCaseContext, e *executorWrap, ts *TestSuite, tc *TestCase, step TestStep, templater *Templater, l *log.Entry, detailsLevel string) {

	var isOK bool
	var errors []Failure
	var failures []Failure
	var systemerr, systemout string

	var retry int

	for retry = 0; retry <= e.retry && !isOK; retry++ {
		if retry > 1 && !isOK {
			log.Debugf("Sleep %d, it's %d attempt", e.delay, retry)
			time.Sleep(time.Duration(e.delay) * time.Second)
		}

		result, err := runTestStepExecutor(tcc, e, ts, step, templater, l)

		if err != nil {
			tc.Failures = append(tc.Failures, Failure{Value: err.Error()})
			continue
		}

		// add result in templater
		ts.Templater.Add(tc.Name, result)

		if h, ok := e.executor.(executorWithDefaultAssertions); ok {
			isOK, errors, failures, systemout, systemerr = applyChecks(&result, step, h.GetDefaultAssertions(), l)
		} else {
			isOK, errors, failures, systemout, systemerr = applyChecks(&result, step, nil, l)
		}
		// add result again for extracts values
		ts.Templater.Add(tc.Name, result)

		log.Debugf("result step:%+v", result)

		if isOK {
			break
		}
	}
	tc.Errors = append(tc.Errors, errors...)
	tc.Failures = append(tc.Failures, failures...)
	if retry > 1 && (len(failures) > 0 || len(errors) > 0) {
		tc.Failures = append(tc.Failures, Failure{Value: fmt.Sprintf("It's a failure after %d attempts", retry)})
	}
	tc.Systemout.Value += systemout
	tc.Systemerr.Value += systemerr
}

func runTestStepExecutor(tcc TestCaseContext, e *executorWrap, ts *TestSuite, step TestStep, templater *Templater, l *log.Entry) (ExecutorResult, error) {
	if e.timeout == 0 {
		return e.executor.Run(tcc, l, step)
	}

	ctxTimeout, cancel := context.WithTimeout(context.Background(), time.Duration(e.timeout)*time.Second)
	defer cancel()

	ch := make(chan ExecutorResult)
	cherr := make(chan error)
	go func(tcc TestCaseContext, e *executorWrap, step TestStep, l *log.Entry) {
		result, err := e.executor.Run(tcc, l, step)
		if err != nil {
			cherr <- err
		} else {
			ch <- result
		}
	}(tcc, e, step, l)

	select {
	case err := <-cherr:
		return nil, err
	case result := <-ch:
		return result, nil
	case <-ctxTimeout.Done():
		return nil, fmt.Errorf("Timeout after %d second(s)", e.timeout)
	}
}
