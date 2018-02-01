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

	if err := cmd.Run(); err != nil {
		return "", err
	}

	stdOut = buffOut.String()
	stdErr := buffErr.String()

	if len(stdErr) > 0 {
		return stdOut, fmt.Errorf(stdErr)
	}

	return stdOut, nil
}
