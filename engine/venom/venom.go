package venom

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
)

type (
	// Aliases contains list of aliases
	Aliases map[string]string

	// ExecutorResult represents an executor result on a test step
	ExecutorResult map[string]string
)

// StepAssertions contains step assertions
type StepAssertions struct {
	Assertions []string `json:"assertions,omitempty" yaml:"assertions,omitempty"`
}

// Executor execute a testStep.
type Executor interface {
	// Run run a Test Step
	Run(*log.Entry, Aliases, TestStep) (ExecutorResult, error)
	// GetDefaultAssertion returns default assertions
	GetDefaultAssertions() StepAssertions
}

var (
	executors = map[string]Executor{}
)

// RegisterExecutor register Test Executors
func RegisterExecutor(name string, e Executor) {
	executors[name] = e
}

// getExecutor initializes a test by name
func getExecutor(t map[string]interface{}) (Executor, error) {

	var name string
	itype, ok := t["type"]
	if ok {
		name = fmt.Sprintf("%s", itype)
	}
	if name == "" {
		name = "exec"
	}

	e, ok := executors[name]
	if !ok {
		return nil, fmt.Errorf("type '%s' is not implemented", name)
	}
	return e, nil
}
