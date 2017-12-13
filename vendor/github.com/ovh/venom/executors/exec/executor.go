package exec

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
	"time"

	"github.com/fsamin/go-dump"
	"github.com/mitchellh/mapstructure"

	"github.com/ovh/venom"
	"github.com/ovh/venom/executors"
)

// Name for test exec
const Name = "exec"

// New returns a new Test Exec
func New() venom.Executor {
	return &Executor{}
}

// Executor represents a Test Exec
type Executor struct {
	Script string `json:"script,omitempty" yaml:"script,omitempty"`
}

// Result represents a step result
type Result struct {
	Executor    Executor `json:"executor,omitempty" yaml:"executor,omitempty"`
	Systemout   string   `json:"systemout,omitempty" yaml:"systemout,omitempty"`
	Systemerr   string   `json:"systemerr,omitempty" yaml:"systemerr,omitempty"`
	Err         string   `json:"err,omitempty" yaml:"err,omitempty"`
	Code        string   `json:"code,omitempty" yaml:"code,omitempty"`
	TimeSeconds float64  `json:"timeseconds,omitempty" yaml:"timeseconds,omitempty"`
	TimeHuman   string   `json:"timehuman,omitempty" yaml:"timehuman,omitempty"`
}

// ZeroValueResult return an empty implemtation of this executor result
func (Executor) ZeroValueResult() venom.ExecutorResult {
	r, _ := executors.Dump(Result{})
	return r
}

// GetDefaultAssertions return default assertions for type exec
func (Executor) GetDefaultAssertions() *venom.StepAssertions {
	return &venom.StepAssertions{Assertions: []string{"result.code ShouldEqual 0"}}
}

// Run execute TestStep of type exec
func (Executor) Run(testCaseContext venom.TestCaseContext, l venom.Logger, step venom.TestStep) (venom.ExecutorResult, error) {

	var e Executor
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}

	if e.Script == "" {
		return nil, fmt.Errorf("Invalid command")
	}

	scriptContent := e.Script

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
		return nil, fmt.Errorf("Cannot create tmp file: %s\n", errt)
	}

	// Put script in file
	l.Debugf("work with tmp file %s", tmpscript)
	n, errw := tmpscript.Write([]byte(scriptContent))
	if errw != nil || n != len(scriptContent) {
		if errw != nil {
			return nil, fmt.Errorf("Cannot write script: %s\n", errw)
		}
		return nil, fmt.Errorf("cannot write all script: %d/%d\n", n, len(scriptContent))
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
			return nil, fmt.Errorf("cannot rename script to add powershell Extension, aborting\n")
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
		return nil, fmt.Errorf("cannot chmod script %s: %s\n", scriptPath, errc)
	}

	start := time.Now()

	cmd := exec.Command(shell, opts...)
	l.Debugf("teststep exec '%s %s'", shell, strings.Join(opts, " "))

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("runScriptAction: Cannot get stdout pipe: %s\n", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("runScriptAction: Cannot get stderr pipe: %s\n", err)
	}

	stdoutreader := bufio.NewReader(stdout)
	stderrreader := bufio.NewReader(stderr)

	result := Result{Executor: e}
	outchan := make(chan bool)
	go func() {
		for {
			line, errs := stdoutreader.ReadString('\n')
			if errs != nil {
				stdout.Close()
				close(outchan)
				return
			}
			result.Systemout += line
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
			result.Systemerr += line
			l.Debugf(line)
		}
	}()

	if err := cmd.Start(); err != nil {
		result.Err = err.Error()
		result.Code = "127"
		l.Debugf(err.Error())
		return dump.ToMap(e, nil, dump.WithDefaultLowerCaseFormatter())
	}

	_ = <-outchan
	_ = <-errchan

	result.Code = "0"
	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				result.Code = strconv.Itoa(status.ExitStatus())
			}
		}
	}

	elapsed := time.Since(start)
	result.TimeSeconds = elapsed.Seconds()
	result.TimeHuman = fmt.Sprintf("%s", elapsed)

	result.Systemout = venom.RemoveNotPrintableChar(strings.TrimRight(result.Systemout, "\n"))
	result.Systemerr = venom.RemoveNotPrintableChar(strings.TrimRight(result.Systemerr, "\n"))

	return executors.Dump(result)
}
