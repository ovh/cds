package main

import (
	"io/ioutil"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/hashicorp/hcl"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

func importNewEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	format := r.FormValue("format")

	proj, errProj := project.Load(db, key, c.User, project.LoadOptions.Default)
	if errProj != nil {
		return sdk.WrapError(errProj, "importNewEnvironmentHandler> Cannot load %s", key)
	}

	var payload = &exportentities.Environment{}

	// Get body
	data, errRead := ioutil.ReadAll(r.Body)
	if errRead != nil {
		return sdk.WrapError(sdk.ErrWrongRequest, "importNewEnvironmentHandler> Unable to read body: %s", errRead)
	}

	f, errF := exportentities.GetFormat(format)
	if errF != nil {
		return sdk.WrapError(sdk.ErrWrongRequest, "importNewEnvironmentHandler> Unable to get format: %s", errF)
	}

	var errorParse error
	switch f {
	case exportentities.FormatJSON, exportentities.FormatHCL:
		errorParse = hcl.Unmarshal(data, payload)
	case exportentities.FormatYAML:
		errorParse = yaml.Unmarshal(data, payload)
	}

	if errorParse != nil {
		return sdk.WrapError(sdk.ErrWrongRequest, "importNewEnvironmentHandler> Cannot parsing: %s", errorParse)
	}

	env := payload.Environment()
	for i := range env.EnvironmentGroups {
		eg := &env.EnvironmentGroups[i]
		g, err := group.LoadGroup(db, eg.Group.Name)
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

	tx, errBegin := db.Begin()
	if errBegin != nil {
		return sdk.WrapError(errBegin, "importNewEnvironmentHandler: Cannot start transaction")
	}

	defer tx.Rollback()

	if err := environment.Import(db, proj, env, msgChan, c.User); err != nil {
		return sdk.WrapError(err, "importNewEnvironmentHandler> Error on import")
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

	if err := sanity.CheckProjectPipelines(db, proj); err != nil {
		return sdk.WrapError(err, "importNewEnvironmentHandler> Cannot check warnings")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "importNewEnvironmentHandler> Cannot commit transaction")
	}

	return WriteJSON(w, r, msgListString, http.StatusOK)
}

func importIntoEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	envName := vars["permEnvironmentName"]
	format := r.FormValue("format")

	proj, errProj := project.Load(db, key, c.User, project.LoadOptions.Default)
	if errProj != nil {
		return sdk.WrapError(errProj, "importIntoEnvironmentHandler> Cannot load %s", key)
	}

	tx, errBegin := db.Begin()
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
		return sdk.WrapError(sdk.ErrWrongRequest, "importIntoEnvironmentHandler> Unable to read body: %s", errRead)
	}

	f, errF := exportentities.GetFormat(format)
	if errF != nil {
		return sdk.WrapError(sdk.ErrWrongRequest, "importIntoEnvironmentHandler> Unable to get format: %s", errF)
	}

	var errorParse error
	switch f {
	case exportentities.FormatJSON, exportentities.FormatHCL:
		errorParse = hcl.Unmarshal(data, payload)
	case exportentities.FormatYAML:
		errorParse = yaml.Unmarshal(data, payload)
	}

	if errorParse != nil {
		return sdk.WrapError(sdk.ErrWrongRequest, "importIntoEnvironmentHandler> Cannot parsing: %s", errorParse)
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

	if err := environment.ImportInto(tx, proj, newEnv, env, msgChan, c.User); err != nil {
		return sdk.WrapError(err, "importIntoEnvironmentHandler> Error on import")
	}

	if err := project.UpdateLastModified(db, c.User, proj); err != nil {
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
		return sdk.WrapError(err, "importIntoEnvironmentHandler> Cannot commit transaction")
	}

	if err := sanity.CheckProjectPipelines(db, proj); err != nil {
		return sdk.WrapError(err, "importIntoEnvironmentHandler> Cannot check warnings")
	}

	return WriteJSON(w, r, msgListString, http.StatusOK)
}
