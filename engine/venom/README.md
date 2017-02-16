# Venom - RUN Integration Tests

## CLI

Install with:
```bash
$ go install github.com/ovh/cds/engine/venom/cli/venom
```

```bash
$ venom run -h
Run Tests

Usage:
  venom run [flags]

Flags:
      --alias stringSlice   --alias cds:'cds -f config.json' --alias cds2:'cds -f config.json'
      --details string      Output Details Level : low, medium, high (default "medium")
      --format string       --formt:yaml, json, xml (default "xml")
      --log string          Log Level : debug, info or warn (default "warn")
      --output-dir string   Output Directory: create tests results file inside this directory
      --parallel int        --parallel=2 (default 1)
      --resume              Output Resume: one line with Total, TotalOK, TotalKO, TotalSkipped, TotalTestSuite (default true)
      --resumeFailures      Output Resume Failures (default true)
```

## TestSuite files

* Run `venom template`
* Examples: https://github.com/ovh/cds/tree/master/tests

Example:

```yaml

name: Title of TestSuite
testcases:
- name: TestCase with default value, exec cmd. Check if exit code != 1
  steps:
  - script: echo 'foo'
    type: exec
- name: Title of First TestCase
  steps:
  - script: echo 'foo'
    assertions:
    - Result.Code ShouldEqual 0
  - script: echo 'bar'
    assertions:
    - Result.StdOut ShouldNotContainSubstring bar

```

## RUN Venom locally on CDS Integration Tests

```bash
cd $GOPATH/src/github.com/ovh/cds/tests
venom run --alias='cdsro:cds -f $HOME/.cds/it.user.ro.json' --alias='cds:cds -f $HOME/.cds/it.user.rw.json' --parallel=5
```

## RUN Venom, with an export xUnit

```bash
venom run  --details=low --format=xml --output-dir="."
```

## Assertion

### Keywords
* ShouldEqual
* ShouldNotEqual
* ShouldAlmostEqual
* ShouldNotAlmostEqual
* ShouldResemble
* ShouldNotResemble
* ShouldPointTo
* ShouldNotPointTo
* ShouldBeNil
* ShouldNotBeNil
* ShouldBeTrue
* ShouldBeFalse
* ShouldBeZeroValue
* ShouldBeGreaterThan
* ShouldBeGreaterThanOrEqualTo
* ShouldBeLessThan
* ShouldBeLessThanOrEqualTo
* ShouldBeBetween
* ShouldNotBeBetween
* ShouldBeBetweenOrEqual
* ShouldNotBeBetweenOrEqual
* ShouldContain
* ShouldNotContain
* ShouldContainKey
* ShouldNotContainKey
* ShouldBeIn
* ShouldNotBeIn
* ShouldBeEmpty
* ShouldNotBeEmpty
* ShouldHaveLength
* ShouldStartWith
* ShouldNotStartWith
* ShouldEndWith
* ShouldNotEndWith
* ShouldBeBlank
* ShouldNotBeBlank
* ShouldContainSubstring
* ShouldNotContainSubstring
* ShouldEqualWithout
* ShouldEqualTrimSpace
* ShouldHappenBefore
* ShouldHappenOnOrBefore
* ShouldHappenAfter
* ShouldHappenOnOrAfter
* ShouldHappenBetween
* ShouldHappenOnOrBetween
* ShouldNotHappenOnOrBetween
* ShouldHappenWithin
* ShouldNotHappenWithin
* ShouldBeChronological


## Write your executor

An executor have to implement this interface

```go

// Executor execute a testStep.
type Executor interface {
	// Run run a Test Step
	Run(*log.Entry, Aliases, TestStep) (ExecutorResult, error)
	// GetDefaultAssertion returns default assertions
	GetDefaultAssertions() StepAssertions
}
```

Example

```go

package myexecutor

import (
	log "github.com/Sirupsen/logrus"
	"github.com/fsamin/go-dump"
	"github.com/mitchellh/mapstructure"

	"github.com/ovh/cds/engine/venom"
)


// Name of executor
const Name = "myexecutor"

// New returns a new Executor
func New() venom.Executor {
	return &Executor{}
}

// Executor struct
type Executor struct {
	Command string `json:"command,omitempty" yaml:"command,omitempty"`
}

// Result represents a step result
type Result struct {
	Command string `json:"command,omitempty" yaml:"command,omitempty"`
	Output  string `json:"Output,omitempty" yaml:"Output,omitempty"`
}

// GetDefaultAssertions return default assertions for this executor
func (Executor) GetDefaultAssertions() venom.StepAssertions {
	return venom.StepAssertions{Assertions: []string{"Result.Command ShouldEqual 0"}}
}

// Run execute TestStep
func (Executor) Run(l *log.Entry, aliases venom.Aliases, step venom.TestStep) (venom.ExecutorResult, error) {

  // transform step to Executor Instance
	var t Executor
	if err := mapstructure.Decode(step, &t); err != nil {
		return nil, err
	}

  // to something with t.Command here...
  //...
  output := "foo"

  // prepare result
  r := Result{
    Command: t.Command, // return Command runn
    Output: output, // return Command runn

  }


	return dump.ToMap(result)
}
```

Feel free to open a Pull Request with your executors.
