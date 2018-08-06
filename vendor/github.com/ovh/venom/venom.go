package venom

import (
	"fmt"
	"io"
	"os"
)

var (
	//Version is set with -ldflags "-X github.com/ovh/venom/venom.Version=$(VERSION)"
	Version = "snapshot"
)

func New() *Venom {
	v := &Venom{
		LogLevel:        "info",
		LogOutput:       os.Stdout,
		PrintFunc:       fmt.Printf,
		executors:       map[string]Executor{},
		contexts:        map[string]TestCaseContext{},
		variables:       map[string]string{},
		EnableProfiling: false,
		IgnoreVariables: []string{},
		OutputFormat:    "xml",
	}
	return v
}

type Venom struct {
	LogLevel  string
	LogOutput io.Writer

	PrintFunc func(format string, a ...interface{}) (n int, err error)
	executors map[string]Executor
	contexts  map[string]TestCaseContext

	testsuites      []TestSuite
	variables       map[string]string
	IgnoreVariables []string
	Parallel        int

	EnableProfiling bool
	OutputFormat    string
	OutputDir       string
	StopOnFailure   bool
}

func (v *Venom) AddVariables(variables map[string]string) {
	for k, variable := range variables {
		v.variables[k] = variable
	}
}

// RegisterExecutor register Test Executors
func (v *Venom) RegisterExecutor(name string, e Executor) {
	v.executors[name] = e
}

// WrapExecutor initializes a test by name
// no type -> exec is default
func (v *Venom) WrapExecutor(t map[string]interface{}, tcc TestCaseContext) (*ExecutorWrap, error) {
	var name string
	var retry, delay, timeout int

	if itype, ok := t["type"]; ok {
		name = fmt.Sprintf("%s", itype)
	}

	if name == "" && tcc.GetName() != "default" {
		name = tcc.GetName()
	} else if name == "" {
		name = "exec"
	}

	retry, errRetry := getAttrInt(t, "retry")
	if errRetry != nil {
		return nil, errRetry
	}
	delay, errDelay := getAttrInt(t, "delay")
	if errDelay != nil {
		return nil, errDelay
	}
	timeout, errTimeout := getAttrInt(t, "timeout")
	if errTimeout != nil {
		return nil, errTimeout
	}

	if e, ok := v.executors[name]; ok {
		ew := &ExecutorWrap{
			executor: e,
			retry:    retry,
			delay:    delay,
			timeout:  timeout,
		}
		return ew, nil
	}

	return nil, fmt.Errorf("[%s] type '%s' is not implemented", tcc.GetName(), name)
}

// RegisterTestCaseContext new register TestCaseContext
func (v *Venom) RegisterTestCaseContext(name string, tcc TestCaseContext) {
	v.contexts[name] = tcc
}

// ContextWrap initializes a context for a testcase
// no type -> parent context
func (v *Venom) ContextWrap(tc *TestCase) (TestCaseContext, error) {
	if tc.Context == nil {
		return v.contexts["default"], nil
	}
	var typeName string
	if itype, ok := tc.Context["type"]; ok {
		typeName = fmt.Sprintf("%s", itype)
	}

	if typeName == "" {
		return v.contexts["default"], nil
	}
	v.contexts[typeName].SetTestCase(*tc)
	return v.contexts[typeName], nil
}

func getAttrInt(t map[string]interface{}, name string) (int, error) {
	var out int
	if i, ok := t[name]; ok {
		var ok bool
		out, ok = i.(int)
		if !ok {
			return -1, fmt.Errorf("attribute %s '%s' is not an integer", name, i)
		}
	}
	if out < 0 {
		out = 0
	}
	return out, nil
}
