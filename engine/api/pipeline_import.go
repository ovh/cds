package main

import (
	"io/ioutil"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/hashicorp/hcl"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/msg"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func importPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	format := r.FormValue("format")
	forceUpdate := FormBool(r, "forceUpdate")

	// Load project
	proj, err := project.Load(db, key, c.User, project.LoadOptions.Default)
	if err != nil {
		return sdk.WrapError(err, "importPipelineHandler> Unable to load project %s", key)
	}

	// Get body
	data, errRead := ioutil.ReadAll(r.Body)
	if errRead != nil {
		return sdk.WrapError(sdk.ErrWrongRequest, "importPipelineHandler> Unable to read body")
	}

	// Compute format
	f, errF := exportentities.GetFormat(format)
	if errF != nil {
		return sdk.WrapError(sdk.ErrWrongRequest, "importPipelineHandler> Unable to get format : %s", errF)
	}

	// Parse the pipeline
	payload := &exportentities.Pipeline{}
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

	// Check if pipeline exists
	exist, err := pipeline.ExistPipeline(db, proj.ID, payload.Name)
	if err != nil {
		return sdk.WrapError(err, "importPipelineHandler> Unable to check if pipeline %s exists", payload.Name)
	}

	//Transform payload to a sdk.Pipeline
	pip, err := payload.Pipeline()
	if err != nil {
		return sdk.WrapError(err, "importPipelineHandler> Unable to parse pipeline %s", payload.Name)
	}

	// Load group in permission
	for i := range pip.GroupPermission {
		eg := &pip.GroupPermission[i]
		g, err := group.LoadGroup(db, eg.Group.Name)
		if err != nil {
			return sdk.WrapError(err, "importPipelineHandler> Error loading groups for permission")
		}
		eg.Group = *g
	}

	allMsg := []msg.Message{}
	msgChan := make(chan msg.Message, 10)
	done := make(chan bool)

	go func() {
		for {
			msg, ok := <-msgChan
			log.Debug("importPipelineHandler >>> %s", msg)
			allMsg = append(allMsg, msg)
			if !ok {
				done <- true
			}
		}
	}()

	tx, errBegin := db.Begin()
	if errBegin != nil {
		log.Warning("importPipelineHandler: Cannot start transaction: %s\n", errBegin)
		return sdk.WrapError(errBegin, "importPipelineHandler: Cannot start transaction")
	}

	defer tx.Rollback()

	var globalError error

	if exist && !forceUpdate {
		return sdk.ErrPipelineAlreadyExists
	} else if exist {
		globalError = pipeline.ImportUpdate(tx, proj, pip, msgChan)
	} else {
		// Import new pipeline
		globalError = pipeline.Import(tx, proj, pip, msgChan)
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

	if globalError != nil {
		myError, ok := globalError.(*sdk.Error)
		if ok {
			return WriteJSON(w, r, msgListString, myError.Status)
		}
		return sdk.WrapError(err, "importPipelineHandler> Unable import pipeline")
	}

	if err := project.UpdateLastModified(tx, c.User, proj); err != nil {
		return sdk.WrapError(err, "importPipelineHandler> Unable to update project")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "importPipelineHandler> Cannot commit transaction")
	}

	var errp error
	proj.Pipelines, errp = pipeline.LoadPipelines(db, proj.ID, true, c.User)
	if errp != nil {
		return sdk.WrapError(errp, "importPipelineHandler> Unable to reload pipelines for project %s", proj.Key)
	}

	if err := sanity.CheckProjectPipelines(db, proj); err != nil {
		return sdk.WrapError(err, "importPipelineHandler> Cannot check warnings")
	}

	return WriteJSON(w, r, msgListString, http.StatusOK)
}
