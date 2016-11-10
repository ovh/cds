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
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/msg"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
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

	templ.ObjectPath = objectpath

	//Insert in database
	dbmap := database.DBMap(db)
	dbtempl := database.TemplateExtension(*templ)
	if err := dbmap.Insert(&dbtempl); err != nil {
		log.Warning("addTemplateHandler>%T %s", err, err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, dbtempl, http.StatusOK)
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
	templ := database.TemplateExtension{}
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
	tmpbuf, err := objectstore.FetchTemplateExtension(sdk.TemplateExtension(templ))
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
	if err := objectstore.DeleteTemplateExtension(sdk.TemplateExtension(templ)); err != nil {
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

	templ.ObjectPath = objectpath

	//Update in database
	dbtempl2 := database.TemplateExtension(*templ2)
	if _, err := dbmap.Update(&dbtempl2); err != nil {
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
	templ := database.TemplateExtension{}
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
	if err := objectstore.DeleteTemplateExtension(sdk.TemplateExtension(templ)); err != nil {
		WriteError(w, r, err)
		return
	}

	//OK
	w.WriteHeader(http.StatusOK)
}

func getBuildTemplatesHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	tpl, err := getTypedTemplatesHandler(db, "BUILD")
	if err != nil {
		WriteError(w, r, err)
		return
	}
	WriteJSON(w, r, tpl, http.StatusOK)
}

func getDeployTemplatesHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	tpl, err := getTypedTemplatesHandler(db, "DEPLOY")
	if err != nil {
		WriteError(w, r, err)
		return
	}
	WriteJSON(w, r, tpl, http.StatusOK)
}

func getTypedTemplatesHandler(db *sql.DB, t string) ([]sdk.Template, error) {
	var tpl []sdk.Template
	tpl = []sdk.Template{
		sdk.Template{
			ID:          template.UglyID,
			Name:        "Void",
			Description: "Empty template",
		},
	}

	tplFromDB := []sdk.TemplateExtension{}
	dbmap := database.DBMap(db)
	if _, err := dbmap.Select(&tplFromDB, "select * from template where type = $1 order by name", t); err != nil {
		log.Warning("getTypedTemplatesHandler> Error : %s", err)
		return nil, err
	}

	for _, t := range tplFromDB {
		params := []sdk.TemplateParam{}
		str, err := dbmap.SelectStr("select params from template_params where template_id = $1", t.ID)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(str), &params); err != nil {
			return nil, err
		}

		tpl = append(tpl, sdk.Template{
			ID:          t.ID,
			Name:        t.Name,
			Description: t.Description,
			Params:      params,
		})

	}

	return tpl, nil
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

	// Get template from DB
	dbmap := database.DBMap(db)
	tmpl := database.TemplateExtension{}
	if err := dbmap.SelectOne(&tmpl, "select * from template where name = $1", opts.TemplateName); err != nil {
		if err == sql.ErrNoRows {
			WriteError(w, r, sdk.ErrUnknownTemplate)
			return
		}
		WriteError(w, r, err)
		return
	}

	// Load the template binary
	sdktmpl := sdk.TemplateExtension(tmpl)
	templ, deferFunc, err := templateextension.Instance(router.authDriver, &sdktmpl, c.User)
	if deferFunc != nil {
		defer deferFunc()
	}
	if err != nil {
		log.Warning("applyTemplatesHandler> error getting template Extension instance : %s", err)
		WriteError(w, r, err)
	}

	// Apply the template
	app, err := templateextension.Apply(templ, proj, opts.TemplateParams, opts.ApplicationName)
	if err != nil {
		log.Warning("applyTemplatesHandler> error applying template : %s", err)
		WriteError(w, r, err)
		return
	}

	//Check reposmanager
	if opts.RepositoriesManagerName != "" {
		app.RepositoriesManager, err = repositoriesmanager.LoadByName(db, opts.RepositoriesManagerName)
		if err != nil {
			log.Warning("applyTemplatesHandler> error getting repositories manager %s : %s", opts.RepositoriesManagerName, err)
			WriteError(w, r, err)
			return
		}

		app.RepositoryFullname = opts.ApplicationRepositoryFullname
	}

	//Start a new transaction
	tx, err := db.Begin()
	if err != nil {
		log.Warning("applyTemplatesHandler> error beginning transaction : %s", err)
		WriteError(w, r, err)
	}

	defer tx.Rollback()

	// Import the application
	done := make(chan bool)
	msgChan := make(chan msg.Message)
	msgList := []string{}
	al := r.Header.Get("Accept-Language")
	go func(array *[]string) {
		for {
			m, more := <-msgChan
			if !more {
				done <- true
				return
			}
			s := m.String(al)
			*array = append(*array, s)
			log.Debug("applyTemplatesHandler> message : %s", s)
		}
	}(&msgList)

	if err := application.Import(tx, proj, app, app.RepositoriesManager, msgChan); err != nil {
		log.Warning("applyTemplatesHandler> error applying template : %s", err)
		WriteError(w, r, err)
		close(msgChan)
		return
	}

	close(msgChan)
	<-done

	log.Debug("applyTemplatesHandler> Commit the transaction")
	if err := tx.Commit(); err != nil {
		log.Warning("applyTemplatesHandler> error commiting transaction : %s", err)
		WriteError(w, r, err)
		return
	}

	deferFunc()
	log.Debug("applyTemplatesHandler> Done")
}
