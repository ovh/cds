package run

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/smartystreets/assertions"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/sdk"
)

var aliases map[string]string

func process() (sdk.Tests, error) {
	log.Infof("Start processing path %s", path)

	aliases = make(map[string]string)

	for _, a := range alias {
		t := strings.Split(a, ":")
		if len(t) < 2 {
			continue
		}
		aliases[t[0]] = strings.Join(t[1:], "")
	}

	fileInfo, _ := os.Stat(path)
	if fileInfo != nil && fileInfo.IsDir() {
		path = filepath.Dir(path) + "/*.yml"
		log.Debugf("path computed:%s", path)
	}

	filesPath, err := filepath.Glob(path)
	if err != nil {
		log.Fatalf("Error reading files on path:%s :%s", path, err)
	}

	tss := []sdk.TestSuite{}

	log.Debugf("Work with parallel %d", parallel)
	parallels := make(chan string, parallel)

	var wg sync.WaitGroup
	wg.Add(len(filesPath))
	queue := make(chan sdk.TestSuite, 1)

	go func() {
		for file := range parallels {
			go func(f string) {

				l := log.WithField("file", f)
				l.Debug("readFile")

				dat, err := ioutil.ReadFile(f)
				if err != nil {
					l.WithError(err).Errorf("Error while reading file")
					wg.Done()
					return
				}

				ts := sdk.TestSuite{}
				if err := yaml.Unmarshal(dat, &ts); err != nil {
					l.WithError(err).Errorf("Error while unmarshal file")
					wg.Done()
					return
				}

				runTestSuite(&ts, l)
				queue <- ts
			}(file)
		}
	}()

	tr := sdk.Tests{}
	go func() {
		for t := range queue {
			tss = append(tss, t)
			if t.Failures > 0 {
				tr.TotalKO += t.Failures
			} else {
				tr.TotalOK += len(t.TestCases) - t.Failures
			}
			if t.Skipped > 0 {
				tr.TotalSkipped += t.Skipped
			}

			tr.Total = tr.TotalKO + tr.TotalOK + tr.TotalSkipped

			wg.Done()
		}
	}()

	for _, file := range filesPath {
		parallels <- file
	}

	wg.Wait()

	log.Infof("end processing path %s", path)

	tr.TestSuites = tss
	return tr, nil
}

func runTestSuite(ts *sdk.TestSuite, l *log.Entry) {
	l = l.WithField("v.testsuite", ts.Name)
	for i, tc := range ts.TestCases {
		if tc.Skipped == 0 {
			runTestCase(&tc, l)
			ts.TestCases[i] = tc
		}

		if len(tc.Failures) > 0 {
			ts.Failures += len(tc.Failures)
		}
		if len(tc.Errors) > 0 {
			ts.Errors += len(tc.Errors)
		}
		if tc.Skipped > 0 {
			ts.Skipped += tc.Skipped
		}
	}
}

func runTestCase(tc *sdk.TestCase, l *log.Entry) {
	l = l.WithField("x.testcase", tc.Name)
	l.Infof("start")
	for _, ts := range tc.TestSteps {
		runTestStep(&ts, l)
		applyResult(tc, &ts, l)
		if len(tc.Failures) > 0 {
			break
		}
	}
	l.Infof("end")
}

func runTestStep(s *sdk.TestStep, l *log.Entry) {
	stype, errdetect := s.DetectType()
	if errdetect != nil {
		s.Result.Err = errdetect
		return
	}
	s.Type = stype
	l = l.WithField("y.teststep", stype)
	l.Debug("start")
	defer l.Debug("end")

	switch stype {
	case "exec":
		l = l.WithField("z.type", "exec")
		runExec(s, l)
		return
	}
	s.Result.Err = fmt.Errorf("Type %s not supported", stype)
	return
}

func runExec(s *sdk.TestStep, l *log.Entry) {
	if s.ScriptContent == "" {
		s.Result.Err = fmt.Errorf("Invalid command")
		return
	}

	scriptContent := s.ScriptContent
	for alias, real := range aliases {
		if strings.HasPrefix(scriptContent, alias) {
			scriptContent = strings.Replace(scriptContent, alias, real, 1)
		}
	}

	// Default shell is sh
	shell := "/bin/sh"
	var opts []string

	// If user wants a specific shell, use it
	if strings.HasPrefix(scriptContent, "#!") {
		t := strings.SplitN(scriptContent, "\n", 2)
		shell = strings.TrimPrefix(t[0], "#!")
		shell = strings.TrimRight(shell, " \t\r\n")
	}

	// except on windows where it's powershell
	if runtime.GOOS == "windows" {
		shell = "PowerShell"
		opts = append(opts, "-ExecutionPolicy", "Bypass", "-Command")
	}

	// Create a tmp file
	tmpscript, errt := ioutil.TempFile(os.TempDir(), "venom-")
	if errt != nil {
		s.Result.Err = fmt.Errorf("Cannot create tmp file: %s\n", errt)
		return
	}

	// Put script in file
	l.Debugf("work with tmp file %s", tmpscript)
	n, errw := tmpscript.Write([]byte(scriptContent))
	if errw != nil || n != len(scriptContent) {
		if errw != nil {
			s.Result.Err = fmt.Errorf("Cannot write script: %s\n", errw)
			return
		}
		s.Result.Err = fmt.Errorf("cannot write all script: %d/%d\n", n, len(scriptContent))
		return
	}

	oldPath := tmpscript.Name()
	tmpscript.Close()
	var scriptPath string
	if runtime.GOOS == "windows" {
		//Remove all .txt Extensions, there is not always a .txt extension
		newPath := strings.Replace(oldPath, ".txt", "", -1)
		//and add .PS1 extension
		newPath = newPath + ".PS1"
		if err := os.Rename(oldPath, newPath); err != nil {
			s.Result.Err = fmt.Errorf("cannot rename script to add powershell Extension, aborting\n")
			return
		}
		//This aims to stop a the very first error and return the right exit code
		psCommand := fmt.Sprintf("& { $ErrorActionPreference='Stop'; & %s ;exit $LastExitCode}", newPath)
		scriptPath = newPath
		opts = append(opts, psCommand)
	} else {
		scriptPath = oldPath
		opts = append(opts, scriptPath)
	}
	defer os.Remove(scriptPath)

	// Chmod file
	if errc := os.Chmod(scriptPath, 0755); errc != nil {
		s.Result.Err = fmt.Errorf("cannot chmod script %s: %s\n", scriptPath, errc)
		return
	}

	cmd := exec.Command(shell, opts...)
	l.Debugf("teststep exec '%s %s'", shell, strings.Join(opts, " "))

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		l.Warning("runScriptAction: Cannot get stdout pipe: %s\n", err)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		l.Warning("runScriptAction: Cannot get stderr pipe: %s\n", err)
		return
	}

	stdoutreader := bufio.NewReader(stdout)
	stderrreader := bufio.NewReader(stderr)

	outchan := make(chan bool)
	go func() {
		for {
			line, errs := stdoutreader.ReadString('\n')
			if errs != nil {
				stdout.Close()
				close(outchan)
				return
			}
			s.Result.StdOut += line
			l.Debugf(line)
		}
	}()

	errchan := make(chan bool)
	go func() {
		for {
			line, errs := stderrreader.ReadString('\n')
			if errs != nil {
				stderr.Close()
				close(errchan)
				return
			}
			s.Result.StdErr += line
			l.Debugf(line)
		}
	}()

	if err := cmd.Start(); err != nil {
		s.Result.Err = err
		s.Result.Code = "127"
		l.Debugf(err.Error())
		return
	}

	_ = <-outchan
	_ = <-errchan

	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				s.Result.Code = strconv.Itoa(status.ExitStatus())
			}
		}

		s.Result.Err = err
		return
	}
	s.Result.Code = "0"
}

func runHTTP(s *sdk.TestStep, l *log.Entry) error {
	return fmt.Errorf("type http not yet implemented")
}

func applyResult(tc *sdk.TestCase, ts *sdk.TestStep, l *log.Entry) error {
	tc.Systemerr.Value = ts.Result.StdErr
	tc.Systemout.Value = ts.Result.StdOut

	if ts.Result.Err != nil {
		tc.Systemerr.Value += ts.Result.Err.Error()
	}

	if len(ts.Assertions) == 0 {
		ts.Assertions = []string{""}
	}

	for _, a := range ts.Assertions {
		checkAssertion(tc, ts, a, l)
	}

	return nil
}

func checkAssertion(tc *sdk.TestCase, ts *sdk.TestStep, assertion string, l *log.Entry) {
	a, erra := getAssertion(ts, assertion, l)
	if erra != nil {
		tc.Errors = append(tc.Errors, sdk.Failure{Value: erra.Error()})
		return
	}

	assert := strings.Split(a, " ")
	if len(assert) < 3 {
		tc.Errors = append(tc.Errors, sdk.Failure{Value: fmt.Sprintf("invalid assertion '%s' type:'%s' len:'%d'", a, ts.Type, len(assert))})
		return
	}

	switch assert[0] {
	case "code":
		checkCode(assert, tc, ts, l)
		return
	}
	tc.Errors = append(tc.Errors, sdk.Failure{Value: fmt.Sprintf("invalid assertion %s", assertion)})
}

func getAssertion(ts *sdk.TestStep, assertion string, l *log.Entry) (string, error) {
	if assertion != "" {
		return assertion, nil
	}

	switch ts.Type {
	case "exec":
		return "code ShouldEqual 0", nil
	case "http":
		return "code ShouldBeLessThan 300", nil
	}

	return "", fmt.Errorf("Invalid assertion of type %s", ts.Type)
}

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

func checkCode(assert []string, tc *sdk.TestCase, ts *sdk.TestStep, l *log.Entry) {
	f, ok := assertMap[assert[1]]
	if !ok {
		tc.Errors = append(tc.Errors, sdk.Failure{Value: fmt.Sprintf("Method not found \"%s\"", assert[1])})
		return
	}
	args := make([]interface{}, len(assert[2:]))
	for i, v := range assert[2:] { // convert []string to []interface for assertions.func()...
		args[i] = v
	}
	out := f(ts.Result.Code, args...)
	if out != "" {
		c := ts.ScriptContent
		if len(c) > 200 {
			c = c[0:200] + "..."
		}
		tc.Failures = append(tc.Failures, sdk.Failure{Value: fmt.Sprintf("%s... give %s", c, out)})
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
