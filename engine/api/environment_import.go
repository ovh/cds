package api

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

// postEnvironmentImportHandler import an environment yml file
// getActionsHandler Retrieve all public actions
// @title import an environment yml file
// @description import an environment yml file with `cdsctl environment import myenv.env.yml`
// @params force=true or false. If false and if the environment already exists, raise an error
func (api *API) postEnvironmentImportHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		force := FormBool(r, "force")

		//Load project
		proj, errp := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx), project.LoadOptions.WithGroups)
		if errp != nil {
			return sdk.WrapError(errp, "Unable load project")
		}

		body, errr := ioutil.ReadAll(r.Body)
		if errr != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errr)
		}
		defer r.Body.Close()

		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(body)
		}

		var eenv = new(exportentities.Environment)
		var errenv error
		switch contentType {
		case "application/json":
			errenv = json.Unmarshal(body, eenv)
		case "application/x-yaml", "text/x-yam":
			errenv = yaml.Unmarshal(body, eenv)
		default:
			return sdk.WrapError(sdk.ErrWrongRequest, "Unsupported content-type: %s", contentType)
		}
		if errenv != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errenv)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Unable to start tx")
		}
		defer tx.Rollback()

		_, msgList, globalError := environment.ParseAndImport(tx, api.Cache, proj, eenv, force, project.DecryptWithBuiltinKey, deprecatedGetUser(ctx))
		msgListString := translate(r, msgList)
		if globalError != nil {
			globalError = sdk.WrapError(globalError, "Unable to import environment %s", eenv.Name)
			if sdk.ErrorIsUnknown(globalError) {
				return globalError
			}
			log.Warning("%v", globalError)
			sdkErr := sdk.ExtractHTTPError(globalError, r.Header.Get("Accept-Language"))
			return service.WriteJSON(w, append(msgListString, sdkErr.Message), sdkErr.Status)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		return service.WriteJSON(w, msgListString, http.StatusOK)
	}
}

func (api *API) importNewEnvironmentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		format := r.FormValue("format")

		proj, errProj := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx), project.LoadOptions.Default, project.LoadOptions.WithGroups, project.LoadOptions.WithPermission)
		if errProj != nil {
			return sdk.WrapError(errProj, "Cannot load %s", key)
		}

		var payload = &exportentities.Environment{}

		// Get body
		data, errRead := ioutil.ReadAll(r.Body)
		if errRead != nil {
			return sdk.NewError(sdk.ErrWrongRequest, sdk.WrapError(errRead, "Unable to read body"))
		}

		f, errF := exportentities.GetFormat(format)
		if errF != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errF)
		}

		var errorParse error
		switch f {
		case exportentities.FormatJSON:
			errorParse = json.Unmarshal(data, payload)
		case exportentities.FormatYAML:
			errorParse = yaml.Unmarshal(data, payload)
		}
		if errorParse != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errorParse)
		}

		env := payload.Environment()
		allMsg := []sdk.Message{}
		msgChan := make(chan sdk.Message, 10)
		done := make(chan bool)

		go func() {
			for {
				msg, ok := <-msgChan
				log.Debug("importNewEnvironmentHandler >>> %v", msg)
				allMsg = append(allMsg, msg)
				if !ok {
					done <- true
				}
			}
		}()

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "Cannot start transaction")
		}

		defer tx.Rollback()

		if err := environment.Import(api.mustDB(), proj, env, msgChan, deprecatedGetUser(ctx)); err != nil {
			return sdk.WrapError(err, "Error on import")
		}

		close(msgChan)
		<-done

		msgListString := translate(r, allMsg)

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		return service.WriteJSON(w, msgListString, http.StatusOK)
	}
}

func (api *API) importIntoEnvironmentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		envName := vars["environmentName"]
		format := r.FormValue("format")

		proj, errProj := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx), project.LoadOptions.Default, project.LoadOptions.WithGroups, project.LoadOptions.WithPermission)
		if errProj != nil {
			return sdk.WrapError(errProj, "Cannot load %s", key)
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "Cannot start transaction")
		}

		defer tx.Rollback()

		if err := environment.Lock(tx, key, envName); err != nil {
			return sdk.WrapError(err, "Cannot lock env %s/%s", key, envName)
		}

		env, errEnv := environment.LoadEnvironmentByName(tx, key, envName)
		if errEnv != nil {
			return sdk.WrapError(errEnv, "Cannot load env %s/%s", key, envName)
		}

		var payload = &exportentities.Environment{}

		// Get body
		data, errRead := ioutil.ReadAll(r.Body)
		if errRead != nil {
			return sdk.NewError(sdk.ErrWrongRequest, sdk.WrapError(errRead, "Unable to read body"))
		}

		f, errF := exportentities.GetFormat(format)
		if errF != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errF)
		}

		var errorParse error
		switch f {
		case exportentities.FormatJSON:
			errorParse = json.Unmarshal(data, payload)
		case exportentities.FormatYAML:
			errorParse = yaml.Unmarshal(data, payload)
		}
		if errorParse != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errorParse)
		}

		newEnv := payload.Environment()
		allMsg := []sdk.Message{}
		msgChan := make(chan sdk.Message, 10)
		done := make(chan bool)

		go func() {
			for {
				msg, ok := <-msgChan
				log.Debug("importIntoEnvironmentHandler >>> %v", msg)
				allMsg = append(allMsg, msg)
				if !ok {
					done <- true
				}
			}
		}()

		if err := environment.ImportInto(tx, proj, newEnv, env, msgChan, deprecatedGetUser(ctx)); err != nil {
			return sdk.WrapError(err, "Error on import")
		}

		close(msgChan)
		<-done

		msgListString := translate(r, allMsg)

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		return service.WriteJSON(w, msgListString, http.StatusOK)
	}
}
