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

func addPluginHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	//Upload file and get plugin information
	ap, params, file, deferFunc, err := fileUploadAndGetPlugin(w, r)
	if deferFunc != nil {
		defer deferFunc()
	}
	if err != nil {
		log.Warning("addPluginHandler>%T %s", err, err)
		return err
	}
	defer file.Close()

	// Check that action does not already exists
	conflict, err := action.Exists(db, ap.Name)
	if err != nil {
		log.Warning("updatePluginHandler>%T %s", err, err)
		return err
	}
	if conflict {
		return sdk.ErrConflict
	}

	//Upload it to objectstore
	objectPath, err := objectstore.StorePlugin(*ap, file)
	if err != nil {
		log.Warning("addPluginHandler> Error while uploading to object store %s: %s\n", ap.Name, err)
		return err
	}
	ap.ObjectPath = objectPath

	tx, err := db.Begin()
	if err != nil {
		log.Warning("addPluginHandler> Cannot start transaction: %s\n", err)
		return err
	}
	defer tx.Rollback()

	//Insert in database
	a, err := actionplugin.Insert(tx, ap, params)
	if err != nil {
		log.Warning("addPluginHandler> Error while inserting action %s in database: %s\n", ap.Name, err)
		objectstore.DeletePlugin(*ap)
		return err

	}

	if err := tx.Commit(); err != nil {
		log.Warning("addPluginHandler> Cannot commit transaction: %s\n", err)
		return err

	}

	return WriteJSON(w, r, a, http.StatusCreated)
}

func updatePluginHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	//Upload file and get plugin information
	ap, params, file, deferFunc, errUpload := fileUploadAndGetPlugin(w, r)
	if deferFunc != nil {
		defer deferFunc()
	}
	if errUpload != nil {
		return sdk.WrapError(errUpload, "updatePluginHandler> fileUploadAndGetPlugin error")
	}

	// Check that action does not already exists
	exists, errExists := action.Exists(db, ap.Name)
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

	tx, errBegin := db.Begin()
	if errBegin != nil {
		return sdk.WrapError(errBegin, "updatePluginHandler> Cannot start transaction")
	}
	defer tx.Rollback()

	//Update in database
	a, errDB := actionplugin.Update(tx, ap, params, c.User.ID)
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
		log.Warning("updatePluginHandler> Cannot commit transaction: %s\n", err)
		return err
	}

	return WriteJSON(w, r, a, http.StatusOK)
}

func deletePluginHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	name := vars["name"]

	if name == "" {
		return sdk.ErrWrongRequest

	}

	//Delete in database
	if err := actionplugin.Delete(db, name, c.User.ID); err != nil {
		log.Warning("deletePluginHandler> Error while deleting action %s in database: %s\n", name, err)
		return err

	}

	//Delete from objectstore
	if err := objectstore.DeletePlugin(sdk.ActionPlugin{Name: name}); err != nil {
		log.Warning("deletePluginHandler> Error while deleting action %s in objectstore: %s\n", name, err)
		return err

	}
	return nil
}

func downloadPluginHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	name := vars["name"]

	if name == "" {
		return sdk.ErrWrongRequest

	}

	f, err := objectstore.FetchPlugin(sdk.ActionPlugin{Name: name})
	if err != nil {
		log.Warning("downloadPluginHandler> Error while fetching plugin: %s\n", name, err)
		return err
	}

	w.Header().Add("Content-Type", "application/octet-stream")
	w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", name))

	if err := objectstore.StreamFile(w, f); err != nil {
		log.Warning("downloadPluginHandler> Error while streaming plugin %s: %s\n", name, err)
		return err
	}

	return nil
}
