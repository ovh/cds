package check

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/smartystreets/assertions"

	"github.com/ovh/cds/sdk"
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

// Assertion checks assertion
func Assertion(assert []string, actual interface{}, tc *sdk.TestCase, ts *sdk.TestStep, l *log.Entry) {
	f, ok := assertMap[assert[1]]
	if !ok {
		tc.Errors = append(tc.Errors, sdk.Failure{Value: fmt.Sprintf("Method not found \"%s\"", assert[1])})
		return
	}
	args := make([]interface{}, len(assert[2:]))
	for i, v := range assert[2:] { // convert []string to []interface for assertions.func()...
		args[i] = v
	}
	out := f(actual, args...)
	if out != "" {
		c := fmt.Sprintf("TestCase:%s\n %s", tc.Name, ts.ScriptContent)
		if len(c) > 200 {
			c = c[0:200] + "..."
		}
		if ts.Result.StdOut != "" {
			out += "\n" + ts.Result.StdOut
		}
		if ts.Result.StdErr != "" {
			out += "\n" + ts.Result.StdErr
		}
		tc.Failures = append(tc.Failures, sdk.Failure{Value: fmt.Sprintf("%s\n%s", c, out)})
	}
}

// assertMap contains list of assertions func
var assertMap = map[string]func(actual interface{}, expected ...interface{}) string{
	"ShouldEqual":          assertions.ShouldEqual,
	"ShouldNotEqual":       assertions.ShouldNotEqual,
	"ShouldAlmostEqual":    assertions.ShouldAlmostEqual,
	"ShouldNotAlmostEqual": assertions.ShouldNotAlmostEqual,
	"ShouldResemble":       assertions.ShouldResemble,
	"ShouldNotResemble":    assertions.ShouldNotResemble,
	"ShouldPointTo":        assertions.ShouldPointTo,
	"ShouldNotPointTo":     assertions.ShouldNotPointTo,
	"ShouldBeNil":          assertions.ShouldBeNil,
	"ShouldNotBeNil":       assertions.ShouldNotBeNil,
	"ShouldBeTrue":         assertions.ShouldBeTrue,
	"ShouldBeFalse":        assertions.ShouldBeFalse,
	"ShouldBeZeroValue":    assertions.ShouldBeZeroValue,

	"ShouldBeGreaterThan":          assertions.ShouldBeGreaterThan,
	"ShouldBeGreaterThanOrEqualTo": assertions.ShouldBeGreaterThanOrEqualTo,
	"ShouldBeLessThan":             assertions.ShouldBeLessThan,
	"ShouldBeLessThanOrEqualTo":    assertions.ShouldBeLessThanOrEqualTo,
	"ShouldBeBetween":              assertions.ShouldBeBetween,
	"ShouldNotBeBetween":           assertions.ShouldNotBeBetween,
	"ShouldBeBetweenOrEqual":       assertions.ShouldBeBetweenOrEqual,
	"ShouldNotBeBetweenOrEqual":    assertions.ShouldNotBeBetweenOrEqual,

	"ShouldContain":       assertions.ShouldContain,
	"ShouldNotContain":    assertions.ShouldNotContain,
	"ShouldContainKey":    assertions.ShouldContainKey,
	"ShouldNotContainKey": assertions.ShouldNotContainKey,
	"ShouldBeIn":          assertions.ShouldBeIn,
	"ShouldNotBeIn":       assertions.ShouldNotBeIn,
	"ShouldBeEmpty":       assertions.ShouldBeEmpty,
	"ShouldNotBeEmpty":    assertions.ShouldNotBeEmpty,
	"ShouldHaveLength":    assertions.ShouldHaveLength,

	"ShouldStartWith":           assertions.ShouldStartWith,
	"ShouldNotStartWith":        assertions.ShouldNotStartWith,
	"ShouldEndWith":             assertions.ShouldEndWith,
	"ShouldNotEndWith":          assertions.ShouldNotEndWith,
	"ShouldBeBlank":             assertions.ShouldBeBlank,
	"ShouldNotBeBlank":          assertions.ShouldNotBeBlank,
	"ShouldContainSubstring":    assertions.ShouldContainSubstring,
	"ShouldNotContainSubstring": assertions.ShouldNotContainSubstring,

	"ShouldEqualWithout":   assertions.ShouldEqualWithout,
	"ShouldEqualTrimSpace": assertions.ShouldEqualTrimSpace,

	"ShouldHappenBefore":         assertions.ShouldHappenBefore,
	"ShouldHappenOnOrBefore":     assertions.ShouldHappenOnOrBefore,
	"ShouldHappenAfter":          assertions.ShouldHappenAfter,
	"ShouldHappenOnOrAfter":      assertions.ShouldHappenOnOrAfter,
	"ShouldHappenBetween":        assertions.ShouldHappenBetween,
	"ShouldHappenOnOrBetween":    assertions.ShouldHappenOnOrBetween,
	"ShouldNotHappenOnOrBetween": assertions.ShouldNotHappenOnOrBetween,
	"ShouldHappenWithin":         assertions.ShouldHappenWithin,
	"ShouldNotHappenWithin":      assertions.ShouldNotHappenWithin,
	"ShouldBeChronological":      assertions.ShouldBeChronological,
}
