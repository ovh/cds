package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	"github.com/kardianos/osext"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func runScriptAction(w *currentWorker) BuiltInAction {
	return func(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, sendLog LoggerFunc) sdk.Result {
		log.Debug("runScriptAction> Begin %p", ctx)
		defer func() {
			log.Debug("runScriptAction> End %p (%s)", ctx, ctx.Err())
		}()
		chanRes := make(chan sdk.Result)

		go func() {
			res := sdk.Result{Status: sdk.StatusSuccess.String()}

			// Get script content
			var scriptContent string
			a := sdk.ParameterFind(a.Parameters, "script")
			scriptContent = a.Value

			// Check that script content is there
			if scriptContent == "" {
				res.Status = sdk.StatusFail.String()
				res.Reason = fmt.Sprintf("script content not provided, aborting\n")
				sendLog(res.Reason)
				chanRes <- res
			}

			// Default shell is sh
			shell := "/bin/sh"
			var opts []string

			// except on windows where it's powershell
			if runtime.GOOS == "windows" {
				shell = "PowerShell"
				opts = append(opts, "-ExecutionPolicy", "Bypass", "-Command")
				// on windows, we add ErrorActionPreference just below
			} else if strings.HasPrefix(scriptContent, "#!") { // If user wants a specific shell, use it
				t := strings.SplitN(scriptContent, "\n", 2)
				shell = strings.TrimPrefix(t[0], "#!")
				shell = strings.TrimRight(shell, " \t\r\n")

				// if it's a shell, we add set -e to failed job when a command is failed
				if isShell(shell) && len(t) >= 2 {
					// there is a shebang, we add set -e add first line after shebang.
					t[1] = fmt.Sprintf("set -e; \n%s", t[1])
				}
				scriptContent = strings.Join(t, "\n")
			} else {
				// no specified shebang, we add set -e; at the beginning
				scriptContent = fmt.Sprintf("set -e; \n%s", scriptContent)
			}

			// Create a tmp file
			tmpscript, errt := ioutil.TempFile(w.basedir, "cds-")
			if errt != nil {
				log.Warning("Cannot create tmp file: %s", errt)
				res.Reason = fmt.Sprintf("cannot create temporary file, aborting\n")
				sendLog(res.Reason)
				res.Status = sdk.StatusFail.String()
				chanRes <- res
			}

			// Put script in file
			n, errw := tmpscript.Write([]byte(scriptContent))
			if errw != nil || n != len(scriptContent) {
				if errw != nil {
					log.Warning("Cannot write script: %s", errw)
				} else {
					log.Warning("cannot write all script: %d/%d", n, len(scriptContent))
				}
				res.Reason = fmt.Sprintf("cannot write script in temporary file, aborting\n")
				sendLog(res.Reason)
				res.Status = sdk.StatusFail.String()
				chanRes <- res
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
					res.Status = sdk.StatusFail.String()
					res.Reason = fmt.Sprintf("cannot rename script to add powershell Extension, aborting\n")
					sendLog(res.Reason)
					chanRes <- res
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
			if err := os.Chmod(scriptPath, 0755); err != nil {
				log.Warning("runScriptAction> cannot chmod script %s: %s", scriptPath, err)
				res.Reason = fmt.Sprintf("cannot chmod script, aborting")
				sendLog(res.Reason)
				res.Status = sdk.StatusFail.String()
				chanRes <- res
			}

			log.Info("runScriptAction> %s %s", shell, strings.Trim(fmt.Sprint(opts), "[]"))
			cmd := exec.CommandContext(ctx, shell, opts...)
			res.Status = sdk.StatusUnknown.String()

			env := os.Environ()
			cmd.Env = []string{}
			// filter technical env variables
			for _, e := range env {
				if strings.HasPrefix(e, "CDS_") {
					continue
				}
				cmd.Env = append(cmd.Env, e)
			}

			//We have to let it here for some legacy reason
			cmd.Env = append(cmd.Env, "CDS_KEY=********")

			// worker export http port
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%d", WorkerServerPort, w.exportPort))

			//DEPRECATED - BEGIN
			// manage keys
			if w.currentJob.pkey != "" && w.currentJob.gitsshPath != "" {
				cmd.Env = append(cmd.Env, fmt.Sprintf("PKEY=%s", w.currentJob.pkey))
				cmd.Env = append(cmd.Env, fmt.Sprintf("GIT_SSH=%s", w.currentJob.gitsshPath))
			}
			//DEPRECATED - END

			//set up environment variables from pipeline build job parameters
			for _, p := range *params {
				// avoid put private key in environment var as it's a binary value
				if (p.Type == sdk.KeyPGPParameter || p.Type == sdk.KeySSHParameter) && strings.HasSuffix(p.Name, ".priv") {
					continue
				}
				if p.Type == sdk.KeyParameter && !strings.HasSuffix(p.Name, ".pub") {
					continue
				}
				envName := strings.Replace(p.Name, ".", "_", -1)
				envName = strings.ToUpper(envName)
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", envName, p.Value))
			}

			for _, p := range w.currentJob.buildVariables {
				envName := strings.Replace(p.Name, ".", "_", -1)
				envName = strings.ToUpper(envName)
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", envName, p.Value))
			}

			workerpath, err := osext.Executable()
			if err != nil {
				log.Warning("runScriptAction: Cannot get worker path: %s", err)
				res.Reason = "Failure due to internal error (Worker Path)"
				sendLog(res.Reason)
				res.Status = sdk.StatusFail.String()
				chanRes <- res
			}

			log.Info("Worker binary path: %s", path.Dir(workerpath))
			for i := range cmd.Env {
				if strings.HasPrefix(cmd.Env[i], "PATH") {
					cmd.Env[i] = fmt.Sprintf("%s:%s", cmd.Env[i], path.Dir(workerpath))
					break
				}
			}

			stdout, err := cmd.StdoutPipe()
			if err != nil {
				log.Warning("runScriptAction: Cannot get stdout pipe: %s", err)
				res.Reason = "Failure due to internal error"
				sendLog(res.Reason)
				res.Status = sdk.StatusFail.String()
				chanRes <- res
			}

			stderr, err := cmd.StderrPipe()
			if err != nil {
				log.Warning("runScriptAction: Cannot get stderr pipe: %s", err)
				res.Reason = "Failure due to internal error"
				sendLog(res.Reason)
				res.Status = sdk.StatusFail.String()
				chanRes <- res
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
					sendLog(line)
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
					sendLog(line)
				}
			}()

			if err := cmd.Start(); err != nil {
				res.Reason = fmt.Sprintf("%s\n", err)
				sendLog(res.Reason)
				res.Status = sdk.StatusFail.String()
				chanRes <- res
			}

			<-outchan
			<-errchan
			if err := cmd.Wait(); err != nil {
				res.Reason = fmt.Sprintf("%s\n", err)
				sendLog(res.Reason)
				res.Status = sdk.StatusFail.String()
				chanRes <- res
			}

			res.Status = sdk.StatusSuccess.String()
			chanRes <- res
		}()

		defer w.drainLogsAndCloseLogger(ctx)

		var res sdk.Result
		// Wait for a result
		select {
		case <-ctx.Done():
			log.Error("CDS Worker execution canceled: %v", ctx.Err())
			sendLog("CDS Worker execution canceled")
			res = sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: "CDS Worker execution canceled",
			}
			break

		case res = <-chanRes:
			break
		}

		log.Info("runScriptAction> %s %s", res.GetStatus(), res.GetReason())
		return res
	}
}

func isShell(in string) bool {
	for _, v := range []string{"ksh", "bash", "sh", "zsh"} {
		if strings.HasSuffix(in, v) {
			return true
		}
	}
	return false
}
