package repo

import (
	"bytes"
	"fmt"
	"os/exec"
)

func (r Repo) runCmd(name string, args ...string) (stdOut string, err error) {
	cmd := exec.Command(name, args...)
	buffOut := new(bytes.Buffer)
	buffErr := new(bytes.Buffer)
	cmd.Dir = r.path
	cmd.Stderr = buffErr
	cmd.Stdout = buffOut

	if r.sshKey != nil {
		envs, err := r.setupSSHKey()
		if err != nil {
			return "", err
		}
		cmd.Env = append(cmd.Env, envs...)
		if r.verbose {
			r.log("Using %v\n", envs)
		}
	}

	if r.verbose {
		r.log("Running command %+v\n", cmd)
	}

	runErr := cmd.Run()

	stdOut = buffOut.String()
	stdErr := buffErr.String()

	if !cmd.ProcessState.Success() {
		if len(stdErr) > 0 {
			return stdOut, fmt.Errorf("%s (%v)", stdErr, runErr)
		}
		return stdOut, fmt.Errorf("exited with error: %v", runErr)
	}

	if runErr != nil {
		return stdOut, fmt.Errorf("%s (%v)", stdErr, runErr)
	}

	return stdOut, nil
}
