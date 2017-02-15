package script

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	log "github.com/Sirupsen/logrus"

	"github.com/ovh/cds/engine/venom"
	"github.com/ovh/cds/engine/venom/check"
	"github.com/ovh/cds/sdk"
)

func init() {
	venom.RegisterTestFactory("script", func() venom.Test { return &TestExec{} })
}

// TestExec represents a Test Exec
type TestExec struct{}

// Check check a script result
func (*TestExec) Check(tc *sdk.TestCase, ts *sdk.TestStep, assertion string, l *log.Entry) {
	assert := strings.Split(assertion, " ")
	if len(assert) < 3 {
		tc.Errors = append(tc.Errors, sdk.Failure{Value: fmt.Sprintf("invalid assertion '%s' len:'%d'", assertion, len(assert))})
		return
	}

	switch assert[0] {
	case "code":
		check.Assertion(assert, ts.Result.Code, tc, ts, l)
		return
	case "stderr":
		check.Assertion(assert, ts.Result.StdErr, tc, ts, l)
		return
	case "stdout":
		check.Assertion(assert, ts.Result.StdOut, tc, ts, l)
		return
	}

	tc.Errors = append(tc.Errors, sdk.Failure{Value: fmt.Sprintf("invalid assertion %s", assertion)})
}

// GetDefaultAssertion return default assertion for type exec
func (*TestExec) GetDefaultAssertion(a string) string {
	return "code ShouldEqual 0"
}

// Run execute TestStep of type exec
func (*TestExec) Run(s *sdk.TestStep, l *log.Entry, aliases map[string]string) {
	if s.ScriptContent == "" {
		s.Result.Err = fmt.Errorf("Invalid command")
		return
	}

	scriptContent := s.ScriptContent
	for alias, real := range aliases {
		if strings.Contains(scriptContent, alias+" ") {
			scriptContent = strings.Replace(scriptContent, alias+" ", real+" ", 1)
		}
	}

	s.ScriptContent = scriptContent

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
