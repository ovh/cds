package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	"github.com/kardianos/osext"

	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func runScriptAction(a *sdk.Action, pbJob sdk.PipelineBuildJob, stepOrder int) sdk.Result {
	res := sdk.Result{Status: sdk.StatusSuccess}

	// Get script content
	var scriptContent string
	for _, a := range a.Parameters {
		if a.Name == "script" {
			scriptContent = a.Value
			break
		}
	}

	// Check that script content is there
	if scriptContent == "" {
		res.Status = sdk.StatusFail
		res.Reason = fmt.Sprintf("script content not provided, aborting\n")
		sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
		return res
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
	tmpscript, err := ioutil.TempFile(os.TempDir(), "cds-")
	if err != nil {
		log.Warning("Cannot create tmp file: %s\n", err)
		res.Reason = fmt.Sprintf("cannot create temporary file, aborting\n")
		sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
		res.Status = sdk.StatusFail
		return res
	}

	// Put script in file
	n, err := tmpscript.Write([]byte(scriptContent))
	if err != nil || n != len(scriptContent) {
		if err != nil {
			log.Warning("Cannot write script: %s\n", err)
		} else {
			log.Warning("cannot write all script: %d/%d\n", n, len(scriptContent))
		}
		res.Reason = fmt.Sprintf("cannot write script in temporary file, aborting\n")
		sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
		res.Status = sdk.StatusFail
		return res
	}

	oldPath := tmpscript.Name()
	tmpscript.Close()
	var scriptPath string
	if runtime.GOOS == "windows" {
		//Remove all .txt Extensions, there is not always a .txt extension
		newPath := strings.Replace(oldPath, ".txt", "", -1)
		//and add .PS1 extension
		newPath = newPath + ".PS1"
		err = os.Rename(oldPath, newPath)
		if err != nil {
			res.Status = sdk.StatusFail
			res.Reason = fmt.Sprintf("cannot rename script to add powershell Extension, aborting\n")
			sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
			return res
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
	err = os.Chmod(scriptPath, 0755)
	if err != nil {
		log.Warning("runScriptAction> cannot chmod script %s: %s\n", scriptPath, err)
		res.Reason = fmt.Sprintf("cannot chmod script, aborting\n")
		sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
		res.Status = sdk.StatusFail
		return res
	}
	log.Notice("runScriptAction> %s %s", shell, strings.Trim(fmt.Sprint(opts), "[]"))
	cmd := exec.Command(shell, opts...)
	res.Status = sdk.StatusUnknown

	env := os.Environ()
	cmd.Env = []string{}
	// filter technical env variables
	for _, e := range env {
		if strings.HasPrefix(e, "CDS_MODEL=") ||
			strings.HasPrefix(e, "CDS_TTL=") ||
			strings.HasPrefix(e, "CDS_SINGLE_USE=") ||
			strings.HasPrefix(e, "CDS_NAME=") ||
			strings.HasPrefix(e, "CDS_KEY=") ||
			strings.HasPrefix(e, "CDS_API=") ||
			strings.HasPrefix(e, "CDS_HATCHERY=") {
			continue
		}
		cmd.Env = append(cmd.Env, e)
	}

	// worker export http port
	cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%d", WorkerServerPort, exportport))

	//DEPRECATED - BEGIN
	// manage keys
	if pkey != "" && gitsshPath != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", pKEY, pkey))
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", gitSSH, gitsshPath))
	}
	//DEPRECATED - END

	//set up environment variables from pipeline build job parameters
	for _, p := range pbJob.Parameters {
		envName := strings.Replace(p.Name, ".", "_", -1)
		envName = strings.ToUpper(envName)
		if sdk.NeedPlaceholder(p.Type) {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", envName, p.Value))
		} else {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", envName, sdk.PasswordPlaceholder))
		}
	}

	workerpath, err := osext.Executable()
	if err != nil {
		log.Warning("runScriptAction: Cannot get worker path: %s\n", err)
		res.Reason = "Failure due to internal error (Worker Path)"
		sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
		res.Status = sdk.StatusFail
		return res
	}
	log.Notice("Worker binary path: %s\n", path.Dir(workerpath))

	for i := range cmd.Env {
		if strings.HasPrefix(cmd.Env[i], "PATH") {
			cmd.Env[i] = fmt.Sprintf("%s:%s", cmd.Env[i], path.Dir(workerpath))
			break
		}
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Warning("runScriptAction: Cannot get stdout pipe: %s\n", err)
		res.Reason = "Failure due to internal error"
		sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
		res.Status = sdk.StatusFail
		return res
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Warning("runScriptAction: Cannot get stderr pipe: %s\n", err)
		res.Reason = "Failure due to internal error"
		sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
		res.Status = sdk.StatusFail
		return res
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
			log.Info(">Line: %s", line)
			sendLog(pbJob.ID, line, pbJob.PipelineBuildID, stepOrder, false)
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
			sendLog(pbJob.ID, line, pbJob.PipelineBuildID, stepOrder, false)
		}
	}()

	err = cmd.Start()
	if err != nil {
		res.Reason = fmt.Sprintf("%s\n", err)
		sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
		res.Status = sdk.StatusFail
		return res
	}

	_ = <-outchan
	_ = <-errchan
	err = cmd.Wait()
	if err != nil {
		res.Reason = fmt.Sprintf("%s\n", err)
		sendLog(pbJob.ID, res.Reason, pbJob.PipelineBuildID, stepOrder, false)
		res.Status = sdk.StatusFail
		return res
	}

	res.Status = sdk.StatusSuccess
	return res
}
