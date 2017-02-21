package main

import (
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func getParametersInPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	pipelineName := vars["permPipelineKey"]

	p, err := pipeline.LoadPipeline(db, key, pipelineName, false)
	if err != nil {
		log.Warning("getParametersInPipelineHandler: Cannot load %s: %s\n", pipelineName, err)
		return err
	}

	parameters, err := pipeline.GetAllParametersInPipeline(db, p.ID)
	if err != nil {
		log.Warning("getParametersInPipelineHandler: Cannot get parameters for pipeline %s: %s\n", pipelineName, err)
		return err
	}

	return WriteJSON(w, r, parameters, http.StatusOK)
}

func deleteParameterFromPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	pipelineName := vars["permPipelineKey"]
	paramName := vars["name"]

	p, err := pipeline.LoadPipeline(db, key, pipelineName, false)
	if err != nil {
		log.Warning("deleteParameterFromPipelineHandler: Cannot load %s: %s\n", pipelineName, err)
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteParameterFromPipelineHandler: Cannot start transaction: %s\n", err)
		return err
	}
	defer tx.Rollback()

	if err := pipeline.DeleteParameterFromPipeline(tx, p.ID, paramName); err != nil {
		log.Warning("deleteParameterFromPipelineHandler: Cannot delete %s: %s\n", paramName, err)
		return err
	}

	if err := pipeline.UpdatePipelineLastModified(tx, p); err != nil {
		log.Warning("deleteParameterFromPipelineHandler> Cannot update pipeline last_modified date: %s", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("deleteParameterFromPipelineHandler: Cannot commit transaction: %s\n", err)
		return err
	}

	p.Parameter, err = pipeline.GetAllParametersInPipeline(db, p.ID)
	if err != nil {
		log.Warning("deleteParameterFromPipelineHandler: Cannot load pipeline parameters: %s\n", err)
		return err
	}
	return WriteJSON(w, r, p, http.StatusOK)
}

// Deprecated
func updateParametersInPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	pipelineName := vars["permPipelineKey"]

	var pipParams []sdk.Parameter
	if err := UnmarshalBody(r, &pipParams); err != nil {
		return err
	}

	pip, err := pipeline.LoadPipeline(db, key, pipelineName, false)
	if err != nil {
		log.Warning("updateParametersInPipelineHandler: Cannot load %s: %s\n", pipelineName, err)
		return err
	}
	pip.Parameter, err = pipeline.GetAllParametersInPipeline(db, pip.ID)
	if err != nil {
		log.Warning("updateParametersInPipelineHandler> Cannot GetAllParametersInPipeline: %s\n", err)
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateParametersInPipelineHandler: Cannot start transaction: %s", err)
		return sdk.ErrUnknownError
	}
	defer tx.Rollback()

	// Check with exising parameter to know whether parameter has been deleted, update or added
	var deleted, updated, added []sdk.Parameter
	var found bool
	for _, p := range pip.Parameter {
		found = false
		for _, new := range pipParams {
			// If we found a parameter with the same id but different value, then its modified
			if p.ID == new.ID {
				updated = append(updated, new)
				found = true
				break
			}
		}
		// If parameter is not found in new batch, then it  has been deleted
		if !found {
			deleted = append(deleted, p)
		}
	}

	// Added parameter are the one present in new batch but not in db
	for _, new := range pipParams {
		found = false
		for _, p := range pip.Parameter {
			if p.ID == new.ID {
				found = true
				break
			}
		}
		if !found {
			added = append(added, new)
		}
	}

	// Ok now permform actual update
	for i := range added {
		p := &added[i]
		if err := pipeline.InsertParameterInPipeline(tx, pip.ID, p); err != nil {
			log.Warning("UpdatePipelineParameters> Cannot insert new params %s: %s", p.Name, err)
			return err
		}
	}
	for _, p := range updated {
		if err := pipeline.UpdateParameterInPipeline(tx, pip.ID, p); err != nil {
			log.Warning("UpdatePipelineParameters> Cannot update parameter %s: %s", p.Name, err)
			return err
		}
	}
	for _, p := range deleted {
		if err := pipeline.DeleteParameterFromPipeline(tx, pip.ID, p.Name); err != nil {
			log.Warning("UpdatePipelineParameters> Cannot delete parameter %s: %s", p.Name, err)
			return err
		}
	}

	query := `
			UPDATE application
			SET last_modified = current_timestamp
			FROM application_pipeline
			WHERE application_pipeline.application_id = application.id
			AND application_pipeline.pipeline_id = $1
		`
	if _, err := tx.Exec(query, pip.ID); err != nil {
		log.Warning("UpdatePipelineParameters> Cannot update linked application [%d]: %s", pip.ID, err)
		return err
	}

	if err := pipeline.UpdatePipelineLastModified(tx, pip); err != nil {
		log.Warning("UpdatePipelineParameters> Cannot update pipeline last_modified date: %s", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("updateParametersInPipelineHandler: Cannot commit transaction: %s", err)
		return sdk.ErrUnknownError
	}

	return WriteJSON(w, r, append(added, updated...), http.StatusOK)
}

func updateParameterInPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	pipelineName := vars["permPipelineKey"]
	paramName := vars["name"]

	var newParam sdk.Parameter
	if err := UnmarshalBody(r, &newParam); err != nil {
		return err
	}
	if newParam.Name != paramName {
		return sdk.ErrWrongRequest
	}

	p, err := pipeline.LoadPipeline(db, key, pipelineName, false)
	if err != nil {
		log.Warning("updateParameterInPipelineHandler: Cannot load %s: %s\n", pipelineName, err)
		return err
	}

	paramInPipeline, err := pipeline.CheckParameterInPipeline(db, p.ID, paramName)
	if err != nil {
		log.Warning("updateParameterInPipelineHandler: Cannot check if parameter %s is already in the pipeline %s: %s\n", paramName, pipelineName, err)
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateParameterInPipelineHandler: Cannot start transaction:  %s\n", err)
		return err
	}
	defer tx.Rollback()

	if paramInPipeline {
		if err := pipeline.UpdateParameterInPipeline(tx, p.ID, newParam); err != nil {
			log.Warning("updateParameterInPipelineHandler: Cannot update parameter %s in pipeline %s:  %s\n", paramName, pipelineName, err)
			return err
		}
	}

	if err := pipeline.UpdatePipelineLastModified(tx, p); err != nil {
		log.Warning("updateParameterInPipelineHandler: Cannot update pipeline last_modified date:  %s\n", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("updateParameterInPipelineHandler: Cannot commit transaction:  %s\n", err)
		return err
	}

	p.Parameter, err = pipeline.GetAllParametersInPipeline(db, p.ID)
	if err != nil {
		log.Warning("updateParameterInPipelineHandler: Cannot load pipeline parameters:  %s\n", err)
		return err
	}
	return WriteJSON(w, r, p, http.StatusOK)
}

func addParameterInPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	pipelineName := vars["permPipelineKey"]
	paramName := vars["name"]

	var newParam sdk.Parameter
	if err := UnmarshalBody(r, &newParam); err != nil {
		return err
	}
	if newParam.Name != paramName {
		log.Warning("addParameterInPipelineHandler> Wrong param name got %s instead of %s", newParam.Name, paramName)
		return sdk.ErrWrongRequest
	}

	p, err := pipeline.LoadPipeline(db, key, pipelineName, false)
	if err != nil {
		log.Warning("addParameterInPipelineHandler: Cannot load %s: %s\n", pipelineName, err)
		return err
	}

	paramInProject, err := pipeline.CheckParameterInPipeline(db, p.ID, paramName)
	if err != nil {
		log.Warning("addParameterInPipelineHandler: Cannot check if parameter %s is already in the pipeline %s: %s\n", paramName, pipelineName, err)
		return err
	}
	if paramInProject {
		log.Warning("addParameterInPipelineHandler:Parameter %s is already in the pipeline %s\n", paramName, pipelineName)
		return sdk.ErrParameterExists
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("addParameterInPipelineHandler: Cannot start transaction: %s\n", err)
		return err
	}
	defer tx.Rollback()

	if !paramInProject {
		if err := pipeline.InsertParameterInPipeline(tx, p.ID, &newParam); err != nil {
			log.Warning("addParameterInPipelineHandler: Cannot add parameter %s in pipeline %s:  %s\n", paramName, pipelineName, err)
			return err
		}
	}

	if err := pipeline.UpdatePipelineLastModified(tx, p); err != nil {
		log.Warning("addParameterInPipelineHandler> Cannot update pipeline last_modified date: %s", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("addParameterInPipelineHandler: Cannot commit transaction: %s\n", err)
		return err
	}

	p.Parameter, err = pipeline.GetAllParametersInPipeline(db, p.ID)
	if err != nil {
		log.Warning("addParameterInPipelineHandler: Cannot get pipeline parameters: %s\n", err)
		return err
	}

	return WriteJSON(w, r, p, http.StatusOK)
}
