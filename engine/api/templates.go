package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/template"
	"github.com/ovh/cds/engine/api/templateextension"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func fileUploadAndGetTemplate(w http.ResponseWriter, r *http.Request) (*sdk.TemplateExtension, []sdk.TemplateParam, io.ReadCloser, func(), error) {
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
	tmpls := []database.TemplateExtension{}
	_, err := dbmap.Select(&tmpls, "select * from template order by id")
	if err != nil {
		log.Warning("getTemplatesHandler> Error: %s", err)
		WriteError(w, r, err)
		return
	}
	//Load actions and params
	for i := range tmpls {
		_, err := dbmap.Select(&tmpls[i].Actions, "select action.name from action, template_action where template_action.action_id = action.id and template_id = $1", tmpls[i].ID)
		if err != nil {
			log.Warning("getTemplatesHandler> Error: %s", err)
			WriteError(w, r, err)
			return
		}
		params := []sdk.TemplateParam{}
		str, err := dbmap.SelectStr("select params from template_params where template_id = $1", tmpls[i].ID)
		if err != nil {
			WriteError(w, r, err)
			return
		}
		if err := json.Unmarshal([]byte(str), &params); err != nil {
			WriteError(w, r, err)
			return
		}
		tmpls[i].Params = params
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

	log.Info("Uploaded template %s", templ.Identifier)
	log.Info("Template params %v", params)

	//Check actions
	for _, a := range templ.Actions {
		log.Debug("Checking action %s", a)
		pa, err := action.LoadPublicAction(db, a)
		if err != nil {
			WriteError(w, r, err)
			return
		}
		if pa == nil {
			WriteError(w, r, sdk.ErrNoAction)
			return
		}
	}

	//Upload to objectstore
	objectpath, err := objectstore.StoreTemplateExtension(*templ, file)
	if err != nil {
		log.Warning("addTemplateHandler>%T %s", err, err)
		WriteError(w, r, err)
		return
	}

	//Set the objectpath in the template
	templ.ObjectPath = objectpath

	//Insert in database
	if err := templateextension.Insert(db, templ); err != nil {
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

	//Find it
	templ, err := templateextension.LoadByID(db, int64(id))

	//Store previous file from objectstore
	tmpbuf, err := objectstore.FetchTemplateExtension(*templ)
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
	if err := objectstore.DeleteTemplateExtension(*templ); err != nil {
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

	//Check actions
	for _, a := range templ2.Actions {
		log.Debug("updateTemplateHandler> Checking action %s", a)
		pa, err := action.LoadPublicAction(db, a)
		if err != nil {
			WriteError(w, r, err)
			return
		}
		if pa == nil {
			WriteError(w, r, sdk.ErrNoAction)
			return
		}
	}

	//Upload to objectstore
	objectpath, err := objectstore.StoreTemplateExtension(*templ2, file)
	if err != nil {
		log.Warning("updateTemplateHandler>%T %s", err, err)
		WriteError(w, r, err)
		return
	}

	templ2.ObjectPath = objectpath

	if err := templateextension.Update(db, templ2); err != nil {
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

	//Load it
	templ, err := templateextension.LoadByID(db, int64(id))
	if err != nil {
		WriteError(w, r, err)
		return
	}

	//Delete it
	if err := templateextension.Delete(db, templ); err != nil {
		WriteError(w, r, err)
		return
	}

	//Delete from storage
	if err := objectstore.DeleteTemplateExtension(*templ); err != nil {
		WriteError(w, r, err)
		return
	}

	//OK
	w.WriteHeader(http.StatusOK)
}

func getBuildTemplatesHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	tpl, err := templateextension.LoadByType(db, "BUILD")
	if err != nil {
		WriteError(w, r, err)
		return
	}
	WriteJSON(w, r, tpl, http.StatusOK)
}

func getDeployTemplatesHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	tpl, err := templateextension.LoadByType(db, "DEPLOY")
	if err != nil {
		WriteError(w, r, err)
		return
	}
	WriteJSON(w, r, tpl, http.StatusOK)
}

func applyTemplatesHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]

	//Load the project
	proj, err := project.LoadProject(db, projectKey, c.User)
	if err != nil {
		log.Warning("applyTemplatesHandler> Cannot load project %s: %s\n", projectKey, err)
		WriteError(w, r, err)
		return
	}

	// Get data in body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	// Parse body to sdk.ApplyTemplatesOptions
	var opts sdk.ApplyTemplatesOptions
	if err := json.Unmarshal(data, &opts); err != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	//Create a session for current user
	sessionKey, err := auth.NewSession(router.authDriver, c.User)
	if err != nil {
		log.Critical("Instance> Error while creating new session: %s\n", err)
		WriteError(w, r, err)
		return
	}

	//Apply the template
	msg, err := template.ApplyTemplate(db, proj, opts, c.User, sessionKey)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	al := r.Header.Get("Accept-Language")
	msgList := []string{}

	for _, m := range msg {
		s := m.String(al)
		msgList = append(msgList, s)
	}

	WriteJSON(w, r, msgList, http.StatusOK)
}
