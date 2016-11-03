package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"bytes"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/log"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/template"
	"github.com/ovh/cds/engine/api/templateextension"
	"github.com/ovh/cds/sdk"
)

func fileUploadAndGetTemplate(w http.ResponseWriter, r *http.Request) (*sdk.TemplateExtention, []sdk.TemplateParam, io.ReadCloser, func(), error) {
	r.ParseMultipartForm(64 << 20)
	file, handler, err := r.FormFile("UploadFile")
	if err != nil {
		log.Warning("fileUploadAndGetTemplate> %s", err)
		log.Debug("fileUploadAndGetTemplate> %v", r.Header)
		return nil, nil, nil, nil, err
	}

	filename := handler.Filename
	t := strings.Split(handler.Filename, "/")
	if len(t) > 1 {
		filename = t[len(t)-1]
	}

	log.Debug("fileUploadAndGetTemplate> file upload detected : %s", filename)
	defer file.Close()

	tmp, err := ioutil.TempDir("", "cds-template")
	if err != nil {
		log.Critical("fileUploadAndGetTemplate> %s", err)
		return nil, nil, nil, nil, err
	}
	deferFunc := func() {
		log.Debug("fileUploadAndGetTemplate> deleting file %s", tmp)
		os.RemoveAll(tmp)
	}

	log.Debug("fileUploadAndGetTemplate> creating temporary directory")
	tmpfn := filepath.Join(tmp, filename)
	f, err := os.OpenFile(tmpfn, os.O_WRONLY|os.O_CREATE, 0700)
	if err != nil {
		log.Critical("fileUploadAndGetTemplate> %s", err)
		return nil, nil, nil, deferFunc, err
	}

	log.Debug("fileUploadAndGetTemplate> writing file %s", tmpfn)
	io.Copy(f, file)
	f.Close()

	content, err := os.Open(tmpfn)
	if err != nil {
		log.Critical("fileUploadAndGetTemplate> %s", err)
		return nil, nil, nil, deferFunc, err
	}

	ap, params, err := templateextension.Get(filename, tmpfn)
	if err != nil {
		log.Warning("fileUploadAndGetTemplate> unable to get template info: %s", err)
		return nil, nil, nil, deferFunc, sdk.NewError(sdk.ErrPluginInvalid, err)
	}

	return ap, params, content, deferFunc, nil
}

func getTemplatesHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	dbmap := database.DBMap(db)
	tmpls := []sdk.TemplateExtention{}
	_, err := dbmap.Select(&tmpls, "select * from template order by id")
	if err != nil {
		log.Warning("getTemplatesHandler> Error: %s", err)
		WriteError(w, r, err)
		return
	}
	WriteJSON(w, r, tmpls, http.StatusOK)
}

func addTemplateHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	//Upload file and get as a template object
	templ, params, file, deferFunc, err := fileUploadAndGetTemplate(w, r)
	if deferFunc != nil {
		defer deferFunc()
	}
	if err != nil {
		log.Warning("addTemplateHandler>%T %s", err, err)
		WriteError(w, r, err)
		return
	}
	defer file.Close()

	log.Debug("Uploaded template %s", templ.Identifier)
	log.Debug("Template params %v", params)

	//Upload to objectstore
	objectpath, err := objectstore.StoreTemplateExtension(*templ, file)
	if err != nil {
		log.Warning("addTemplateHandler>%T %s", err, err)
		WriteError(w, r, err)
		return
	}

	templ.ObjectPath = objectpath

	//Insert in database
	dbmap := database.DBMap(db)
	if err := dbmap.Insert(templ); err != nil {
		log.Warning("addTemplateHandler>%T %s", err, err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, templ, http.StatusOK)
}

func updateTemplateHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get id from URL
	vars := mux.Vars(r)
	sid := vars["id"]

	if sid == "" {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	//Get int from string
	id, err := strconv.Atoi(sid)
	if err != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	//Get the database map
	dbmap := database.DBMap(db)

	//Find it
	templ := sdk.TemplateExtention{}
	if err := dbmap.SelectOne(&templ, "select * from template where id = $1", id); err != nil {
		if err == sql.ErrNoRows {
			WriteError(w, r, sdk.ErrNotFound)
			return
		}
		log.Warning("updateTemplateHandler>%T %s", err, err)
		WriteError(w, r, err)
		return
	}

	//Store previous file from objectstore
	tmpbuf, err := objectstore.FetchTemplateExtension(templ)
	if err != nil {
		log.Warning("updateTemplateHandler>Unable to fetch plugin: %s", err)
		WriteError(w, r, sdk.NewError(sdk.ErrPluginInvalid, err))
		return
	}
	defer tmpbuf.Close()

	//Read it
	btes, err := ioutil.ReadAll(tmpbuf)
	if err != nil {
		log.Warning("updateTemplateHandler>%T %s", err, err)
		WriteError(w, r, err)
		return
	}

	//Delete from storage
	if err := objectstore.DeleteTemplateExtension(templ); err != nil {
		WriteError(w, r, err)
		return
	}

	//Upload file and get as a template object
	templ2, params, file, deferFunc, err := fileUploadAndGetTemplate(w, r)
	if deferFunc != nil {
		defer deferFunc()
	}
	if err != nil {
		log.Warning("addTemplateHandler>%T %s", err, err)
		WriteError(w, r, err)
		return
	}
	defer file.Close()
	templ2.ID = templ.ID

	log.Debug("Uploaded template %s", templ2.Identifier)
	log.Debug("Template params %v", params)

	//Upload to objectstore
	objectpath, err := objectstore.StoreTemplateExtension(*templ2, file)
	if err != nil {
		log.Warning("updateTemplateHandler>%T %s", err, err)
		WriteError(w, r, err)
		return
	}

	templ.ObjectPath = objectpath

	//Update in database
	if _, err := dbmap.Update(templ2); err != nil {
		//re-store the old file in case of error
		if _, err := objectstore.StoreTemplateExtension(*templ2, ioutil.NopCloser(bytes.NewBuffer(btes))); err != nil {
			log.Warning("updateTemplateHandler> Error while uploading to object store %s: %s\n", templ2.Name, err)
			WriteError(w, r, err)
			return
		}

		log.Warning("updateTemplateHandler>%T %s", err, err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, templ2, http.StatusOK)

}

func deleteTemplateHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get id from URL
	vars := mux.Vars(r)
	sid := vars["id"]

	if sid == "" {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	//Get int from string
	id, err := strconv.Atoi(sid)
	if err != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	//Get the database map
	dbmap := database.DBMap(db)

	//Find it
	templ := sdk.TemplateExtention{}
	if err := dbmap.SelectOne(&templ, "select * from template where id = $1", id); err != nil {
		if err == sql.ErrNoRows {
			WriteError(w, r, sdk.ErrNotFound)
			return
		}
		log.Warning("deleteTemplateHandler>%T %s", err, err)
		WriteError(w, r, err)
		return
	}

	//Delete it
	n, err := dbmap.Delete(&templ)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	//Check if it has been deleted
	if n == 0 {
		WriteError(w, r, sdk.ErrNotFound)
		return
	}

	//Delete from storage
	if err := objectstore.DeleteTemplateExtension(templ); err != nil {
		WriteError(w, r, err)
		return
	}

	//OK
	w.WriteHeader(http.StatusOK)
}

func getBuildTemplatesHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	tpl := []sdk.Template{
		sdk.Template{
			ID:          template.UglyID,
			Name:        "Void",
			Description: "Empty template",
		},
	}

	tplFromDB := []sdk.TemplateExtention{}
	dbmap := database.DBMap(db)
	if _, err := dbmap.Select(&tplFromDB, "select * from template where type = 'BUILD' order by name"); err != nil {
		log.Warning("getBuildTemplates> Error : %s", err)
		WriteError(w, r, err)
		return
	}

	for _, t := range tplFromDB {
		params := []sdk.TemplateParam{}
		str, err := dbmap.SelectStr("select params from template_params where template_id = $1", t.ID)
		log.Debug(str)
		if err != nil {
			WriteError(w, r, err)
			return
		}
		if err := json.Unmarshal([]byte(str), &params); err != nil {
			WriteError(w, r, err)
			return
		}

		tpl = append(tpl, sdk.Template{
			ID:          t.ID,
			Name:        t.Name,
			Description: t.Description,
			Params:      params,
		})

	}

	WriteJSON(w, r, tpl, http.StatusOK)
}

func getDeployTemplates(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	var tpl []sdk.Template
	WriteJSON(w, r, tpl, http.StatusOK)
}
