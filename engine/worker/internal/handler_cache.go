package internal

import (
	"archive/tar"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func cachePushHandler(wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		// Get body
		data, errRead := ioutil.ReadAll(r.Body)
		if errRead != nil {
			errRead = sdk.Error{
				Message: "worker cache push > Cannot read body : " + errRead.Error(),
				Status:  http.StatusInternalServerError,
			}
			log.Error("%v", errRead)
			writeError(w, r, errRead)
			return
		}

		var c sdk.Cache
		if err := json.Unmarshal(data, &c); err != nil {
			err = sdk.Error{
				Message: "worker cache push > Cannot unmarshall body : " + err.Error(),
				Status:  http.StatusInternalServerError,
			}
			log.Error("%v", err)
			writeError(w, r, err)
			return
		}

		res, size, errTar := sdk.CreateTarFromPaths(wk.Workspace(), c.WorkingDirectory, c.Files, nil)
		if errTar != nil {
			errTar = sdk.Error{
				Message: "worker cache push > Cannot tar : " + errTar.Error(),
				Status:  http.StatusBadRequest,
			}
			log.Error("%v", errTar)
			writeError(w, r, errTar)
			return
		}
		params := wk.currentJob.wJob.Parameters
		projectKey := sdk.ParameterValue(params, "cds.project")
		if projectKey == "" {
			errP := sdk.Error{
				Message: "worker cache push > Cannot find project",
				Status:  http.StatusInternalServerError,
			}
			log.Error("%v", errP)
			writeError(w, r, errP)
			return
		}

		var errPush error
		for i := 0; i < 10; i++ {
			if errPush = wk.client.WorkflowCachePush(projectKey, sdk.DefaultIfEmptyStorage(c.IntegrationName), vars["ref"], res, size); errPush == nil {
				return
			}
			time.Sleep(3 * time.Second)
			log.Error("worker cache push > cannot push cache (retry x%d) : %v", i, errPush)
		}

		err := sdk.Error{
			Message: "worker cache push > Cannot push cache: " + errPush.Error(),
			Status:  http.StatusInternalServerError,
		}
		log.Error("%v", err)
		writeError(w, r, err)
	}
}

func cachePullHandler(wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		path := r.FormValue("path")
		integrationName := sdk.DefaultIfEmptyStorage(r.FormValue("integration"))
		params := wk.currentJob.wJob.Parameters
		projectKey := sdk.ParameterValue(params, "cds.project")
		bts, err := wk.client.WorkflowCachePull(projectKey, integrationName, vars["ref"])
		if err != nil {
			err = sdk.Error{
				Message: "worker cache pull > Cannot pull cache: " + err.Error(),
				Status:  http.StatusNotFound,
			}
			writeError(w, r, err)
			return
		}

		tr := tar.NewReader(bts)
		for {
			header, errH := tr.Next()
			if errH == io.EOF {
				break
			}

			if errH != nil {
				errH = sdk.Error{
					Message: "worker cache pull > Unable to read tar file: " + errH.Error(),
					Status:  http.StatusBadRequest,
				}
				writeJSON(w, errH, http.StatusBadRequest)
				return
			}

			if header == nil {
				continue
			}

			// the target location where the dir/file should be created
			target := filepath.Join(path, header.Name)

			// check the file type
			switch header.Typeflag {
			// if its a dir and it doesn't exist create it
			case tar.TypeDir:
				if _, err := os.Stat(target); err != nil {
					if err := os.MkdirAll(target, 0755); err != nil {
						err = sdk.Error{
							Message: "worker cache pull > Unable to mkdir all files : " + err.Error(),
							Status:  http.StatusInternalServerError,
						}
						writeJSON(w, err, http.StatusInternalServerError)
						return
					}
				}
			case tar.TypeSymlink:
				if err := os.Symlink(header.Linkname, target); err != nil {
					err = sdk.Error{
						Message: "worker cache pull > Unable to create symlink: " + err.Error(),
						Status:  http.StatusInternalServerError,
					}
					writeJSON(w, err, http.StatusInternalServerError)
					return
				}

				// if it's a file create it
			case tar.TypeReg, tar.TypeLink:
				// if directory of file does not exist, create it before
				if _, err := os.Stat(filepath.Dir(target)); err != nil {
					if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
						err = sdk.Error{
							Message: "worker cache pull > Unable to mkdir all files : " + err.Error(),
							Status:  http.StatusInternalServerError,
						}
						writeJSON(w, err, http.StatusInternalServerError)
						return
					}
				}

				f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, os.FileMode(header.Mode))
				if err != nil {
					sdkErr := sdk.Error{
						Message: "worker cache pull > Unable to open file: " + err.Error(),
						Status:  http.StatusInternalServerError,
					}
					writeJSON(w, sdkErr, sdkErr.Status)
					return
				}

				// copy over contents
				if _, err := io.Copy(f, tr); err != nil {
					_ = f.Close()
					sdkErr := sdk.Error{
						Message: "worker cache pull > Cannot copy content file: " + err.Error(),
						Status:  http.StatusInternalServerError,
					}
					writeJSON(w, sdkErr, sdkErr.Status)
					return
				}

				_ = f.Close()
			}
		}
	}
}
