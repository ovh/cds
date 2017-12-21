package api

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/template"
	"github.com/ovh/cds/engine/api/templateextension"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
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
		log.Error("fileUploadAndGetTemplate> %s", err)
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
		log.Error("fileUploadAndGetTemplate> %s", err)
		return nil, nil, nil, deferFunc, err
	}

	log.Debug("fileUploadAndGetTemplate> writing file %s", tmpfn)
	io.Copy(f, file)
	f.Close()

	content, err := os.Open(tmpfn)
	if err != nil {
		log.Error("fileUploadAndGetTemplate> %s", err)
		return nil, nil, nil, deferFunc, err
	}

	ap, params, err := templateextension.Get(filename, tmpfn)
	if err != nil {
		log.Warning("fileUploadAndGetTemplate> unable to get template info: %s", err)
		return nil, nil, nil, deferFunc, sdk.NewError(sdk.ErrPluginInvalid, err)
	}

	return ap, params, content, deferFunc, nil
}

func (api *API) getTemplatesHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		tmpls, err := templateextension.All(api.mustDB())
		if err != nil {
			return sdk.WrapError(err, "getTemplatesHandler>%T", err)
		}
		return WriteJSON(w, r, tmpls, http.StatusOK)
	}
}

func (api *API) addTemplateHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Upload file and get as a template object
		templ, params, file, deferFunc, errf := fileUploadAndGetTemplate(w, r)
		if deferFunc != nil {
			defer deferFunc()
		}
		if errf != nil {
			return sdk.WrapError(errf, "addTemplateHandler>%T", errf)
		}
		defer file.Close()

		log.Debug("Uploaded template %s", templ.Identifier)
		log.Debug("Template params %v", params)

		//Check actions
		for _, a := range templ.Actions {
			log.Debug("Checking action %s", a)
			pa, err := action.LoadPublicAction(api.mustDB(), a)
			if err != nil {
				return sdk.WrapError(err, "addTemplateHandler> err on loadPublicAction")
			}
			if pa == nil {
				return sdk.ErrNoAction
			}
		}

		//Upload to objectstore
		objectpath, err := objectstore.StoreTemplateExtension(*templ, file)
		if err != nil {
			return sdk.WrapError(err, "addTemplateHandler>%T", err)
		}

		//Set the objectpath in the template
		templ.ObjectPath = objectpath

		//Insert in database
		if err := templateextension.Insert(api.mustDB(), templ); err != nil {
			return sdk.WrapError(err, "addTemplateHandler>%T", err)
		}

		return WriteJSON(w, r, templ, http.StatusOK)
	}
}

func (api *API) updateTemplateHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, errr := requestVarInt(r, "id")
		if errr != nil {
			return sdk.WrapError(errr, "updateTemplateHandler> Invalid id")
		}

		//Find it
		templ, errLoad := templateextension.LoadByID(api.mustDB(), int64(id))
		if errLoad != nil {
			return sdk.WrapError(sdk.ErrNotFound, "updateTemplateHandler>Unable to load template: %s", errLoad)
		}

		//Store previous file from objectstore
		tmpbuf, errFetch := objectstore.FetchTemplateExtension(*templ)
		if errFetch != nil {
			return sdk.WrapError(sdk.ErrPluginInvalid, "updateTemplateHandler>Unable to fetch template: %s", errFetch)
		}
		defer tmpbuf.Close()

		//Read it
		btes, errRead := ioutil.ReadAll(tmpbuf)
		if errRead != nil {
			return sdk.WrapError(errRead, "updateTemplateHandler>%T", errRead)
		}

		//Delete from storage
		if err := objectstore.DeleteTemplateExtension(*templ); err != nil {
			return sdk.WrapError(err, "updateTemplateHandler> error on DeleteTemplateExtension")
		}

		//Upload file and get as a template object
		templ2, params, file, deferFunc, err := fileUploadAndGetTemplate(w, r)
		if deferFunc != nil {
			defer deferFunc()
		}
		if err != nil {
			return sdk.WrapError(err, "addTemplateHandler>%T", err)
		}

		defer file.Close()
		templ2.ID = templ.ID

		log.Debug("Uploaded template %s", templ2.Identifier)
		log.Debug("Template params %v", params)

		//Check actions
		for _, a := range templ2.Actions {
			log.Debug("updateTemplateHandler> Checking action %s", a)
			pa, errlp := action.LoadPublicAction(api.mustDB(), a)
			if errlp != nil {
				return sdk.WrapError(errlp, "updateTemplateHandler> error on loadPublicAction")
			}
			if pa == nil {
				return sdk.ErrNoAction
			}
		}

		//Upload to objectstore
		objectpath, errStore := objectstore.StoreTemplateExtension(*templ2, file)
		if errStore != nil {
			return sdk.WrapError(errStore, "updateTemplateHandler>%T", errStore)
		}

		templ2.ObjectPath = objectpath

		if errUpdate := templateextension.Update(api.mustDB(), templ2); errUpdate != nil {
			//re-store the old file in case of error
			if _, err := objectstore.StoreTemplateExtension(*templ2, ioutil.NopCloser(bytes.NewBuffer(btes))); err != nil {
				return sdk.WrapError(err, "updateTemplateHandler> Error while uploading to object store %s", templ2.Name)
			}

			return sdk.WrapError(errUpdate, "updateTemplateHandler>%T", errUpdate)
		}

		return WriteJSON(w, r, templ2, http.StatusOK)
	}
}

func (api *API) deleteTemplateHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, errr := requestVarInt(r, "id")
		if errr != nil {
			return sdk.WrapError(errr, "deleteTemplateHandler> Invalid id")
		}

		//Load it
		templ, err := templateextension.LoadByID(api.mustDB(), int64(id))
		if err != nil {
			return sdk.WrapError(err, "deleteTemplateHandler> error on LoadByID")
		}

		//Delete it
		if err := templateextension.Delete(api.mustDB(), templ); err != nil {
			return sdk.WrapError(err, "deleteTemplateHandler> error on Delete")
		}

		//Delete from storage
		if err := objectstore.DeleteTemplateExtension(*templ); err != nil {
			return sdk.WrapError(err, "deleteTemplateHandler> error on DeleteTemplate")
		}

		//OK
		return nil
	}
}

func (api *API) getBuildTemplatesHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		tpl, err := templateextension.LoadByType(api.mustDB(), "BUILD")
		if err != nil {
			return sdk.WrapError(err, "getBuildTemplatesHandler> error on loadByType")
		}
		return WriteJSON(w, r, tpl, http.StatusOK)
	}
}

func (api *API) getDeployTemplatesHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		tpl, err := templateextension.LoadByType(api.mustDB(), "DEPLOY")
		if err != nil {
			return sdk.WrapError(err, "getDeployTemplatesHandler> error on loadByType")
		}
		return WriteJSON(w, r, tpl, http.StatusOK)
	}
}

func (api *API) applyTemplateHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]

		// Load the project
		proj, errload := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx),
			project.LoadOptions.Default,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithGroups)
		if errload != nil {
			return sdk.WrapError(errload, "applyTemplatesHandler> Cannot load project %s", projectKey)
		}

		// Parse body to sdk.ApplyTemplatesOptions
		var opts sdk.ApplyTemplatesOptions
		if err := UnmarshalBody(r, &opts); err != nil {
			return err
		}

		// Create a session for current user
		sessionKey, errnew := auth.NewSession(api.Router.AuthDriver, getUser(ctx))
		if errnew != nil {
			return sdk.WrapError(errnew, "applyTemplateHandler> Error while creating new session")
		}

		// Apply the template
		log.Debug("applyTemplateHandler> applyTemplate")
		_, errapply := template.ApplyTemplate(api.mustDB(), api.Cache, proj, opts, getUser(ctx), sessionKey, api.Config.URL.API)
		if errapply != nil {
			return sdk.WrapError(errapply, "applyTemplateHandler> Error while applyTemplate")
		}

		proj, errPrj := project.Load(api.mustDB(), api.Cache, proj.Key, getUser(ctx), project.LoadOptions.Default, project.LoadOptions.WithPipelines)
		if errPrj != nil {
			return sdk.WrapError(errPrj, "applyTemplatesHandler> Cannot load project")
		}

		return WriteJSON(w, r, proj, http.StatusOK)
	}
}

func (api *API) applyTemplateOnApplicationHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		appName := vars["permApplicationName"]

		// Load the project
		proj, errLoad := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx), project.LoadOptions.Default)
		if errLoad != nil {
			return sdk.WrapError(errLoad, "applyTemplateOnApplicationHandler> Cannot load project %s", projectKey)
		}

		// Load the application
		app, errLoadByName := application.LoadByName(api.mustDB(), api.Cache, projectKey, appName, getUser(ctx), application.LoadOptions.Default)
		if errLoadByName != nil {
			return sdk.WrapError(errLoadByName, "applyTemplateOnApplicationHandler> Cannot load application %s", appName)
		}

		// Parse body to sdk.ApplyTemplatesOptions
		var opts sdk.ApplyTemplatesOptions
		if err := UnmarshalBody(r, &opts); err != nil {
			return err
		}

		//Create a session for current user
		sessionKey, err := auth.NewSession(api.Router.AuthDriver, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "applyTemplateOnApplicationHandler> Error while creating new session")
		}

		//Apply the template
		msg, err := template.ApplyTemplateOnApplication(api.mustDB(), api.Cache, proj, app, opts, getUser(ctx), sessionKey, api.Config.URL.API)
		if err != nil {
			return sdk.WrapError(err, "applyTemplateOnApplicationHandler> Error on apply template on application")
		}

		msgList := translate(r, msg)
		return WriteJSON(w, r, msgList, http.StatusOK)
	}
}
