package grpcplugins

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
	"github.com/ovh/cds/sdk/vcs"
	"github.com/spf13/afero"
)

func InstallSSHKey(ctx context.Context, actPlug *actionplugin.Common, workDirs *sdk.WorkerDirectories, keyName, sshFilePath, privateKey, gitURL string) error {
	if sshFilePath == "" {
		sshFilePath = ".ssh/id_rsa-" + keyName
	}
	absPath := sshFilePath
	pathToLog := absPath
	if !sdk.PathIsAbs(sshFilePath) {
		pathToLog = "${{ cds.workspace }}/" + sshFilePath
		var err error
		absPath, err = filepath.Abs(filepath.Join(workDirs.WorkingDir, sshFilePath))
		if err != nil {
			return fmt.Errorf("unable to compute ssh key absolute path: %v", err)
		}
	}

	destinationDirectory := filepath.Dir(absPath)
	if err := afero.NewOsFs().MkdirAll(destinationDirectory, os.FileMode(0755)); err != nil {
		return fmt.Errorf("unable to create directory %s: %v", destinationDirectory, err)
	}

	if err := vcs.WriteKey(afero.NewOsFs(), absPath, privateKey); err != nil {
		return fmt.Errorf("cannot setup ssh key %s : %v", keyName, err)
	}
	Logf(actPlug, "sshkey %s has been created here: %s", keyName, sshFilePath)

	if gitURL == "" {
		Logf(actPlug, "To be able to use git command in a further step, you must run this command first:")
		Successf(actPlug, "export GIT_SSH_COMMAND=\"ssh -i %s -o StrictHostKeyChecking=no\"", pathToLog)
	} else {
		u, err := url.Parse(gitURL)
		if err != nil {
			return fmt.Errorf("unable to parse git url: %s", gitURL)
		}
		host, port, _ := net.SplitHostPort(u.Host)
		if port == "" {
			port = "22"
		}

		goRoutines := sdk.NewGoRoutines(ctx)

		workDirs, err := GetWorkerDirectories(ctx, actPlug)
		if err != nil {
			return fmt.Errorf("unable to get working directory: %v", err)
		}
		scriptContent := fmt.Sprintf("ssh-keyscan -t rsa -p %s %s >> ${HOME}/.ssh/known_hosts", port, host)

		chanRes := make(chan *actionplugin.ActionResult)
		goRoutines.Exec(ctx, "InstallSSHKey-runScript", func(ctx context.Context) {
			RunScript(ctx, actPlug, chanRes, workDirs.WorkingDir, scriptContent)
		})

		select {
		case <-ctx.Done():
			return fmt.Errorf("CDS Worker execution canceled: " + ctx.Err().Error())
		case result := <-chanRes:
			if result.Status == sdk.StatusFail {
				return fmt.Errorf(result.Details)
			}
			return nil
		}
	}
	return nil
}
