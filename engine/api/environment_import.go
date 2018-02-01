package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

// postEnvironmentImportHandler import an environment yml file
// getActionsHandler Retrieve all public actions
// @title import an environment yml file
// @description import an environment yml file with `cdsctl environment import myenv.env.yml`
// @params force=true or false. If false and if the environment already exists, raise an error
func (api *API) postEnvironmentImportHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]
		force := FormBool(r, "force")

		//Load project
		proj, errp := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.WithGroups)
		if errp != nil {
			return sdk.WrapError(errp, "postEnvironmentImportHandler>> Unable load project")
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
			return sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("unsupported content-type: %s", contentType))
		}

		if errenv != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errenv)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "postEnvironmentImportHandler> Unable to start tx")
		}
		defer tx.Rollback()

		msgList, globalError := environment.ParseAndImport(tx, api.Cache, proj, eenv, force, project.DecryptWithBuiltinKey, getUser(ctx))
		msgListString := translate(r, msgList)

		if globalError != nil {
			myError, ok := globalError.(sdk.Error)
			if ok {
				return WriteJSON(w, r, msgListString, myError.Status)
			}
			return sdk.WrapError(globalError, "postEnvironmentImportHandler> Unable import environment %s", eenv.Name)
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), proj, sdk.ProjectPipelineLastModificationType); err != nil {
			return sdk.WrapError(err, "postEnvironmentImportHandler> Unable to update project")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "postEnvironmentImportHandler> Cannot commit transaction")
		}

		return WriteJSON(w, r, msgListString, http.StatusOK)
	}
}

func (api *API) importNewEnvironmentHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]
		format := r.FormValue("format")

		proj, errProj := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default, project.LoadOptions.WithGroups, project.LoadOptions.WithPermission)
		if errProj != nil {
			return sdk.WrapError(errProj, "importNewEnvironmentHandler> Cannot load %s", key)
		}

		var payload = &exportentities.Environment{}

		// Get body
		data, errRead := ioutil.ReadAll(r.Body)
		if errRead != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "importNewEnvironmentHandler> Unable to read body")
		}

		f, errF := exportentities.GetFormat(format)
		if errF != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "importNewEnvironmentHandler> Unable to get format")
		}

		var errorParse error
		switch f {
		case exportentities.FormatJSON:
			errorParse = json.Unmarshal(data, payload)
		case exportentities.FormatYAML:
			errorParse = yaml.Unmarshal(data, payload)
		}

		if errorParse != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "importNewEnvironmentHandler> Cannot parsing")
		}

		env := payload.Environment()
		for i := range env.EnvironmentGroups {
			eg := &env.EnvironmentGroups[i]
			g, err := group.LoadGroup(api.mustDB(), eg.Group.Name)
			if err != nil {
				return sdk.WrapError(err, "importNewEnvironmentHandler> Error on import")
			}
			eg.Group = *g
		}

		allMsg := []sdk.Message{}
		msgChan := make(chan sdk.Message, 10)
		done := make(chan bool)

		go func() {
			for {
				msg, ok := <-msgChan
				log.Debug("importNewEnvironmentHandler >>> %s", msg)
				allMsg = append(allMsg, msg)
				if !ok {
					done <- true
				}
			}
		}()

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "importNewEnvironmentHandler: Cannot start transaction")
		}

		defer tx.Rollback()

		if err := environment.Import(api.mustDB(), proj, env, msgChan, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "importNewEnvironmentHandler> Error on import")
		}

		close(msgChan)
		<-done

		msgListString := translate(r, allMsg)

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "importNewEnvironmentHandler> Cannot commit transaction")
		}

		return WriteJSON(w, r, msgListString, http.StatusOK)
	}
}

func (api *API) importIntoEnvironmentHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		envName := vars["permEnvironmentName"]
		format := r.FormValue("format")

		proj, errProj := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default, project.LoadOptions.WithGroups, project.LoadOptions.WithPermission)
		if errProj != nil {
			return sdk.WrapError(errProj, "importIntoEnvironmentHandler> Cannot load %s", key)
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "importIntoEnvironmentHandler: Cannot start transaction")
		}

		defer tx.Rollback()

		if err := environment.Lock(tx, key, envName); err != nil {
			return sdk.WrapError(err, "importIntoEnvironmentHandler> Cannot lock env %s/%s", key, envName)
		}

		env, errEnv := environment.LoadEnvironmentByName(tx, key, envName)
		if errEnv != nil {
			return sdk.WrapError(errEnv, "importIntoEnvironmentHandler> Cannot load env %s/%s", key, envName)
		}

		var payload = &exportentities.Environment{}

		// Get body
		data, errRead := ioutil.ReadAll(r.Body)
		if errRead != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "importIntoEnvironmentHandler> Unable to read body")
		}

		f, errF := exportentities.GetFormat(format)
		if errF != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "importIntoEnvironmentHandler> Unable to get format")
		}

		var errorParse error
		switch f {
		case exportentities.FormatJSON:
			errorParse = json.Unmarshal(data, payload)
		case exportentities.FormatYAML:
			errorParse = yaml.Unmarshal(data, payload)
		}

		if errorParse != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "importIntoEnvironmentHandler> Cannot parsing")
		}

		newEnv := payload.Environment()

		for i := range newEnv.EnvironmentGroups {
			eg := &newEnv.EnvironmentGroups[i]
			g, err := group.LoadGroup(tx, eg.Group.Name)
			if err != nil {
				return sdk.WrapError(err, "importIntoEnvironmentHandler> Error on import")
			}
			eg.Group = *g
		}
		allMsg := []sdk.Message{}
		msgChan := make(chan sdk.Message, 10)
		done := make(chan bool)

		go func() {
			for {
				msg, ok := <-msgChan
				log.Debug("importIntoEnvironmentHandler >>> %s", msg)
				allMsg = append(allMsg, msg)
				if !ok {
					done <- true
				}
			}
		}()

		if err := environment.ImportInto(tx, proj, newEnv, env, msgChan, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "importIntoEnvironmentHandler> Error on import")
		}

		if err := project.UpdateLastModified(api.mustDB(), api.Cache, getUser(ctx), proj, sdk.ProjectEnvironmentLastModificationType); err != nil {
			return sdk.WrapError(err, "importIntoEnvironmentHandler> Cannot update project last modified date")
		}

		close(msgChan)
		<-done

		msgListString := translate(r, allMsg)

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "importIntoEnvironmentHandler> Cannot commit transaction")
		}

		return WriteJSON(w, r, msgListString, http.StatusOK)
	}
}
