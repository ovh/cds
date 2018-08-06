package venom

import (
	"context"
	"fmt"
	"time"
)

//RunTestStep executes a venom testcase is a venom context
func (v *Venom) RunTestStep(tcc TestCaseContext, e *ExecutorWrap, ts *TestSuite, tc *TestCase, stepNumber int, step TestStep, l Logger) ExecutorResult {
	var assertRes assertionsApplied

	var retry int
	var result ExecutorResult

	for retry = 0; retry <= e.retry && !assertRes.ok; retry++ {
		if retry > 1 && !assertRes.ok {
			l.Debugf("Sleep %d, it's %d attempt", e.delay, retry)
			time.Sleep(time.Duration(e.delay) * time.Second)
		}

		var err error
		result, err = runTestStepExecutor(tcc, e, ts, step, l)

		if err != nil {
			tc.Failures = append(tc.Failures, Failure{Value: RemoveNotPrintableChar(err.Error())})
			continue
		}

		// add result in templater
		ts.Templater.Add(tc.Name, stringifyExecutorResult(result))

		if h, ok := e.executor.(executorWithDefaultAssertions); ok {
			assertRes = applyChecks(&result, *tc, stepNumber, step, h.GetDefaultAssertions())
		} else {
			assertRes = applyChecks(&result, *tc, stepNumber, step, nil)
		}
		// add result again for extracts values
		ts.Templater.Add(tc.Name, stringifyExecutorResult(result))

		if assertRes.ok {
			break
		}
	}
	tc.Errors = append(tc.Errors, assertRes.errors...)
	tc.Failures = append(tc.Failures, assertRes.failures...)
	if retry > 1 && (len(assertRes.failures) > 0 || len(assertRes.errors) > 0) {
		tc.Failures = append(tc.Failures, Failure{Value: fmt.Sprintf("It's a failure after %d attempts", retry)})
	}
	tc.Systemout.Value += assertRes.systemout
	tc.Systemerr.Value += assertRes.systemerr

	return result
}

func stringifyExecutorResult(e ExecutorResult) map[string]string {
	out := make(map[string]string)
	for k, v := range e {
		out[k] = fmt.Sprintf("%v", v)
	}
	return out
}

func runTestStepExecutor(tcc TestCaseContext, e *ExecutorWrap, ts *TestSuite, step TestStep, l Logger) (ExecutorResult, error) {
	if e.timeout == 0 {
		return e.executor.Run(tcc, l, step, ts.WorkDir)
	}

	ctxTimeout, cancel := context.WithTimeout(context.Background(), time.Duration(e.timeout)*time.Second)
	defer cancel()

	ch := make(chan ExecutorResult)
	cherr := make(chan error)
	go func(tcc TestCaseContext, e *ExecutorWrap, step TestStep, l Logger) {
		result, err := e.executor.Run(tcc, l, step, ts.WorkDir)
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
