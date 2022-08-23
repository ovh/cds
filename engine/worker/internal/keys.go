package internal

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/vcs"

	"github.com/spf13/afero"
)

func (wk *CurrentWorker) InstallKey(key sdk.Variable) (*workerruntime.KeyResponse, error) {
	switch key.Type {
	case string(sdk.KeyTypeSSH):
		keysDirectory, err := workerruntime.KeysDirectory(wk.currentJob.context)
		if err != nil {
			return nil, sdk.WithStack(err)
		}

		installedKeyPath := path.Join(keysDirectory.Name(), key.Name)
		if err := vcs.CleanAllSSHKeys(wk.basedir, keysDirectory.Name()); err != nil {
			return nil, sdk.NewError(sdk.ErrUnknownError, fmt.Errorf("Cannot clean ssh keys : %v", err))
		}

		if err := vcs.SetupSSHKey(wk.basedir, keysDirectory.Name(), key); err != nil {
			return nil, sdk.NewError(sdk.ErrUnknownError, fmt.Errorf("Cannot setup ssh key %s : %v", key.Name, err))
		}

		if x, ok := wk.BaseDir().(*afero.BasePathFs); ok {
			installedKeyPath, _ = x.RealPath(installedKeyPath)
		}

		return &workerruntime.KeyResponse{
			PKey:    installedKeyPath,
			Type:    sdk.KeyTypeSSH,
			Content: []byte(key.Value),
		}, nil

	case string(sdk.KeyTypePGP):
		gpg2Found := false

		if _, err := exec.LookPath("gpg2"); err == nil {
			gpg2Found = true
		}

		if !gpg2Found {
			if _, err := exec.LookPath("gpg"); err != nil {
				return nil, sdk.NewError(sdk.ErrWorkerErrorCommand, fmt.Errorf("Cannot use gpg in your worker because you haven't gpg or gpg2 binary"))
			}
		}
		content := []byte(key.Value)
		tmpfile, errTmpFile := os.CreateTemp("", key.Name)
		if errTmpFile != nil {
			return nil, sdk.NewError(sdk.ErrWorkerErrorCommand, fmt.Errorf("Cannot setup pgp key %s : %v", key.Name, errTmpFile))
		}
		defer func() {
			_ = os.Remove(tmpfile.Name())
		}()

		if _, err := tmpfile.Write(content); err != nil {
			return nil, sdk.NewError(sdk.ErrWorkerErrorCommand, fmt.Errorf("Cannot setup pgp key file %s : %v", key.Name, err))
		}

		if err := tmpfile.Close(); err != nil {
			return nil, sdk.NewError(sdk.ErrWorkerErrorCommand, fmt.Errorf("Cannot setup pgp key file %s (close) : %v", key.Name, err))
		}

		gpgBin := "gpg"
		if gpg2Found {
			gpgBin = "gpg2"
		}
		cmd := exec.Command(gpgBin, "--import", tmpfile.Name())
		var out bytes.Buffer
		var outErr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &outErr
		if err := cmd.Run(); err != nil {
			outString := string(out.Bytes())
			outErrString := string(outErr.Bytes())
			return nil, sdk.NewError(sdk.ErrWorkerErrorCommand, fmt.Errorf("Cannot import pgp key %s (%v): %s %s", key.Name, err, outString, outErrString))
		}
		return &workerruntime.KeyResponse{
			Type:    sdk.KeyTypePGP,
			PKey:    tmpfile.Name(),
			Content: content,
		}, nil

	default:
		return nil, sdk.NewError(sdk.ErrWorkerErrorCommand, fmt.Errorf("Type key %s is not implemented", key.Type))
	}
}

func (wk *CurrentWorker) InstallKeyTo(key sdk.Variable, destinationPath string) (*workerruntime.KeyResponse, error) {
	switch key.Type {
	case string(sdk.KeyTypeSSH):
		var absPath string
		if x, ok := wk.BaseDir().(*afero.BasePathFs); ok {
			absPath, _ = x.RealPath(destinationPath)
			absPath, _ = filepath.Abs(path.Dir(absPath))
		}

		if !sdk.PathIsAbs(destinationPath) {
			destinationPath = filepath.Join(absPath, filepath.Base(destinationPath))
		}

		destinationDirectory := filepath.Dir(destinationPath)
		if err := afero.NewOsFs().MkdirAll(destinationDirectory, os.FileMode(0755)); err != nil {
			return nil, fmt.Errorf("unable to create directory %s: %v", destinationDirectory, err)
		}

		if err := vcs.WriteKey(afero.NewOsFs(), destinationPath, key.Value); err != nil {
			return nil, sdk.NewError(sdk.ErrWorkerErrorCommand, fmt.Errorf("Cannot setup ssh key %s : %v", key.Name, err))
		}

		return &workerruntime.KeyResponse{
			PKey:    destinationPath,
			Type:    sdk.KeyTypeSSH,
			Content: []byte(key.Value),
		}, nil

	case string(sdk.KeyTypePGP):
		tmpFileName, content, err := sdk.ImportGPGKey("", key.Name, key.Value)
		if err != nil {
			return nil, sdk.NewError(sdk.ErrWorkerErrorCommand, err)
		}
		return &workerruntime.KeyResponse{
			Type:    sdk.KeyTypePGP,
			PKey:    tmpFileName,
			Content: content,
		}, nil

	default:
		return nil, sdk.NewError(sdk.ErrWorkerErrorCommand, fmt.Errorf("Type key %s is not implemented", key.Type))
	}
}
