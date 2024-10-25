package grpcplugins

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bugsnag/osext"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

type script struct {
	dir     string
	shell   string
	content []byte
	opts    []string
}

func RunScript(ctx context.Context, actPlug *actionplugin.Common, chanRes chan *actionplugin.ActionResult, workingDir string, content string) error {
	gores := &actionplugin.ActionResult{Status: sdk.StatusSuccess}

	script := prepareScriptContent(content, workingDir)

	fs := afero.NewOsFs()

	deferFunc, err := writeScriptContent(ctx, script, fs)
	if deferFunc != nil {
		defer deferFunc()
	}
	if err != nil {
		gores.Status = sdk.StatusFail
		gores.Details = fmt.Sprintf("%v", err)
		chanRes <- gores
		return err
	}

	cmd := exec.CommandContext(ctx, script.shell, script.opts...)
	pr, pw := io.Pipe()
	cmd.Dir = script.dir
	cmd.Stdout = pw
	cmd.Stderr = pw
	cmd.Env = os.Environ()

	workerpath, err := osext.Executable()
	if err != nil {
		gores.Status = sdk.StatusFail
		gores.Details = fmt.Sprintf("failure due to internal error (Worker Path): %v", err)
		chanRes <- gores
		return err
	}

	for i := range cmd.Env {
		if strings.HasPrefix(cmd.Env[i], "PATH") {
			cmd.Env[i] = fmt.Sprintf("%s:%s", cmd.Env[i], path.Dir(workerpath))
			break
		}
	}

	reader := bufio.NewReader(pr)

	outchan := make(chan bool)
	goRoutines := sdk.NewGoRoutines(ctx)
	goRoutines.Exec(ctx, "runActionScriptPlugin-runScript-outchan", func(ctx context.Context) {
		for {
			line, errs := reader.ReadString('\n')
			if line != "" {
				Log(actPlug, line)
			}
			if errs != nil {
				close(outchan)
				return
			}
		}
	})

	if err := cmd.Start(); err != nil {
		gores.Status = sdk.StatusFail
		gores.Details = fmt.Sprintf("unable to start command: %v", err)
		chanRes <- gores
		return err
	}

	if err := cmd.Wait(); err != nil {
		gores.Status = sdk.StatusFail
		gores.Details = fmt.Sprintf("command failure: %v", err)
		chanRes <- gores
		return err
	}

	_ = pr.Close()
	<-outchan

	chanRes <- gores
	return nil
}

func prepareScriptContent(scriptContent string, workingDir string) *script {
	var script = script{}

	// Set default shell based on os
	if isWindows() {
		script.shell = "PowerShell"
		script.opts = []string{"-ExecutionPolicy", "Bypass", "-Command"}
	} else {
		script.shell = "/bin/sh"
		script.opts = []string{"-e"}
	}

	if strings.HasPrefix(scriptContent, "#!") { // If user wants a specific shell, use it
		t := strings.SplitN(scriptContent, "\n", 2)
		script.shell = strings.TrimPrefix(t[0], "#!")             // Find out the shebang
		script.shell = strings.TrimRight(script.shell, " \t\r\n") // Remove all the trailing shit
		splittedShell := strings.Split(script.shell, " ")         // Split it to find options
		script.shell = splittedShell[0]
		script.opts = splittedShell[1:]
		// if it's a shell, we add set -e to failed job when a command is failed
		if !isWindows() && isShell(script.shell) && len(splittedShell) == 1 {
			script.opts = []string{"-e"}
		}
		if isWindows() && isPowerShell(script.shell) && len(splittedShell) == 1 {
			script.opts = []string{"-ExecutionPolicy", "Bypass", "-Command"}
		}
		if len(t) > 1 {
			scriptContent = t[1]
		}
	}

	script.content = []byte(scriptContent)
	script.dir = workingDir

	return &script
}

func writeScriptContent(ctx context.Context, script *script, fs afero.Fs) (func(), error) {
	workDir, err := fs.Open(script.dir)
	if err != nil {
		return nil, errors.Errorf("unable to open working directory %s [%s]: %v", script.dir, filepath.Base(script.dir), err)
	}
	fi, err := workDir.Stat()
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	if !fi.IsDir() {
		return nil, errors.Errorf("working directory %s is not a directory: %v", script.dir, err)
	}

	// Generate a random string 16 chars length
	bs := make([]byte, 16)
	if _, err := rand.Read(bs); err != nil {
		return nil, sdk.WithStack(err)
	}
	tmpFileName := hex.EncodeToString(bs)[0:16]

	if isWindows() {
		tmpFileName += ".PS1"
	}

	scriptPath := filepath.Join(path.Dir(script.dir), tmpFileName)

	tmpscript, err := fs.OpenFile(scriptPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0700)
	if err != nil {
		return nil, errors.Errorf("cannot create temporary file, aborting: %v", err)
	}

	// Put script in file
	n, err := tmpscript.Write(script.content)
	if err != nil || n != len(script.content) {
		if err != nil {
			return nil, errors.Errorf("unable to create script: %v", err)
		} else {
			return nil, errors.Errorf("unable to write all script: %d/%d", n, len(script.content))
		}
	}

	if err := tmpscript.Close(); err != nil {
		return nil, errors.Errorf("unable to write script to %s", tmpscript)
	}

	var realScriptPath = scriptPath

	switch x := fs.(type) {
	case *afero.BasePathFs:
		realScriptPath, err = x.RealPath(tmpscript.Name())
		if err != nil {
			return nil, errors.Errorf("unable to get script working dir: %v", err)
		}
		realScriptPath, err = filepath.Abs(realScriptPath)
		if err != nil {
			return nil, errors.Errorf("unable to get script working dir: %v", err)
		}
	}

	if isWindows() && isPowerShell(script.shell) {
		// This aims to stop a the very first error and return the right exit code
		psCommand := fmt.Sprintf("& { $ErrorActionPreference='Stop'; & %s ;exit $LastExitCode}", realScriptPath)
		script.opts = append(script.opts, psCommand)
	} else {
		script.opts = append(script.opts, realScriptPath)
	}

	deferFunc := func() {
		filename := filepath.Join(path.Dir(script.dir), tmpFileName)
		_ = fs.Remove(filename)
	}

	// Chmod file
	if err := fs.Chmod(tmpscript.Name(), 0755); err != nil {
		return deferFunc, errors.Errorf("cannot chmod script %s: %v, aborting", tmpscript.Name(), err)
	}
	return deferFunc, nil
}

func isWindows() bool {
	return sdk.GOOS == "windows" || runtime.GOOS == "windows" || os.Getenv("CDS_WORKER_PSHELL_MODE") == "true"
}

func isShell(in string) bool {
	for _, v := range []string{"ksh", "bash", "sh", "zsh"} {
		if strings.HasSuffix(in, v) {
			return true
		}
	}
	return false
}

func isPowerShell(in string) bool {
	for _, v := range []string{"PowerShell", "pwsh.exe", "powershell.exe"} {
		if strings.HasSuffix(in, v) {
			return true
		}
	}
	return false
}
