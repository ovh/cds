package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sanity"
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

func getTemplatesHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	tmpls, err := templateextension.All(db)
	if err != nil {
		log.Warning("getTemplatesHandler>%T %s", err, err)
		return err

	}
	return WriteJSON(w, r, tmpls, http.StatusOK)
}

func addTemplateHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	//Upload file and get as a template object
	templ, params, file, deferFunc, err := fileUploadAndGetTemplate(w, r)
	if deferFunc != nil {
		defer deferFunc()
	}
	if err != nil {
		log.Warning("addTemplateHandler>%T %s", err, err)
		return err

	}
	defer file.Close()

	log.Info("Uploaded template %s", templ.Identifier)
	log.Info("Template params %v", params)

	//Check actions
	for _, a := range templ.Actions {
		log.Debug("Checking action %s", a)
		pa, err := action.LoadPublicAction(db, a)
		if err != nil {
			return err

		}
		if pa == nil {
			return sdk.ErrNoAction
		}
	}

	//Upload to objectstore
	objectpath, err := objectstore.StoreTemplateExtension(*templ, file)
	if err != nil {
		log.Warning("addTemplateHandler>%T %s", err, err)
		return err
	}

	//Set the objectpath in the template
	templ.ObjectPath = objectpath

	//Insert in database
	if err := templateextension.Insert(db, templ); err != nil {
		log.Warning("addTemplateHandler>%T %s", err, err)
		return err

	}

	return WriteJSON(w, r, templ, http.StatusOK)
}

func updateTemplateHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get id from URL
	vars := mux.Vars(r)
	sid := vars["id"]

	if sid == "" {
		return sdk.ErrWrongRequest

	}

	//Get int from string
	id, err := strconv.Atoi(sid)
	if err != nil {
		return sdk.ErrWrongRequest

	}

	//Find it
	templ, err := templateextension.LoadByID(db, int64(id))
	if err != nil {
		log.Warning("updateTemplateHandler>Unable to load template: %s", err)
		return sdk.NewError(sdk.ErrNotFound, err)
	}

	//Store previous file from objectstore
	tmpbuf, err := objectstore.FetchTemplateExtension(*templ)
	if err != nil {
		log.Warning("updateTemplateHandler>Unable to fetch template: %s", err)
		return sdk.NewError(sdk.ErrPluginInvalid, err)
	}
	defer tmpbuf.Close()

	//Read it
	btes, err := ioutil.ReadAll(tmpbuf)
	if err != nil {
		log.Warning("updateTemplateHandler>%T %s", err, err)
		return err

	}

	//Delete from storage
	if err := objectstore.DeleteTemplateExtension(*templ); err != nil {
		return err

	}

	//Upload file and get as a template object
	templ2, params, file, deferFunc, err := fileUploadAndGetTemplate(w, r)
	if deferFunc != nil {
		defer deferFunc()
	}
	if err != nil {
		log.Warning("addTemplateHandler>%T %s", err, err)
		return err

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
			return err
		}
		if pa == nil {
			return sdk.ErrNoAction
		}
	}

	//Upload to objectstore
	objectpath, err := objectstore.StoreTemplateExtension(*templ2, file)
	if err != nil {
		log.Warning("updateTemplateHandler>%T %s", err, err)
		return err
	}

	templ2.ObjectPath = objectpath

	if err := templateextension.Update(db, templ2); err != nil {
		//re-store the old file in case of error
		if _, err := objectstore.StoreTemplateExtension(*templ2, ioutil.NopCloser(bytes.NewBuffer(btes))); err != nil {
			log.Warning("updateTemplateHandler> Error while uploading to object store %s: %s\n", templ2.Name, err)
			return err

		}

		log.Warning("updateTemplateHandler>%T %s", err, err)
		return err

	}

	return WriteJSON(w, r, templ2, http.StatusOK)
}

func deleteTemplateHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get id from URL
	vars := mux.Vars(r)
	sid := vars["id"]

	if sid == "" {
		return sdk.ErrWrongRequest

	}

	//Get int from string
	id, err := strconv.Atoi(sid)
	if err != nil {
		return sdk.ErrWrongRequest

	}

	//Load it
	templ, err := templateextension.LoadByID(db, int64(id))
	if err != nil {
		return err

	}

	//Delete it
	if err := templateextension.Delete(db, templ); err != nil {
		return err

	}

	//Delete from storage
	if err := objectstore.DeleteTemplateExtension(*templ); err != nil {
		return err

	}

	//OK
	return nil
}

func getBuildTemplatesHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	tpl, err := templateextension.LoadByType(db, "BUILD")
	if err != nil {
		return err

	}
	return WriteJSON(w, r, tpl, http.StatusOK)
}

func getDeployTemplatesHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	tpl, err := templateextension.LoadByType(db, "DEPLOY")
	if err != nil {
		return err

	}
	return WriteJSON(w, r, tpl, http.StatusOK)
}

func applyTemplateHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]

	// Load the project
	proj, err := project.Load(db, projectKey, c.User)
	if err != nil {
		log.Warning("applyTemplatesHandler> Cannot load project %s: %s\n", projectKey, err)
		return err

	}

	proj.Environments, err = environment.LoadEnvironments(db, projectKey, true, c.User)
	if err != nil {
		log.Warning("applyTemplatesHandler: Cannot load environments: %s\n", projectKey, err)
		return err
	}

	// Load groups on the project
	if err := group.LoadGroupByProject(db, proj); err != nil {
		log.Warning("applyTemplatesHandler> Cannot load project groups %s: %s\n", projectKey, err)
		return err

	}

	// Get data in body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("applyTemplatesHandler> Cannot read body %s: %s\n", string(data), err)
		return sdk.ErrWrongRequest

	}

	// Parse body to sdk.ApplyTemplatesOptions
	var opts sdk.ApplyTemplatesOptions
	if err := json.Unmarshal(data, &opts); err != nil {
		log.Warning("applyTemplatesHandler> Cannot parse body %s: %s\n", string(data), err)
		return sdk.ErrWrongRequest

	}

	// Create a session for current user
	sessionKey, err := auth.NewSession(router.authDriver, c.User)
	if err != nil {
		log.Critical("applyTemplateHandler> Error while creating new session: %s\n", err)
		return err

	}

	// Apply the template
	log.Debug("applyTemplateHandler> applyTemplate")
	msg, err := template.ApplyTemplate(db, proj, opts, c.User, sessionKey)
	if err != nil {
		return err

	}

	al := r.Header.Get("Accept-Language")
	msgList := []string{}

	for _, m := range msg {
		s := m.String(al)
		msgList = append(msgList, s)
	}

	log.Debug("applyTemplatesHandler> Check warnings on project")
	if err := sanity.CheckProjectPipelines(db, proj); err != nil {
		log.Warning("applyTemplatesHandler> Cannot check warnings: %s\n", err)
		return err

	}

	apps, errApp := application.LoadApplications(db, proj.Key, false, true, c.User)
	if errApp != nil {
		log.Warning("applyTemplatesHandler> Cannot load applications: %s\n", err)
		return err

	}
	proj.Applications = apps

	for _, a := range apps {
		if err := sanity.CheckApplication(db, proj, &a); err != nil {
			log.Warning("applyTemplatesHandler> Cannot check application sanity: %s\n", err)
			return err
		}
	}

	return WriteJSON(w, r, proj, http.StatusOK)
}

func applyTemplateOnApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]

	// Load the project
	proj, err := project.Load(db, projectKey, c.User)
	if err != nil {
		log.Warning("applyTemplateOnApplicationHandler> Cannot load project %s: %s\n", projectKey, err)
		return err

	}

	// Load the application
	app, err := application.LoadApplicationByName(db, projectKey, appName)
	if err != nil {
		log.Warning("applyTemplateOnApplicationHandler> Cannot load application %s: %s\n", appName, err)
		return err

	}

	// Get data in body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("applyTemplateOnApplicationHandler> Unable to read body : %s\n", err)
		return sdk.ErrWrongRequest

	}

	// Parse body to sdk.ApplyTemplatesOptions
	var opts sdk.ApplyTemplatesOptions
	if err := json.Unmarshal(data, &opts); err != nil {
		return sdk.ErrWrongRequest

	}

	//Create a session for current user
	sessionKey, err := auth.NewSession(router.authDriver, c.User)
	if err != nil {
		log.Critical("Instance> Error while creating new session: %s\n", err)
		return err

	}

	//Apply the template
	msg, err := template.ApplyTemplateOnApplication(db, proj, app, opts, c.User, sessionKey)
	if err != nil {
		return err

	}

	al := r.Header.Get("Accept-Language")
	msgList := []string{}

	for _, m := range msg {
		s := m.String(al)
		msgList = append(msgList, s)
	}

	log.Debug("applyTemplatesHandler> Check warnings on project")
	if err := sanity.CheckProjectPipelines(db, proj); err != nil {
		log.Warning("applyTemplatesHandler> Cannot check warnings: %s\n", err)
		return err

	}

	return WriteJSON(w, r, msgList, http.StatusOK)
}
