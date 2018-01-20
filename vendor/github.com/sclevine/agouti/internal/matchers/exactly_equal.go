package matchers

import (
	"fmt"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
)

type exactlyEqualMatcher struct {
	expected interface{}
}

func (m *exactlyEqualMatcher) Match(actual interface{}) (success bool, err error) {
	if actual == nil && m.expected == nil {
		return false, fmt.Errorf("Refusing to compare <nil> to <nil>.")
	}
	return actual == m.expected, nil
}

func (m *exactlyEqualMatcher) FailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "to exactly equal", m.expected)
}

func (m *exactlyEqualMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "not to exactly equal", m.expected)
}

func ExactlyEqual(expected interface{}) types.GomegaMatcher {
	return &exactlyEqualMatcher{expected: expected}
}
