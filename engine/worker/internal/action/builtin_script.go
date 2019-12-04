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

	"github.com/spf13/afero"

	"github.com/kardianos/osext"

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

func prepareScriptContent(parameters []sdk.Parameter) (*script, error) {
	var script = script{
		shell: "/bin/sh",
	}

	// Get script content
	var scriptContent string
	a := sdk.ParameterFind(parameters, "script")
	scriptContent = a.Value

	// Check that script content is there
	if scriptContent == "" {
		return nil, errors.New("script content not provided, aborting")
	}

	// except on windows where it's powershell
	if sdk.GOOS == "windows" {
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
		scriptContent = t[1]
	} else {
		script.opts = []string{"-e"}
	}

	script.content = []byte(scriptContent)

	return &script, nil
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
	path := filepath.Join(basedir.Name(), tmpFileName)

	log.Debug("writeScriptContent> Opening file %s", path)

	tmpscript, err := fs.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0700)
	if err != nil {
		log.Warning(ctx, "writeScriptContent> Cannot create tmp file: %s", err)
		return nil, fmt.Errorf("cannot create temporary file, aborting: %v", err)
	}
	log.Debug("runScriptAction> writeScriptContent> Writing script to %s", tmpscript.Name())

	// Put script in file
	n, errw := tmpscript.Write(script.content)
	if errw != nil || n != len(script.content) {
		if errw != nil {
			log.Warning(ctx, "cannot write script: %s", errw)
		} else {
			log.Warning(ctx, "cannot write all script: %d/%d", n, len(script.content))
		}
		return nil, errors.New("cannot write script in temporary file, aborting")
	}

	if err := tmpscript.Close(); err != nil {
		return nil, fmt.Errorf("unable to write script to %s", tmpscript)
	}

	if runtime.GOOS == "windows" {
		//and add .PS1 extension
		//newName := tmpFileName + ".PS1"
		//if err := basedir.Rename(tmpFileName, newName); err != nil {
		//	return nil, errors.New("cannot rename script to add powershell Extension, aborting")
		//}
		//tmpFileName = newName
		////This aims to stop a the very first error and return the right exit code
		//psCommand := fmt.Sprintf("& { $ErrorActionPreference='Stop'; & %s ;exit $LastExitCode}", tmpFileName)
		//scriptPath = newPath
		//script.opts = append(script.opts, psCommand)
	} else {
		script.opts = append(script.opts, tmpFileName)
		switch x := fs.(type) {
		case *afero.BasePathFs:
			script.dir, err = x.RealPath(basedir.Name())
			if err != nil {
				return nil, fmt.Errorf("unable to get script working dir: %v", err)
			}
		default:
			script.dir = basedir.Name()
		}
	}
	deferFunc := func() {
		if err := fs.Remove(tmpFileName); err != nil {
			log.Error(ctx, "unable to remove %s: %v", tmpFileName, err)
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
		script, err := prepareScriptContent(a.Parameters)
		if err != nil {
			chanErr <- err
		}

		if x, ok := wk.BaseDir().(*afero.BasePathFs); ok {
			script.dir, _ = x.RealPath(workdir.Name())
		} else {
			script.dir = workdir.Name()
		}

		deferFunc, err := writeScriptContent(ctx, script, wk.BaseDir(), workdir)
		if deferFunc != nil {
			defer deferFunc()
		}
		if err != nil {
			chanErr <- err
		}

		log.Info(ctx, "runScriptAction> Running command %s %s in %s", script.shell, strings.Trim(fmt.Sprint(script.opts), "[]"), script.dir)
		cmd := exec.CommandContext(ctx, script.shell, script.opts...)
		res.Status = sdk.StatusUnknown
		cmd.Dir = script.dir
		cmd.Env = wk.Environ()

		workerpath, err := osext.Executable()
		if err != nil {
			chanErr <- fmt.Errorf("Failure due to internal error (Worker Path): %v", err)
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
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			chanErr <- fmt.Errorf("Failure due to internal error: unable to capture stderr: %v", err)
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
				wk.SendLog(ctx, workerruntime.LevelInfo, line)
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
				wk.SendLog(ctx, workerruntime.LevelWarn, line)
			}
		}()

		if err := cmd.Start(); err != nil {
			chanErr <- fmt.Errorf("unable to start command: %v", err)
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
