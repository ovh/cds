package internal

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/rockbears/log"
	"github.com/spf13/afero"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func cachePushHandler(ctx context.Context, wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := workerruntime.SetJobID(ctx, wk.currentJob.wJob.ID)
		ctx = workerruntime.SetStepOrder(ctx, wk.currentJob.currentStepIndex)
		ctx = workerruntime.SetStepName(ctx, wk.currentJob.currentStepName)

		cdnArtifact := wk.FeatureEnabled(sdk.FeatureCDNArtifact)

		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			err = sdk.Error{
				Message: "worker cache push > Cannot read body : " + err.Error(),
				Status:  http.StatusInternalServerError,
			}
			log.Error(ctx, "%v", err)
			writeError(w, r, err)
			return
		}

		var c sdk.Cache
		if err := json.Unmarshal(data, &c); err != nil {
			err = sdk.Error{
				Message: "worker cache push > Cannot unmarshall body : " + err.Error(),
				Status:  http.StatusInternalServerError,
			}
			log.Error(ctx, "%v", err)
			writeError(w, r, err)
			return
		}

		tmpDirectory, err := workerruntime.TmpDirectory(wk.currentJob.context)
		if err != nil {
			err = sdk.Error{
				Message: "worker cache push > Cannot get tmp directory : " + err.Error(),
				Status:  http.StatusInternalServerError,
			}
			log.Error(ctx, "%v", err)
			writeError(w, r, err)
			return
		}

		tarF, err := afero.TempFile(wk.BaseDir(), tmpDirectory.Name(), "tar-")
		if err != nil {
			err = sdk.Error{
				Message: "worker cache push > Cannot create tmp tar file : " + err.Error(),
				Status:  http.StatusInternalServerError,
			}
			log.Error(ctx, "%v", err)
			writeError(w, r, err)
			return
		}

		if err := sdk.CreateTarFromPaths(afero.NewOsFs(), c.WorkingDirectory, c.Files, tarF, nil); err != nil {
			_ = tarF.Close() // nolint
			err = sdk.Error{
				Message: fmt.Sprintf("worker cache push > Cannot tar (%+v) : %v", c.Files, err.Error()),
				Status:  http.StatusBadRequest,
			}
			log.Error(ctx, "%v", err)
			writeError(w, r, err)
			return
		}

		tarInfo, err := tarF.Stat()
		if err != nil {
			_ = tarF.Close() // nolint
			err = sdk.Error{
				Message: "worker cache push > Cannot get tmp tar file info : " + err.Error(),
				Status:  http.StatusInternalServerError,
			}
			log.Error(ctx, "%v", err)
			writeError(w, r, err)
			return
		}

		params := wk.currentJob.wJob.Parameters
		projectKey := sdk.ParameterValue(params, "cds.project")
		if projectKey == "" {
			_ = tarF.Close() // nolint
			err := sdk.Error{
				Message: "worker cache push > Cannot find project",
				Status:  http.StatusInternalServerError,
			}
			log.Error(ctx, "%v", err)
			writeError(w, r, err)
			return
		}

		tarPath := tarF.Name()

		if err := tarF.Close(); err != nil {
			err := sdk.Error{
				Message: "worker cache push > Cannot close file: " + err.Error(),
				Status:  http.StatusInternalServerError,
			}
			log.Error(ctx, "%v", err)
			writeError(w, r, err)
			return
		}

		if !cdnArtifact {
			var errPush error
			for i := 0; i < 10; i++ {
				f, err := wk.BaseDir().Open(tarPath)
				if err != nil {
					err := sdk.Error{
						Message: "worker cache push > Cannot open tar file: " + err.Error(),
						Status:  http.StatusInternalServerError,
					}
					log.Error(ctx, "%v", err)
					writeError(w, r, err)
					return
				}
				if errPush = wk.client.WorkflowCachePush(projectKey, sdk.DefaultIfEmptyStorage(c.IntegrationName), c.Tag, f, int(tarInfo.Size())); errPush == nil {
					return
				}
				log.Error(ctx, "worker cache push > cannot push cache (retry x%d) : %v", i, errPush)
				err = sdk.Error{
					Message: "worker cache push > Cannot push cache: " + errPush.Error(),
					Status:  http.StatusInternalServerError,
				}
				time.Sleep(3 * time.Second)
			}
			log.Error(ctx, "%v", err)
			writeError(w, r, err)
			return
		}

		sig, err := wk.WorkerCacheSignature(c.Tag)
		if err != nil {
			err := sdk.Error{
				Message: "worker cache push > Cannot create signature",
				Status:  http.StatusInternalServerError,
			}
			log.Error(ctx, "%v", err)
			writeError(w, r, err)
			return
		}
		duration, err := wk.client.CDNItemUpload(ctx, wk.cdnHttpAddr, sig, wk.BaseDir(), tarF.Name())
		if err != nil {
			err := sdk.Error{
				Message: "worker cache push > Cannot upload cache: %v" + err.Error(),
				Status:  http.StatusInternalServerError,
			}
			log.Error(ctx, "%v", err)
			writeError(w, r, err)
			return
		}
		wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Cache '%s' uploaded in %.2fs to CDS CDN", c.Tag, duration.Seconds()))
	}
}

func cachePullHandler(ctx context.Context, wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := workerruntime.SetJobID(ctx, wk.currentJob.wJob.ID)
		ctx = workerruntime.SetStepOrder(ctx, wk.currentJob.currentStepIndex)
		ctx = workerruntime.SetStepName(ctx, wk.currentJob.currentStepName)

		vars := mux.Vars(req)
		path := req.FormValue("path")

		cdnArtifact := wk.FeatureEnabled(sdk.FeatureCDNArtifact)
		params := wk.currentJob.wJob.Parameters
		projectKey := sdk.ParameterValue(params, "cds.project")

		var r io.Reader
		if cdnArtifact {
			// Get cache link
			items, err := wk.client.QueueWorkerCacheLink(ctx, wk.currentJob.wJob.ID, vars["ref"])
			if err != nil {
				err = sdk.Error{
					Message: "worker cache pull > Cannot get cache links: " + err.Error(),
					Status:  http.StatusNotFound,
				}
				log.Error(ctx, "%v", err)
				writeError(w, req, err)
				return
			}
			if len(items.Items) != 1 {
				err := sdk.Error{
					Message: "worker cache pull > No unique link found",
					Status:  http.StatusNotFound,
				}
				log.Error(ctx, "%v", err)
				writeError(w, req, err)
				return
			}
			// Download cache
			r, err = wk.client.CDNItemDownload(ctx, wk.cdnHttpAddr, items.Items[0].APIRefHash, sdk.CDNTypeItemWorkerCache)
			if err != nil {
				err = sdk.Error{
					Message: "Cannot pull cache: " + err.Error(),
					Status:  http.StatusNotFound,
				}
				log.Error(ctx, "%v", err)
				writeError(w, req, err)
				return
			}
		} else {
			integrationName := sdk.DefaultIfEmptyStorage(req.FormValue("integration"))
			var err error
			r, err = wk.client.WorkflowCachePull(projectKey, integrationName, vars["ref"])
			if err != nil {
				err = sdk.Error{
					Message: "worker cache pull > Cannot pull cache: " + err.Error(),
					Status:  http.StatusNotFound,
				}
				log.Error(ctx, "%v", err)
				writeError(w, req, err)
				return
			}
		}

		log.Debug(ctx, "cachePullHandler> Start read cache tar")

		tr := tar.NewReader(r)
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
				log.Error(ctx, "%v", errH)
				writeJSON(w, errH, http.StatusBadRequest)
				return
			}

			if header == nil {
				continue
			}

			log.Debug(ctx, "cachePullHandler> Tar contains file %s", header.Name)

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
						log.Error(ctx, "%v", err)
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
					log.Error(ctx, "%v", err)
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
						log.Error(ctx, "%v", err)
						writeJSON(w, err, http.StatusInternalServerError)
						return
					}
				}

				log.Debug(ctx, "cachePullHandler> Create file at %s", target)

				f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, os.FileMode(header.Mode))
				if err != nil {
					sdkErr := sdk.Error{
						Message: "worker cache pull > Unable to open file: " + err.Error(),
						Status:  http.StatusInternalServerError,
					}
					log.Error(ctx, "%v", err)
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
					log.Error(ctx, "%v", err)
					writeJSON(w, sdkErr, sdkErr.Status)
					return
				}

				_ = f.Close()
			}
		}
	}
}
