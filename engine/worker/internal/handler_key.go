package internal

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/vcs"
)

var keysDirectory string

func keyInstallHandler(wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		keyName := vars["key"]
		fileName := r.FormValue("file")
		var key *sdk.Variable

		if wk.currentJob.secrets == nil {
			err := sdk.Error{
				Message: "Cannot find any keys for your job",
				Status:  http.StatusBadRequest,
			}
			log.Error("%v", err)
			writeJSON(w, err, err.Status)
			return
		}

		for _, k := range wk.currentJob.secrets {
			if k.Name == ("cds.key." + keyName + ".priv") {
				key = &k
				break
			}
		}

		if key == nil {
			err := sdk.Error{
				Message: fmt.Sprintf("Key %s not found", keyName),
				Status:  http.StatusNotFound,
			}
			log.Error("%v", err)
			writeJSON(w, err, err.Status)
			return
		}

		var installedKeyPath string

		switch key.Type {
		case sdk.KeyTypeSSH:
			if fileName == "" {
				installedKeyPath = path.Join(keysDirectory, key.Name)
				if err := vcs.CleanAllSSHKeys(keysDirectory); err != nil {
					errClean := sdk.Error{
						Message: fmt.Sprintf("Cannot clean ssh keys : %v", err),
						Status:  http.StatusInternalServerError,
					}
					log.Error("%v", errClean)
					writeJSON(w, errClean, errClean.Status)
					return
				}

				if err := vcs.SetupSSHKey(keysDirectory, *key); err != nil {
					errSetup := sdk.Error{
						Message: fmt.Sprintf("Cannot setup ssh key %s : %v", keyName, err),
						Status:  http.StatusInternalServerError,
					}
					log.Error("%v", errSetup)
					writeJSON(w, errSetup, errSetup.Status)
					return
				}
			} else {
				if err := vcs.WriteKey(fileName, key.Value); err != nil {
					errSetup := sdk.Error{
						Message: fmt.Sprintf("Cannot setup ssh key %s : %v", keyName, err),
						Status:  http.StatusInternalServerError,
					}
					log.Error("%v", errSetup)
					writeJSON(w, errSetup, errSetup.Status)
					return
				}
				installedKeyPath = fileName
			}

			writeJSON(w, workerruntime.KeyResponse{
				PKey: installedKeyPath,
				Type: sdk.KeyTypeSSH,
			}, http.StatusOK)

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
					log.Error("%v", errBinary)
					writeJSON(w, errBinary, errBinary.Status)
					return
				}
			}
			content := []byte(key.Value)
			tmpfile, errTmpFile := ioutil.TempFile("", key.Name)
			if errTmpFile != nil {
				errFile := sdk.Error{
					Message: fmt.Sprintf("Cannot setup pgp key %s : %v", key.Name, errTmpFile),
					Status:  http.StatusInternalServerError,
				}
				log.Error("%v", errFile)
				writeJSON(w, errFile, errFile.Status)
				return
			}
			defer func() {
				_ = os.Remove(tmpfile.Name())
			}()

			if _, err := tmpfile.Write(content); err != nil {
				errW := sdk.Error{
					Message: fmt.Sprintf("Cannot setup pgp key file %s : %v", key.Name, err),
					Status:  http.StatusInternalServerError,
				}
				log.Error("%v", errW)
				writeJSON(w, errW, errW.Status)
				return
			}

			if err := tmpfile.Close(); err != nil {
				errC := sdk.Error{
					Message: fmt.Sprintf("Cannot setup pgp key file %s (close) : %v", key.Name, err),
					Status:  http.StatusInternalServerError,
				}
				log.Error("%v", errC)
				writeJSON(w, errC, errC.Status)
				return
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
				log.Error("%v", errR)
				writeJSON(w, errR, errR.Status)
				return
			}
			writeJSON(w, workerruntime.KeyResponse{Type: sdk.KeyTypePGP}, http.StatusOK)
		default:
			err := sdk.Error{
				Message: fmt.Sprintf("Type key %s is not implemented", key.Type),
				Status:  http.StatusNotImplemented,
			}
			log.Error("%v", err)
			writeJSON(w, err, err.Status)
		}
	}
}
