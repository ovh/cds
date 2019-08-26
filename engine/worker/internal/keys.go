package internal

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/vcs"
)

func (wk *CurrentWorker) InstallKey(key sdk.Variable, destinationPath string) (*workerruntime.KeyResponse, error) {
	switch key.Type {
	case sdk.KeyTypeSSH:
		if destinationPath == "" {
			keysDirectory, err := workerruntime.KeysDirectory(wk.currentJob.context)
			if err != nil {
				return nil, sdk.WithStack(err)
			}

			installedKeyPath := path.Join(keysDirectory.Name(), key.Name)
			if err := vcs.CleanAllSSHKeys(wk.basedir, keysDirectory.Name()); err != nil {
				errClean := sdk.Error{
					Message: fmt.Sprintf("Cannot clean ssh keys : %v", err),
					Status:  http.StatusInternalServerError,
				}
				return nil, sdk.WithStack(errClean)
			}

			if err := vcs.SetupSSHKey(wk.basedir, keysDirectory.Name(), key); err != nil {
				errSetup := sdk.Error{
					Message: fmt.Sprintf("Cannot setup ssh key %s : %v", key.Name, err),
					Status:  http.StatusInternalServerError,
				}
				return nil, sdk.WithStack(errSetup)
			}

			return &workerruntime.KeyResponse{
				PKey: installedKeyPath,
				Type: sdk.KeyTypeSSH,
			}, nil

		}

		if err := vcs.WriteKey(wk.Workspace(), destinationPath, key.Value); err != nil {
			errSetup := sdk.Error{
				Message: fmt.Sprintf("Cannot setup ssh key %s : %v", key.Name, err),
				Status:  http.StatusInternalServerError,
			}
			return nil, sdk.WithStack(errSetup)
		}

		return &workerruntime.KeyResponse{
			PKey: destinationPath,
			Type: sdk.KeyTypeSSH,
		}, nil

	case sdk.KeyTypePGP:
		gpg2Found := false

		if _, err := exec.LookPath("gpg2"); err == nil {
			gpg2Found = true
		}

		if !gpg2Found {
			if _, err := exec.LookPath("gpg"); err != nil {
				errBinary := sdk.Error{
					Message: fmt.Sprintf("Cannot use gpg in your worker because you haven't gpg or gpg2 binary"),
					Status:  http.StatusBadRequest,
				}
				return nil, sdk.WithStack(errBinary)

			}
		}
		content := []byte(key.Value)
		tmpfile, errTmpFile := ioutil.TempFile("", key.Name)
		if errTmpFile != nil {
			errFile := sdk.Error{
				Message: fmt.Sprintf("Cannot setup pgp key %s : %v", key.Name, errTmpFile),
				Status:  http.StatusInternalServerError,
			}
			return nil, sdk.WithStack(errFile)

		}
		defer func() {
			_ = os.Remove(tmpfile.Name())
		}()

		if _, err := tmpfile.Write(content); err != nil {
			errW := sdk.Error{
				Message: fmt.Sprintf("Cannot setup pgp key file %s : %v", key.Name, err),
				Status:  http.StatusInternalServerError,
			}
			return nil, sdk.WithStack(errW)

		}

		if err := tmpfile.Close(); err != nil {
			errC := sdk.Error{
				Message: fmt.Sprintf("Cannot setup pgp key file %s (close) : %v", key.Name, err),
				Status:  http.StatusInternalServerError,
			}
			return nil, sdk.WithStack(errC)

		}

		gpgBin := "gpg"
		if gpg2Found {
			gpgBin = "gpg2"
		}
		cmd := exec.Command(gpgBin, "--import", tmpfile.Name())
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			errR := sdk.Error{
				Message: fmt.Sprintf("Cannot import pgp key %s : %v", key.Name, err),
				Status:  http.StatusInternalServerError,
			}
			return nil, sdk.WithStack(errR)

		}
		return &workerruntime.KeyResponse{
			Type: sdk.KeyTypePGP,
			PKey: tmpfile.Name(),
		}, nil

	default:
		err := sdk.Error{
			Message: fmt.Sprintf("Type key %s is not implemented", key.Type),
			Status:  http.StatusNotImplemented,
		}
		return nil, sdk.WithStack(err)

	}
}
