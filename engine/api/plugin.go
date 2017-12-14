package api

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/actionplugin"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/plugin"
)

func fileUploadAndGetPlugin(w http.ResponseWriter, r *http.Request) (*sdk.ActionPlugin, *plugin.Parameters, io.ReadCloser, func(), error) {
	r.ParseMultipartForm(64 << 20)
	file, handler, err := r.FormFile("UploadFile")
	if err != nil {
		log.Debug("fileUploadAndGetPlugin> %v", r.Header)
		return nil, nil, nil, nil, sdk.WrapError(err, "fileUploadAndGetPlugin> err on formFile")
	}

	filename := handler.Filename
	t := strings.Split(handler.Filename, "/")
	if len(t) > 1 {
		filename = t[len(t)-1]
	}

	log.Debug("fileUploadAndGetPlugin> file upload detected : %s", filename)
	defer file.Close()

	tmp, err := ioutil.TempDir("", "cds-plugin")
	if err != nil {
		return nil, nil, nil, nil, sdk.WrapError(err, "fileUploadAndGetPlugin> err on temp dir.")
	}
	deferFunc := func() {
		log.Debug("fileUploadAndGetPlugin> deleting file %s", tmp)
		os.RemoveAll(tmp)
	}

	log.Debug("fileUploadAndGetPlugin> creating temporary directory")
	tmpfn := filepath.Join(tmp, filename)
	f, err := os.OpenFile(tmpfn, os.O_WRONLY|os.O_CREATE, 0700)
	if err != nil {
		return nil, nil, nil, deferFunc, sdk.WrapError(err, "fileUploadAndGetPlugin> err on openFile")
	}

	log.Debug("fileUploadAndGetPlugin> writing file %s", tmpfn)
	io.Copy(f, file)
	f.Close()

	content, err := os.Open(tmpfn)
	if err != nil {
		return nil, nil, nil, deferFunc, sdk.WrapError(err, "fileUploadAndGetPlugin> err on Open")
	}

	ap, params, err := actionplugin.Get(filename, tmpfn)
	if err != nil {
		return nil, nil, nil, deferFunc, sdk.WrapError(sdk.ErrPluginInvalid, "fileUploadAndGetPlugin> unable to get plugin info: %s", err)
	}

	return ap, params, content, deferFunc, nil
}

func (api *API) addPluginHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Upload file and get plugin information
		ap, params, file, deferFunc, err := fileUploadAndGetPlugin(w, r)
		if deferFunc != nil {
			defer deferFunc()
		}
		if err != nil {
			return sdk.WrapError(err, "addPluginHandler>%T", err)
		}
		defer file.Close()

		// Check that action does not already exists
		conflict, err := action.Exists(api.mustDB(), ap.Name)
		if err != nil {
			return sdk.WrapError(err, "updatePluginHandler>%T", err)
		}
		if conflict {
			return sdk.ErrConflict
		}

		//Upload it to objectstore
		objectPath, err := objectstore.StorePlugin(*ap, file)
		if err != nil {
			return sdk.WrapError(err, "addPluginHandler> Error while uploading to object store %s", ap.Name)
		}
		ap.ObjectPath = objectPath

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "addPluginHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		//Insert in database
		a, err := actionplugin.Insert(tx, ap, params)
		if err != nil {
			objectstore.DeletePlugin(*ap)
			return sdk.WrapError(err, "addPluginHandler> Error while inserting action %s in database", ap.Name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addPluginHandler> Cannot commit transaction")
		}

		return WriteJSON(w, r, a, http.StatusCreated)
	}
}

func (api *API) updatePluginHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Upload file and get plugin information
		ap, params, file, deferFunc, errUpload := fileUploadAndGetPlugin(w, r)
		if deferFunc != nil {
			defer deferFunc()
		}
		if errUpload != nil {
			return sdk.WrapError(errUpload, "updatePluginHandler> fileUploadAndGetPlugin error")
		}

		// Check that action does not already exists
		exists, errExists := action.Exists(api.mustDB(), ap.Name)
		if errExists != nil {
			return sdk.WrapError(errExists, "updatePluginHandler> unable to check if action %s exists", ap.Name)
		}
		if !exists {
			return sdk.WrapError(sdk.ErrNoAction, "updatePluginHandler")
		}

		//Store previous file from objectstore
		buf, errFetch := objectstore.FetchPlugin(*ap)
		if errFetch != nil {
			log.Warning("updatePluginHandler>Unable to fetch plugin: %s", errFetch)
			// do no raise error... it just mean that we cannot fetch the old version of the plugin
		}

		var tmpFile string
		if buf != nil && errFetch == nil {
			defer buf.Close()
			//Read it
			btes, errr := ioutil.ReadAll(buf)
			if errr != nil {
				return sdk.WrapError(errr, "updatePluginHandler> Unable to read old plugin buffer")
			}
			//Get a dir
			tmpDir, errtmp := ioutil.TempDir("", "old-plugin")
			if errtmp != nil {
				return sdk.WrapError(errtmp, "updatePluginHandler> error with tempdir")
			}
			os.MkdirAll(tmpDir, os.FileMode(0700))

			//Get a temp file
			tmpFile = path.Join(tmpDir, ap.Name)

			//Write it
			log.Debug("updatePluginHandler>store oldfile %s in case of error", tmpFile)
			if err := ioutil.WriteFile(tmpFile, btes, os.FileMode(0600)); err != nil {
				return sdk.WrapError(err, "updatePluginHandler>Error writing file %s", tmpFile)
			}
			defer func() {
				log.Debug("updatePluginHandler> deleting file %s", tmpFile)
				os.RemoveAll(tmpFile)
			}()
			//Delete previous file from objectstore
			if err := objectstore.DeletePlugin(*ap); err != nil {
				return sdk.WrapError(err, "updatePluginHandler>Error deleting file")
			}
		}

		//Upload it to objectstore
		objectPath, errStore := objectstore.StorePlugin(*ap, file)
		if errStore != nil {
			return sdk.WrapError(errStore, "updatePluginHandler> Error while uploading to object store %s", ap.Name)
		}
		ap.ObjectPath = objectPath

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "updatePluginHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		//Update in database
		a, errDB := actionplugin.Update(tx, ap, params, getUser(ctx).ID)
		if errDB != nil && tmpFile != "" {
			log.Warning("updatePluginHandler> Error while updating action %s in database: %s\n", ap.Name, errDB)
			//Restore previous file
			oldFile, errO := os.Open(tmpFile)
			if errO != nil {
				return sdk.WrapError(errO, "updatePluginHandler>Error opening file %s ", tmpFile)
			}
			//re-store the old plugin file
			if _, errStore := objectstore.StorePlugin(*ap, oldFile); errStore != nil {
				return sdk.WrapError(errStore, "updatePluginHandler> Error while uploading to object store %s", ap.Name)
			}

			return sdk.WrapError(errDB, "updatePluginHandler> Unable to update plugin", ap.Name)
		}
		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updatePluginHandler> Cannot commit transaction")
		}

		return WriteJSON(w, r, a, http.StatusOK)
	}
}

func (api *API) deletePluginHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["name"]

		if name == "" {
			return sdk.ErrWrongRequest
		}

		//Delete in database
		if err := actionplugin.Delete(api.mustDB(), name, getUser(ctx).ID); err != nil {
			return sdk.WrapError(err, "deletePluginHandler> Error while deleting action %s in database", name)
		}

		//Delete from objectstore
		if err := objectstore.DeletePlugin(sdk.ActionPlugin{Name: name}); err != nil {
			return sdk.WrapError(err, "deletePluginHandler> Error while deleting action %s in objectstore", name)
		}
		return nil
	}
}

func (api *API) downloadPluginHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["name"]

		if name == "" {
			return sdk.ErrWrongRequest
		}
		p := sdk.ActionPlugin{Name: name}

		acceptRedirect := FormBool(r, "accept-redirect")

		if acceptRedirect {
			url, err := objectstore.FetchTempURL(&p)
			if err == nil {
				http.Redirect(w, r, url, http.StatusTemporaryRedirect)
				return nil
			}
			log.Warning("downloadPluginHandler> Unable to get temp url: %v", err)
		}

		f, err := objectstore.FetchPlugin(p)
		if err != nil {
			return sdk.WrapError(err, "downloadPluginHandler> Error while fetching plugin", name)
		}

		w.Header().Add("Content-Type", "application/octet-stream")
		w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", name))

		if _, err := io.Copy(w, f); err != nil {
			_ = f.Close()
			return sdk.WrapError(err, "downloadPluginHandler> Cannot stream artifact")
		}

		if err := f.Close(); err != nil {
			return sdk.WrapError(err, "downloadPluginHandler> Cannot close artifact")
		}

		return nil
	}
}
