package venom

import (
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/fsamin/go-dump"
	"github.com/mitchellh/mapstructure"
	"github.com/smartystreets/assertions"
)

type testingT struct {
	ErrorS []string
}

func (t *testingT) Error(args ...interface{}) {
	for _, a := range args {
		switch v := a.(type) {
		case string:
			t.ErrorS = append(t.ErrorS, v)
		default:
			t.ErrorS = append(t.ErrorS, fmt.Sprintf("%s", v))
		}
	}
}

// applyChecks apply checks on result, return true if all assertions are OK, false otherwise
func applyChecks(executorResult *ExecutorResult, step TestStep, defaultAssertions *StepAssertions, l *log.Entry) (bool, []Failure, []Failure, string, string) {
	isOK, errors, failures, systemout, systemerr := applyAssertions(*executorResult, step, defaultAssertions, l)
	if !isOK {
		return isOK, errors, failures, systemout, systemerr
	}

	isOKExtract, errorsExtract, failuresExtract := applyExtracts(executorResult, step, l)

	errors = append(errors, errorsExtract...)
	failures = append(failures, failuresExtract...)

	return isOKExtract, errors, failures, systemout, systemerr
}

func applyAssertions(executorResult ExecutorResult, step TestStep, defaultAssertions *StepAssertions, l *log.Entry) (bool, []Failure, []Failure, string, string) {
	var sa StepAssertions
	var errors []Failure
	var failures []Failure
	var systemerr, systemout string

	if err := mapstructure.Decode(step, &sa); err != nil {
		return false, []Failure{{Value: fmt.Sprintf("error decoding assertions: %s", err)}}, failures, systemout, systemerr
	}

	if len(sa.Assertions) == 0 && defaultAssertions != nil {
		sa = *defaultAssertions
	}

	isOK := true
	for _, assertion := range sa.Assertions {
		errs, fails := check(assertion, executorResult, l)
		if errs != nil {
			errors = append(errors, *errs)
			isOK = false
		}
		if fails != nil {
			failures = append(failures, *fails)
			isOK = false
		}
	}

	if _, ok := executorResult["result.systemerr"]; ok {
		systemerr = executorResult["result.systemerr"]
	}

	if _, ok := executorResult["result.systemout"]; ok {
		systemout = executorResult["result.systemout"]
	}

	return isOK, errors, failures, systemout, systemerr
}

func check(assertion string, executorResult ExecutorResult, l *log.Entry) (*Failure, *Failure) {
	assert := strings.Split(assertion, " ")
	if len(assert) < 2 {
		return &Failure{Value: fmt.Sprintf("invalid assertion '%s' len:'%d'", assertion, len(assert))}, nil
	}

	actual, ok := executorResult[assert[0]]
	if !ok {
		if assert[1] == "ShouldNotExist" {
			return nil, nil
		}
		return &Failure{Value: fmt.Sprintf("key '%s' does not exist in result of executor: %+v", assert[0], executorResult)}, nil
	} else if assert[1] == "ShouldNotExist" {
		return &Failure{Value: fmt.Sprintf("key '%s' should not exist in result of executor. Value: %+v", assert[0], actual)}, nil
	}

	f, ok := assertMap[assert[1]]
	if !ok {
		return &Failure{Value: fmt.Sprintf("Method not found '%s'", assert[1])}, nil
	}
	args := make([]interface{}, len(assert[2:]))
	for i, v := range assert[2:] { // convert []string to []interface for assertions.func()...
		args[i] = v
	}

	out := f(actual, args...)

	if out != "" {
		prefix := "assertion: " + assertion
		sdump, _ := dump.Sdump(executorResult)
		return nil, &Failure{Value: prefix + "\n" + out + "\n" + sdump}
	}
	return nil, nil
}

// assertMap contains list of assertions func
var assertMap = map[string]func(actual interface{}, expected ...interface{}) string{
	// "ShouldNotExist" see func check
	"ShouldEqual":                  assertions.ShouldEqual,
	"ShouldNotEqual":               assertions.ShouldNotEqual,
	"ShouldAlmostEqual":            assertions.ShouldAlmostEqual,
	"ShouldNotAlmostEqual":         assertions.ShouldNotAlmostEqual,
	"ShouldResemble":               assertions.ShouldResemble,
	"ShouldNotResemble":            assertions.ShouldNotResemble,
	"ShouldPointTo":                assertions.ShouldPointTo,
	"ShouldNotPointTo":             assertions.ShouldNotPointTo,
	"ShouldBeTrue":                 assertions.ShouldBeTrue,
	"ShouldBeFalse":                assertions.ShouldBeFalse,
	"ShouldBeZeroValue":            assertions.ShouldBeZeroValue,
	"ShouldBeGreaterThan":          assertions.ShouldBeGreaterThan,
	"ShouldBeGreaterThanOrEqualTo": assertions.ShouldBeGreaterThanOrEqualTo,
	"ShouldBeLessThan":             assertions.ShouldBeLessThan,
	"ShouldBeLessThanOrEqualTo":    assertions.ShouldBeLessThanOrEqualTo,
	"ShouldBeBetween":              assertions.ShouldBeBetween,
	"ShouldNotBeBetween":           assertions.ShouldNotBeBetween,
	"ShouldBeBetweenOrEqual":       assertions.ShouldBeBetweenOrEqual,
	"ShouldNotBeBetweenOrEqual":    assertions.ShouldNotBeBetweenOrEqual,
	"ShouldContain":                assertions.ShouldContain,
	"ShouldNotContain":             assertions.ShouldNotContain,
	"ShouldContainKey":             assertions.ShouldContainKey,
	"ShouldNotContainKey":          assertions.ShouldNotContainKey,
	"ShouldBeIn":                   assertions.ShouldBeIn,
	"ShouldNotBeIn":                assertions.ShouldNotBeIn,
	"ShouldBeEmpty":                assertions.ShouldBeEmpty,
	"ShouldNotBeEmpty":             assertions.ShouldNotBeEmpty,
	"ShouldHaveLength":             assertions.ShouldHaveLength,
	"ShouldStartWith":              assertions.ShouldStartWith,
	"ShouldNotStartWith":           assertions.ShouldNotStartWith,
	"ShouldEndWith":                assertions.ShouldEndWith,
	"ShouldNotEndWith":             assertions.ShouldNotEndWith,
	"ShouldBeBlank":                assertions.ShouldBeBlank,
	"ShouldNotBeBlank":             assertions.ShouldNotBeBlank,
	"ShouldContainSubstring":       ShouldContainSubstring,
	"ShouldNotContainSubstring":    assertions.ShouldNotContainSubstring,
	"ShouldEqualWithout":           assertions.ShouldEqualWithout,
	"ShouldEqualTrimSpace":         assertions.ShouldEqualTrimSpace,
	"ShouldHappenBefore":           assertions.ShouldHappenBefore,
	"ShouldHappenOnOrBefore":       assertions.ShouldHappenOnOrBefore,
	"ShouldHappenAfter":            assertions.ShouldHappenAfter,
	"ShouldHappenOnOrAfter":        assertions.ShouldHappenOnOrAfter,
	"ShouldHappenBetween":          assertions.ShouldHappenBetween,
	"ShouldHappenOnOrBetween":      assertions.ShouldHappenOnOrBetween,
	"ShouldNotHappenOnOrBetween":   assertions.ShouldNotHappenOnOrBetween,
	"ShouldHappenWithin":           assertions.ShouldHappenWithin,
	"ShouldNotHappenWithin":        assertions.ShouldNotHappenWithin,
	"ShouldBeChronological":        assertions.ShouldBeChronological,
}

// ShouldContainSubstring receives exactly more than 2 string parameters and ensures that the first contains the second as a substring.
func ShouldContainSubstring(actual interface{}, expected ...interface{}) string {
	if len(expected) == 1 {
		return assertions.ShouldContainSubstring(actual, expected...)
	}

	var arg string
	for _, e := range expected {
		arg += fmt.Sprintf("%v ", e)
	}
	return assertions.ShouldContainSubstring(actual, strings.TrimSpace(arg))
}
