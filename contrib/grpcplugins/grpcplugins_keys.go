package grpcplugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
	"github.com/ovh/cds/sdk/vcs"
	"github.com/spf13/afero"
)

func InstallSSHKey(ctx context.Context, actPlug *actionplugin.Common, workDirs *sdk.WorkerDirectories, keyName, sshFilePath, privateKey string) (string, error) {
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
			return pathToLog, fmt.Errorf("unable to compute ssh key absolute path: %v", err)
		}
	}

	destinationDirectory := filepath.Dir(absPath)
	if err := afero.NewOsFs().MkdirAll(destinationDirectory, os.FileMode(0755)); err != nil {
		return pathToLog, fmt.Errorf("unable to create directory %s: %v", destinationDirectory, err)
	}

	if err := vcs.WriteKey(afero.NewOsFs(), absPath, privateKey); err != nil {
		return pathToLog, fmt.Errorf("cannot setup ssh key %s : %v", keyName, err)
	}
	Logf(actPlug, "sshkey %s has been created here: %s", keyName, sshFilePath)
	return pathToLog, nil
}

func InstalGPGKey(ctx context.Context, actPlug *actionplugin.Common, workDirs *sdk.WorkerDirectories, keyName, sshFilePath, privateKey string) (string, error) {
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
			return pathToLog, fmt.Errorf("unable to compute ssh key absolute path: %v", err)
		}
	}

	destinationDirectory := filepath.Dir(absPath)
	if err := afero.NewOsFs().MkdirAll(destinationDirectory, os.FileMode(0755)); err != nil {
		return pathToLog, fmt.Errorf("unable to create directory %s: %v", destinationDirectory, err)
	}

	if err := vcs.WriteKey(afero.NewOsFs(), absPath, privateKey); err != nil {
		return pathToLog, fmt.Errorf("cannot setup ssh key %s : %v", keyName, err)
	}
	Logf(actPlug, "sshkey %s has been created here: %s", keyName, sshFilePath)
	return pathToLog, nil
}
