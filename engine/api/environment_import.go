package api

import (
	"context"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"

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
		force := service.FormBool(r, "force")

		proj, err := project.Load(ctx, api.mustDB(), key, project.LoadOptions.WithGroups)
		if err != nil {
			return sdk.WrapError(err, "unable load project")
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return sdk.NewError(sdk.ErrWrongRequest, err)
		}
		defer r.Body.Close() // nolint

		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(body)
		}
		format, err := exportentities.GetFormatFromContentType(contentType)
		if err != nil {
			return err
		}

		var data exportentities.Environment
		if err := exportentities.Unmarshal(body, format, &data); err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "unable to start tx")
		}
		defer tx.Rollback() // nolint

		_, _, msgList, globalError := environment.ParseAndImport(tx, *proj, data, environment.ImportOptions{Force: force}, project.DecryptWithBuiltinKey, getAPIConsumer(ctx))
		msgListString := translate(r, msgList)
		if globalError != nil {
			globalError = sdk.WrapError(globalError, "Unable to import environment %s", data.Name)
			if sdk.ErrorIsUnknown(globalError) {
				return globalError
			}
			sdkErr := sdk.ExtractHTTPError(globalError, r.Header.Get("Accept-Language"))
			return service.WriteJSON(w, append(msgListString, sdkErr.Message), sdkErr.Status)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, msgListString, http.StatusOK)
	}
}

func (api *API) importNewEnvironmentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]

		proj, err := project.Load(ctx, api.mustDB(), key, project.LoadOptions.Default,
			project.LoadOptions.WithGroups, project.LoadOptions.WithPermission)
		if err != nil {
			return sdk.WrapError(err, "cannot load %s", key)
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return sdk.NewError(sdk.ErrWrongRequest, sdk.WrapError(err, "unable to read body"))
		}

		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(body)
		}
		format, err := exportentities.GetFormatFromContentType(contentType)
		if err != nil {
			return err
		}

		var data exportentities.Environment
		if err := exportentities.Unmarshal(body, format, &data); err != nil {
			return err
		}

		env := data.Environment()
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

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err := environment.Import(tx, *proj, env, msgChan, getAPIConsumer(ctx)); err != nil {
			return sdk.WithStack(err)
		}

		close(msgChan)
		<-done

		msgListString := translate(r, allMsg)

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, msgListString, http.StatusOK)
	}
}

func (api *API) importIntoEnvironmentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		envName := vars["environmentName"]

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err := environment.Lock(tx, key, envName); err != nil {
			return sdk.WrapError(err, "cannot lock env %s/%s", key, envName)
		}

		env, err := environment.LoadEnvironmentByName(tx, key, envName)
		if err != nil {
			return sdk.WrapError(err, "cannot load env %s/%s", key, envName)
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to read body"))
		}

		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(body)
		}
		format, err := exportentities.GetFormatFromContentType(contentType)
		if err != nil {
			return err
		}

		var data exportentities.Environment
		if err := exportentities.Unmarshal(body, format, &data); err != nil {
			return err
		}

		newEnv := data.Environment()
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

		if err := environment.ImportInto(tx, newEnv, env, msgChan, getAPIConsumer(ctx)); err != nil {
			return sdk.WithStack(err)
		}

		close(msgChan)
		<-done

		msgListString := translate(r, allMsg)

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, msgListString, http.StatusOK)
	}
}
