package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/actionplugin"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/plugin"
)

func fileUploadAndGetPlugin(w http.ResponseWriter, r *http.Request) (*sdk.ActionPlugin, *plugin.Parameters, io.ReadCloser, func(), error) {
	r.ParseMultipartForm(64 << 20)
	file, handler, err := r.FormFile("UploadFile")
	if err != nil {
		log.Warning("fileUploadAndGetPlugin> %s", err)
		log.Debug("fileUploadAndGetPlugin> %v", r.Header)
		return nil, nil, nil, nil, err
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
		log.Critical("fileUploadAndGetPlugin> %s", err)
		return nil, nil, nil, nil, err
	}
	deferFunc := func() {
		log.Debug("fileUploadAndGetPlugin> deleting file %s", tmp)
		os.RemoveAll(tmp)
	}

	log.Debug("fileUploadAndGetPlugin> creating temporary directory")
	tmpfn := filepath.Join(tmp, filename)
	f, err := os.OpenFile(tmpfn, os.O_WRONLY|os.O_CREATE, 0700)
	if err != nil {
		log.Critical("fileUploadAndGetPlugin> %s", err)
		return nil, nil, nil, deferFunc, err
	}

	log.Debug("fileUploadAndGetPlugin> writing file %s", tmpfn)
	io.Copy(f, file)
	f.Close()

	content, err := os.Open(tmpfn)
	if err != nil {
		log.Critical("fileUploadAndGetPlugin> %s", err)
		return nil, nil, nil, deferFunc, err
	}

	ap, params, err := actionplugin.Get(filename, tmpfn)
	if err != nil {
		log.Warning("fileUploadAndGetPlugin> unable to get plugin info: %s", err)
		return nil, nil, nil, deferFunc, sdk.NewError(sdk.ErrPluginInvalid, err)
	}

	return ap, params, content, deferFunc, nil
}

func addPluginHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {
	//Upload file and get plugin information
	ap, params, file, deferFunc, err := fileUploadAndGetPlugin(w, r)
	if deferFunc != nil {
		defer deferFunc()
	}
	if err != nil {
		log.Warning("addPluginHandler>%T %s", err, err)
		WriteError(w, r, err)
		return
	}
	defer file.Close()

	// Check that action does not already exists
	conflict, err := action.Exists(db, ap.Name)
	if err != nil {
		log.Warning("updatePluginHandler>%T %s", err, err)
		WriteError(w, r, err)
		return
	}
	if conflict {
		WriteError(w, r, sdk.ErrConflict)
		return
	}

	//Upload it to objectstore
	objectPath, err := objectstore.StorePlugin(*ap, file)
	if err != nil {
		log.Warning("addPluginHandler> Error while uploading to object store %s: %s\n", ap.Name, err)
		WriteError(w, r, err)
		return
	}
	ap.ObjectPath = objectPath

	tx, err := db.Begin()
	if err != nil {
		log.Warning("addPluginHandler> Cannot start transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	//Insert in database
	a, err := actionplugin.Insert(tx, ap, params)
	if err != nil {
		log.Warning("addPluginHandler> Error while inserting action %s in database: %s\n", ap.Name, err)
		objectstore.DeletePlugin(*ap)
		WriteError(w, r, err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Warning("addPluginHandler> Cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, a, http.StatusCreated)
	return
}

func updatePluginHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {
	//Upload file and get plugin information
	ap, params, file, deferFunc, err := fileUploadAndGetPlugin(w, r)
	if deferFunc != nil {
		defer deferFunc()
	}

	if err != nil {
		log.Warning("updatePluginHandler>%T %s", err, err)
		WriteError(w, r, err)
		return
	}

	// Check that action does not already exists
	exists, err := action.Exists(db, ap.Name)
	if err != nil {
		log.Warning("updatePluginHandler>%T %s", err, err)
		WriteError(w, r, err)
		return
	}
	if !exists {
		WriteError(w, r, sdk.ErrNoAction)
		return
	}

	//Store previous file from objectstore
	buf, err := objectstore.FetchPlugin(*ap)
	if err != nil {
		log.Warning("updatePluginHandler>Unable to fetch plugin: %s", err)
		WriteError(w, r, sdk.NewError(sdk.ErrPluginInvalid, err))
		return
	}
	defer buf.Close()

	//Read it
	btes, err := ioutil.ReadAll(buf)
	if err != nil {
		log.Warning("updatePluginHandler>%T %s", err, err)
		WriteError(w, r, err)
		return
	}
	//Get a dir
	tmpDir, err := ioutil.TempDir("", "old-plugin")
	if err != nil {
		log.Warning("updatePluginHandler> error with tempdir %T %s", err, err)
		WriteError(w, r, err)
		return
	}
	os.MkdirAll(tmpDir, os.FileMode(0700))

	//Get a temp file
	tmpFile := path.Join(tmpDir, ap.Name)

	//Write it
	log.Debug("updatePluginHandler>store oldfile %s in case of error", tmpFile)
	if err := ioutil.WriteFile(tmpFile, btes, os.FileMode(0600)); err != nil {
		log.Warning("updatePluginHandler>Error writing file %s %T %s", tmpFile, err, err)
		WriteError(w, r, err)
		return
	}
	defer func() {
		log.Debug("updatePluginHandler> deleting file %s", tmpFile)
		os.RemoveAll(tmpFile)
	}()

	//Delete previous file from objectstore
	objectstore.DeletePlugin(*ap)
	if err != nil {
		log.Warning("updatePluginHandler>Error deleting file %T %s", err, err)
		WriteError(w, r, err)
		return
	}

	//Upload it to objectstore
	objectPath, err := objectstore.StorePlugin(*ap, file)
	if err != nil {
		log.Warning("updatePluginHandler> Error while uploading to object store %s: %s\n", ap.Name, err)
		WriteError(w, r, err)
		return
	}
	ap.ObjectPath = objectPath

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updatePluginHandler> Cannot start transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	//Update in database
	a, errDB := actionplugin.Update(tx, ap, params, c.User.ID)
	if errDB != nil {
		log.Warning("updatePluginHandler> Error while updating action %s in database: %s\n", ap.Name, err)

		//Restore previous file
		oldFile, err := os.Open(tmpFile)
		if err != nil {
			log.Warning("updatePluginHandler>Error opening file %s %T %s", tmpFile, err, err)
			WriteError(w, r, err)
			return
		}
		//re-store the old plugin file
		if _, err := objectstore.StorePlugin(*ap, oldFile); err != nil {
			log.Warning("updatePluginHandler> Error while uploading to object store %s: %s\n", ap.Name, err)
			WriteError(w, r, err)
			return
		}

		log.Warning("updatePluginHandler>%T %s", errDB, errDB)
		WriteError(w, r, errDB)
		return
	}
	if err := tx.Commit(); err != nil {
		log.Warning("updatePluginHandler> Cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, a, http.StatusOK)
	return
}

func deletePluginHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {
	vars := mux.Vars(r)
	name := vars["name"]

	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	//Delete in database
	if err := actionplugin.Delete(db, name, c.User.ID); err != nil {
		log.Warning("deletePluginHandler> Error while deleting action %s in database: %s\n", name, err)
		WriteError(w, r, err)
		return
	}

	//Delete from objectstore
	if err := objectstore.DeletePlugin(sdk.ActionPlugin{Name: name}); err != nil {
		log.Warning("deletePluginHandler> Error while deleting action %s in objectstore: %s\n", name, err)
		WriteError(w, r, err)
		return
	}
}

func downloadPluginHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {
	vars := mux.Vars(r)
	name := vars["name"]

	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	f, err := objectstore.FetchPlugin(sdk.ActionPlugin{Name: name})
	if err != nil {
		log.Warning("downloadPluginHandler> Error while fetching plugin: %s\n", name, err)
		WriteError(w, r, err)
		return
	}

	w.Header().Add("Content-Type", "application/octet-stream")
	w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", name))

	if err := objectstore.StreamFile(w, f); err != nil {
		log.Warning("downloadPluginHandler> Error while streaming plugin %s: %s\n", name, err)
		WriteError(w, r, err)
		return
	}
}
