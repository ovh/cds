package action

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kardianos/osext"
	"github.com/spf13/afero"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type script struct {
	dir     string
	shell   string
	content []byte
	opts    []string
}

func prepareScriptContent(parameters []sdk.Parameter, basedir afero.Fs, workdir afero.File) (*script, error) {
	var script = script{
		shell: "/bin/sh",
	}

	// Get script content
	var scriptContent string
	a := sdk.ParameterFind(parameters, "script")
	scriptContent = a.Value

	// except on windows where it's powershell
	if isWindows() {
		script.shell = "PowerShell"
		script.opts = []string{"-ExecutionPolicy", "Bypass", "-Command"}
		// on windows, we add ErrorActionPreference just below
	} else if strings.HasPrefix(scriptContent, "#!") { // If user wants a specific shell, use it
		t := strings.SplitN(scriptContent, "\n", 2)
		script.shell = strings.TrimPrefix(t[0], "#!")             // Find out the shebang
		script.shell = strings.TrimRight(script.shell, " \t\r\n") // Remove all the trailing shit
		splittedShell := strings.Split(script.shell, " ")         // Split it to find options
		script.shell = splittedShell[0]
		script.opts = splittedShell[1:]
		// if it's a shell, we add set -e to failed job when a command is failed
		if isShell(script.shell) && len(splittedShell) == 1 {
			script.opts = append(script.opts, "-e")
		}
		if len(t) > 1 {
			scriptContent = t[1]
		}

	} else {
		script.opts = []string{"-e"}
	}

	script.content = []byte(scriptContent)

	if x, ok := basedir.(*afero.BasePathFs); ok {
		script.dir, _ = x.RealPath(workdir.Name())
	} else {
		script.dir = workdir.Name()
	}

	log.Debug("prepareScriptContent> script.dir is %s", script.dir)

	return &script, nil
}

func isWindows() bool {
	return sdk.GOOS == "windows" || runtime.GOOS == "windows" || os.Getenv("CDS_WORKER_PSHELL_MODE") == "true"
}

func writeScriptContent(ctx context.Context, script *script, fs afero.Fs, basedir afero.File) (func(), error) {
	fi, err := basedir.Stat()
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		panic("basedir is not a directory")
	}

	// Create a tmp file

	// Generate a random string 16 chars length
	bs := make([]byte, 16)
	if _, err := rand.Read(bs); err != nil {
		return nil, err
	}
	tmpFileName := hex.EncodeToString(bs)[0:16]
	log.Debug("writeScriptContent> Basedir name is %s (%T)", basedir.Name(), basedir)

	if isWindows() {
		tmpFileName += ".PS1"
		log.Debug("runScriptAction> renaming powershell script to %s", tmpFileName)
	}

	scriptPath := filepath.Join(path.Dir(basedir.Name()), tmpFileName)
	log.Debug("writeScriptContent> Opening file %s", scriptPath)

	tmpscript, err := fs.OpenFile(scriptPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0700)
	if err != nil {
		log.Warning(ctx, "writeScriptContent> Cannot create tmp file: %s", err)
		return nil, fmt.Errorf("cannot create temporary file, aborting: %v", err)
	}
	log.Debug("runScriptAction> writeScriptContent> Writing script to %s", tmpscript.Name())

	// Put script in file
	n, errw := tmpscript.Write(script.content)
	if errw != nil || n != len(script.content) {
		if errw != nil {
			log.Warning(ctx, "writeScriptContent> cannot write script: %s", errw)
		} else {
			log.Warning(ctx, "writeScriptContent> cannot write all script: %d/%d", n, len(script.content))
		}
		return nil, errors.New("cannot write script in temporary file, aborting")
	}

	if err := tmpscript.Close(); err != nil {
		return nil, fmt.Errorf("unable to write script to %s", tmpscript)
	}

	var realScriptPath = scriptPath

	switch x := fs.(type) {
	case *afero.BasePathFs:
		realScriptPath, err = x.RealPath(tmpscript.Name())
		if err != nil {
			return nil, fmt.Errorf("unable to get script working dir: %v", err)
		}
		realScriptPath, err = filepath.Abs(realScriptPath)
		if err != nil {
			return nil, fmt.Errorf("unable to get script working dir: %v", err)
		}
	}

	if isWindows() {
		//This aims to stop a the very first error and return the right exit code
		psCommand := fmt.Sprintf("& { $ErrorActionPreference='Stop'; & %s ;exit $LastExitCode}", realScriptPath)
		script.opts = append(script.opts, psCommand)
	} else {
		script.opts = append(script.opts, realScriptPath)
	}

	log.Debug("writeScriptContent> script realpath is %s", realScriptPath)
	log.Debug("writeScriptContent> script directory is %s", script.dir)

	deferFunc := func() {
		filename := filepath.Join(path.Dir(basedir.Name()), tmpFileName)
		log.Debug("writeScriptContent> removing file %s", filename)
		if err := fs.Remove(filename); err != nil {
			log.Error(ctx, "unable to remove %s: %v", filename, err)
		}
	}

	// Chmod file
	if err := fs.Chmod(tmpscript.Name(), 0755); err != nil {
		log.Warning(ctx, "runScriptAction> cannot chmod script %s: %s", tmpscript.Name(), err)
		return deferFunc, fmt.Errorf("cannot chmod script %s: %v, aborting", tmpscript.Name(), err)
	}

	return deferFunc, nil
}

func RunScriptAction(ctx context.Context, wk workerruntime.Runtime, a sdk.Action, secrets []sdk.Variable) (sdk.Result, error) {
	chanRes := make(chan sdk.Result)
	chanErr := make(chan error)

	workdir, err := workerruntime.WorkingDirectory(ctx)
	if err != nil {
		return sdk.Result{}, err
	}

	go func() {
		res := sdk.Result{Status: sdk.StatusSuccess}
		script, err := prepareScriptContent(a.Parameters, wk.BaseDir(), workdir)
		if err != nil {
			chanErr <- err
			res.Status = sdk.StatusFail
			chanRes <- res
			return
		}

		deferFunc, err := writeScriptContent(ctx, script, wk.BaseDir(), workdir)
		if deferFunc != nil {
			defer deferFunc()
		}
		if err != nil {
			chanErr <- err
			res.Status = sdk.StatusFail
			chanRes <- res
			return
		}

		log.Info(ctx, "runScriptAction> Running command %s %s in %s", script.shell, strings.Trim(fmt.Sprint(script.opts), "[]"), script.dir)
		cmd := exec.CommandContext(ctx, script.shell, script.opts...)
		res.Status = sdk.StatusUnknown

		cmd.Dir = script.dir
		cmd.Env = wk.Environ()

		workerpath, err := osext.Executable()
		if err != nil {
			chanErr <- fmt.Errorf("Failure due to internal error (Worker Path): %v", err)
			res.Status = sdk.StatusFail
			chanRes <- res
			return
		}

		log.Debug("runScriptAction> Worker binary path: %s", path.Dir(workerpath))
		for i := range cmd.Env {
			if strings.HasPrefix(cmd.Env[i], "PATH") {
				cmd.Env[i] = fmt.Sprintf("%s:%s", cmd.Env[i], path.Dir(workerpath))
				break
			}
		}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			chanErr <- fmt.Errorf("Failure due to internal error: unable to capture stdout: %v", err)
			res.Status = sdk.StatusFail
			chanRes <- res
			return
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			chanErr <- fmt.Errorf("Failure due to internal error: unable to capture stderr: %v", err)
			res.Status = sdk.StatusFail
			chanRes <- res
			return
		}

		stdoutreader := bufio.NewReader(stdout)
		stderrreader := bufio.NewReader(stderr)

		outchan := make(chan bool)
		go func() {
			for {
				line, errs := stdoutreader.ReadString('\n')
				if line != "" {
					wk.SendLog(ctx, workerruntime.LevelInfo, line)
				}
				if errs != nil {
					stdout.Close()
					close(outchan)
					return
				}
			}
		}()

		errchan := make(chan bool)
		go func() {
			for {
				line, errs := stderrreader.ReadString('\n')
				if line != "" {
					wk.SendLog(ctx, workerruntime.LevelWarn, line)
				}
				if errs != nil {
					stderr.Close()
					close(errchan)
					return
				}
			}
		}()

		if err := cmd.Start(); err != nil {
			chanErr <- fmt.Errorf("unable to start command: %v", err)
			res.Status = sdk.StatusFail
			chanRes <- res
			return
		}

		<-outchan
		<-errchan
		if err := cmd.Wait(); err != nil {
			chanErr <- fmt.Errorf("command failure: %v", err)
		}

		res.Status = sdk.StatusSuccess
		chanRes <- res
	}()

	var res sdk.Result
	var globalErr error
	// Wait for a result
	select {
	case <-ctx.Done():
		log.Error(ctx, "CDS Worker execution canceled: %v", ctx.Err())
		return res, errors.New("CDS Worker execution canceled")
	case res = <-chanRes:
	case globalErr = <-chanErr:
	}

	log.Info(ctx, "runScriptAction> %s %s", res.Status, res.Reason)
	return res, globalErr
}

func isShell(in string) bool {
	for _, v := range []string{"ksh", "bash", "sh", "zsh"} {
		if strings.HasSuffix(in, v) {
			return true
		}
	}
	return false
}
