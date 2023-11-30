package internal

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

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

		data, err := io.ReadAll(r.Body)
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
		if err := sdk.JSONUnmarshal(data, &c); err != nil {
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

		if _, err = tarF.Stat(); err != nil {
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

		if err := tarF.Close(); err != nil {
			err := sdk.Error{
				Message: "worker cache push > Cannot close file: " + err.Error(),
				Status:  http.StatusInternalServerError,
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
		duration, err := wk.client.CDNItemUpload(ctx, wk.cfg.CDNEndpoint, sig, wk.BaseDir(), tarF.Name())
		if err != nil {
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
		wkDirFS := afero.NewOsFs()
		if err := wkDirFS.MkdirAll(path, os.FileMode(0744)); err != nil {
			newError := sdk.NewError(sdk.ErrInvalidData, fmt.Errorf("unable to create destination directory: %v", err))
			writeError(w, req, newError)
			return
		}

		dest := filepath.Join(path, "workercache.tar")
		f, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(0755))
		if err != nil {
			err = sdk.NewError(sdk.ErrUnknownError, fmt.Errorf("cannot create file (OpenFile) %s: %s", dest, err))
			log.Error(ctx, "%v", err)
			writeError(w, req, err)
			return
		}
		if err := wk.client.CDNItemDownload(ctx, wk.cfg.CDNEndpoint, items.Items[0].APIRefHash, sdk.CDNTypeItemWorkerCache, items.Items[0].MD5, f); err != nil {
			_ = f.Close()
			err = sdk.Error{
				Message: "worker cache pull > Cannot pull cache: " + err.Error(),
				Status:  http.StatusNotFound,
			}
			log.Error(ctx, "%v", err)
			writeError(w, req, err)
			return
		}
		if err := f.Close(); err != nil {
			err = sdk.Error{
				Message: fmt.Sprintf("worker cache pull > unable to close file %s: %v", dest, err),
				Status:  http.StatusInternalServerError,
			}
			log.Error(ctx, "%v", err)
			writeError(w, req, err)
			return
		}

		// Open tar file
		log.Info(ctx, "extracting worker cache %s / %s", dest, vars["ref"])
		archive, err := wkDirFS.Open(dest)
		if err != nil {
			e := sdk.Error{
				Message: "worker cache pull > unable to open archive: " + err.Error(),
				Status:  http.StatusInternalServerError,
			}
			log.Error(ctx, "%v", e)
			writeError(w, req, e)
			return
		}
		if err := extractArchive(ctx, archive, path); err != nil {
			log.Error(ctx, "%v", err)
			writeError(w, req, err)
			return
		}
		if err := wkDirFS.Remove(dest); err != nil {
			e := sdk.Error{
				Message: "unable to remove worker cache archive: " + err.Error(),
				Status:  http.StatusInternalServerError,
			}
			log.Error(ctx, "%v", e)
			writeError(w, req, e)
			return
		}
	}
}

func extractArchive(ctx context.Context, r io.Reader, path string) *sdk.Error {
	tr := tar.NewReader(r)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return &sdk.Error{
				Message: "worker cache pull > Unable to read tar file: " + err.Error(),
				Status:  http.StatusBadRequest,
			}
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
					return &sdk.Error{
						Message: "worker cache pull > Unable to mkdir all files : " + err.Error(),
						Status:  http.StatusInternalServerError,
					}
				}
			}
		case tar.TypeSymlink:
			if err := os.Symlink(header.Linkname, target); err != nil {
				return &sdk.Error{
					Message: "worker cache pull > Unable to create symlink: " + err.Error(),
					Status:  http.StatusInternalServerError,
				}
			}

			// if it's a file create it
		case tar.TypeReg, tar.TypeLink:
			// if directory of file does not exist, create it before
			if _, err := os.Stat(filepath.Dir(target)); err != nil {
				if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
					return &sdk.Error{
						Message: "worker cache pull > Unable to mkdir all files : " + err.Error(),
						Status:  http.StatusInternalServerError,
					}
				}
			}

			log.Debug(ctx, "cachePullHandler> Create file at %s", target)

			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return &sdk.Error{
					Message: "worker cache pull > Unable to open file: " + err.Error(),
					Status:  http.StatusInternalServerError,
				}
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				_ = f.Close()
				return &sdk.Error{
					Message: "worker cache pull > Cannot copy content file: " + err.Error(),
					Status:  http.StatusInternalServerError,
				}
			}
			_ = f.Close()
		}
	}
	return nil
}
