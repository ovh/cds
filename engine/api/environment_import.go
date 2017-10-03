package api

import (
	"context"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/hashicorp/hcl"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) importNewEnvironmentHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]
		format := r.FormValue("format")

		proj, errProj := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default, project.LoadOptions.WithGroups, project.LoadOptions.WithPermission)
		if errProj != nil {
			log.Warning("importNewEnvironmentHandler> Cannot load %s: %s\n", key, errProj)
			return errProj
		}

		var payload = &exportentities.Environment{}

		// Get body
		data, errRead := ioutil.ReadAll(r.Body)
		if errRead != nil {
			log.Warning("importNewEnvironmentHandler> Unable to read body : %s\n", errRead)
			return sdk.ErrWrongRequest
		}

		f, errF := exportentities.GetFormat(format)
		if errF != nil {
			log.Warning("importNewEnvironmentHandler> Unable to get format : %s\n", errF)
			return sdk.ErrWrongRequest
		}

		var errorParse error
		switch f {
		case exportentities.FormatJSON, exportentities.FormatHCL:
			errorParse = hcl.Unmarshal(data, payload)
		case exportentities.FormatYAML:
			errorParse = yaml.Unmarshal(data, payload)
		}

		if errorParse != nil {
			log.Warning("importNewEnvironmentHandler> Cannot parsing: %s\n", errorParse)
			return sdk.ErrWrongRequest
		}

		env := payload.Environment()
		for i := range env.EnvironmentGroups {
			eg := &env.EnvironmentGroups[i]
			g, err := group.LoadGroup(api.mustDB(), eg.Group.Name)
			if err != nil {
				log.Warning("importNewEnvironmentHandler> Error on import : %s", err)
				return err
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
			log.Warning("importNewEnvironmentHandler: Cannot start transaction: %s\n", errBegin)
			return errBegin
		}

		defer tx.Rollback()

		if err := environment.Import(api.mustDB(), proj, env, msgChan, getUser(ctx)); err != nil {
			log.Warning("importNewEnvironmentHandler> Error on import : %s", err)
			return err
		}

		close(msgChan)
		<-done

		al := r.Header.Get("Accept-Language")
		msgListString := []string{}

		for _, m := range allMsg {
			s := m.String(al)
			if s != "" {
				msgListString = append(msgListString, s)
			}
		}

		if err := sanity.CheckProjectPipelines(api.mustDB(), api.Cache, proj); err != nil {
			log.Warning("importNewEnvironmentHandler> Cannot check warnings: %s\n", err)
			return err
		}

		if err := tx.Commit(); err != nil {
			log.Warning("importNewEnvironmentHandler> Cannot commit transaction: %s\n", err)
			return err
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
			log.Warning("importIntoEnvironmentHandler> Cannot load %s: %s\n", key, errProj)
			return errProj
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			log.Warning("importIntoEnvironmentHandler: Cannot start transaction: %s\n", errBegin)
			return errBegin
		}

		defer tx.Rollback()

		if err := environment.Lock(tx, key, envName); err != nil {
			log.Warning("importIntoEnvironmentHandler> Cannot lock env %s/%s: %s\n", key, envName, err)
			return err
		}

		env, errEnv := environment.LoadEnvironmentByName(tx, key, envName)
		if errEnv != nil {
			log.Warning("importIntoEnvironmentHandler> Cannot load env %s/%s: %s\n", key, envName, errEnv)
			return errEnv
		}

		var payload = &exportentities.Environment{}

		// Get body
		data, errRead := ioutil.ReadAll(r.Body)
		if errRead != nil {
			log.Warning("importIntoEnvironmentHandler> Unable to read body : %s\n", errRead)
			return sdk.ErrWrongRequest
		}

		f, errF := exportentities.GetFormat(format)
		if errF != nil {
			log.Warning("importIntoEnvironmentHandler> Unable to get format : %s\n", errF)
			return sdk.ErrWrongRequest
		}

		var errorParse error
		switch f {
		case exportentities.FormatJSON, exportentities.FormatHCL:
			errorParse = hcl.Unmarshal(data, payload)
		case exportentities.FormatYAML:
			errorParse = yaml.Unmarshal(data, payload)
		}

		if errorParse != nil {
			log.Warning("importIntoEnvironmentHandler> Cannot parsing: %s\n", errorParse)
			return sdk.ErrWrongRequest
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
			log.Warning("importIntoEnvironmentHandler> Error on import : %s", err)
			return err
		}

		if err := project.UpdateLastModified(api.mustDB(), api.Cache, getUser(ctx), proj); err != nil {
			return sdk.WrapError(err, "importIntoEnvironmentHandler> Cannot update project last modified date")
		}

		close(msgChan)
		<-done

		al := r.Header.Get("Accept-Language")
		msgListString := []string{}

		for _, m := range allMsg {
			s := m.String(al)
			if s != "" {
				msgListString = append(msgListString, s)
			}
		}

		if err := tx.Commit(); err != nil {
			log.Warning("importIntoEnvironmentHandler> Cannot commit transaction: %s\n", err)
			return err
		}

		if err := sanity.CheckProjectPipelines(api.mustDB(), api.Cache, proj); err != nil {
			log.Warning("importIntoEnvironmentHandler> Cannot check warnings: %s\n", err)
			return err
		}

		return WriteJSON(w, r, msgListString, http.StatusOK)
	}
}
