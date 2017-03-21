package main

import (
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/poller"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func addPollerHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]
	pipName := vars["permPipelineKey"]

	//Load the application
	app, err := application.LoadByName(db, projectKey, appName, c.User, application.LoadOptions.Default)
	if err != nil {
		return sdk.WrapError(err, "addPollerHandler> Cannot load application")
	}

	app.RepositoryPollers, err = poller.LoadByApplication(db, app.ID)
	if err != nil {
		return sdk.WrapError(err, "addPollerHandler> cannot load application poller %s/%s", projectKey, appName)
	}

	//Find the pipeline
	var pip *sdk.Pipeline
	for _, p := range app.Pipelines {
		if p.Pipeline.Name == pipName {
			pip = &p.Pipeline
			break
		}
	}

	//Check if pipeline has been found
	if pip == nil {
		log.Warning("addPollerHandler> Cannot load pipeline: %s", pipName)
		return sdk.WrapError(sdk.ErrPipelineNotFound, "sdk.ErrPipelineNotFound", pipName)
	}

	var h sdk.RepositoryPoller
	if err := UnmarshalBody(r, &h); err != nil {
		return sdk.WrapError(err, "addPollerHandler> Cannot unmarshal body")
	}

	h.Application = *app
	h.Pipeline = *pip
	h.Enabled = true

	//Check it the application is attached to a repository
	if app.RepositoriesManager == nil {
		return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "addPollerHandler> No repository on application")
	}

	b, e := repositoriesmanager.CheckApplicationIsAttached(db, app.RepositoriesManager.Name, projectKey, appName)
	if e != nil {
		return sdk.WrapError(e, "addPollerHandler> Cannot check app (%s,%s,%s)", app.RepositoriesManager.Name, projectKey, appName)
	}

	if !b && app.RepositoryFullname == "" {
		return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "addPollerHandler> No repository on application")
	}

	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(e, "addPollerHandler> Cannot start transaction")
	}
	defer tx.Rollback()

	// Insert poller in database
	err = poller.Insert(db, &h)
	if err != nil {
		return sdk.WrapError(err, "addPollerHandler: cannot insert poller in db")
	}

	err = application.UpdateLastModified(tx, app, c.User)
	if err != nil {
		return sdk.WrapError(err, "addPollerHandler: cannot update application (%s) lastmodified date", app.Name)
	}

	err = tx.Commit()
	if err != nil {
		return sdk.WrapError(err, "addPollerHandler> Cannot commit transaction")
	}

	app.RepositoryPollers = append(app.RepositoryPollers, h)
	var errW error
	app.Workflows, errW = workflow.LoadCDTree(db, projectKey, appName, c.User)
	if errW != nil {
		return sdk.WrapError(errW, "addPollerHandler> Cannot load workflow")
	}

	return WriteJSON(w, r, app, http.StatusOK)
}

func updatePollerHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]
	pipName := vars["permPipelineKey"]

	//Load the application
	app, err := application.LoadByName(db, projectKey, appName, c.User, application.LoadOptions.Default)
	if err != nil {
		return sdk.WrapError(err, "updatePollerHandler> Cannot load application")
	}

	app.RepositoryPollers, err = poller.LoadByApplication(db, app.ID)
	if err != nil {
		return sdk.WrapError(err, "updatePollerHandler> cannot load application poller %s/%s", projectKey, appName)
	}

	//Find the pipeline
	var pip *sdk.Pipeline
	for _, p := range app.Pipelines {
		if p.Pipeline.Name == pipName {
			pip = &p.Pipeline
			break
		}
	}

	//Check if pipeline has been found
	if pip == nil {
		return sdk.WrapError(sdk.ErrPipelineNotFound, "updatePollerHandler> Cannot load pipeline: %s", pipName)
	}

	var h sdk.RepositoryPoller
	if err := UnmarshalBody(r, &h); err != nil {
		return err
	}

	h.Application = *app
	h.Pipeline = *pip

	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "updatePollerHandler> cannot start transaction")

	}
	defer tx.Rollback()

	// Update poller in database
	err = poller.Update(tx, &h)
	if err != nil {
		return sdk.WrapError(err, "updatePollerHandler: cannot update poller in db")

	}

	if err = application.UpdateLastModified(tx, app, c.User); err != nil {
		return sdk.WrapError(err, "updatePollerHandler: cannot update application last modified date")
	}

	if err = tx.Commit(); err != nil {
		return sdk.WrapError(err,"updatePollerHandler> cannot commit transaction" )
	}

	app.RepositoryPollers, err = poller.LoadByApplication(db, app.ID)
	if err != nil {
		return sdk.WrapError(err, "updatePollerHandler> cannot load pollers")
	}
	var errW error
	app.Workflows, errW = workflow.LoadCDTree(db, projectKey, appName, c.User)
	if errW != nil {
		return sdk.WrapError(errW, "updatePollerHandler> Cannot load workflow")
	}

	return WriteJSON(w, r, app, http.StatusOK)
}

func getApplicationPollersHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectName := vars["key"]
	appName := vars["permApplicationName"]

	a, err := application.LoadByName(db, projectName, appName, c.User)
	if err != nil {
		return sdk.WrapError(err, "getApplicationHooksHandler> cannot load application %s/%s", projectName, appName)
	}

	a.RepositoryPollers, err = poller.LoadByApplication(db, a.ID)
	if err != nil {
		log.Warning("getApplicationHooksHandler> cannot load application poller %s/%s: %s\n", projectName, appName, err)
		return sdk.WrapError(err, "getApplicationHooksHandler> cannot load application poller %s/%s", projectName, appName,)
	}

	return WriteJSON(w, r, a.RepositoryPollers, http.StatusOK)
}

func getPollersHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectName := vars["key"]
	appName := vars["permApplicationName"]
	pipelineName := vars["permPipelineKey"]

	p, err := pipeline.LoadPipeline(db, projectName, pipelineName, false)
	if err != nil {
		return sdk.WrapError(err, "getPollersHandler> cannot load pipeline %s/%s", projectName, pipelineName)
	}

	a, err := application.LoadByName(db, projectName, appName, c.User)
	if err != nil {
		log.Warning("getPollersHandler> cannot load application %s/%s: %s\n", projectName, appName, err)
		return sdk.WrapError(err, "getPollersHandler> cannot load application %s/%s", projectName, appName)

	}

	poller, err := poller.LoadByApplicationAndPipeline(db, a.ID, p.ID)
	if err != nil {
		return sdk.WrapError(err, "getPollersHandler> cannot load poller with ID %d %d", p.ID, a.ID)

	}

	return WriteJSON(w, r, poller, http.StatusOK)
}

func deletePollerHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]
	pipelineName := vars["permPipelineKey"]

	p, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		return sdk.WrapError(err, "deletePollerHandler> cannot load pipeline %s/%s", projectKey, pipelineName)

	}

	a, err := application.LoadByName(db, projectKey, appName, c.User)
	if err != nil {
		return sdk.WrapError(err, "deletePollerHandler> cannot load application %s/%s", projectKey, appName)
	}

	po, err := poller.LoadByApplicationAndPipeline(db, a.ID, p.ID)
	if err != nil {
		return sdk.WrapError(err, "deletePollerHandler> cannot load poller")

	}

	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "deletePollerHandler> cannot start transaction")

	}
	defer tx.Rollback()

	if err = poller.Delete(tx, po); err != nil {
		return sdk.WrapError(err, "deletePollerHandler> cannot delete poller")

	}

	if err = application.UpdateLastModified(tx, a, c.User); err != nil {
		return sdk.WrapError(err, "deletePollerHandler> cannot update application last modified date")

	}

	if err = tx.Commit(); err != nil {
		return sdk.WrapError(err, "deletePollerHandler> cannot commit transaction")

	}

	a.RepositoryPollers, err = poller.LoadByApplication(db, a.ID)
	if err != nil {
		return sdk.WrapError(err, "deletePollerHandler> cannot load pollers")

	}
	var errW error
	a.Workflows, errW = workflow.LoadCDTree(db, projectKey, appName, c.User)
	if errW != nil {
		return sdk.WrapError(errW, "deletePollerHandler> Cannot load workflow")
	}

	return WriteJSON(w, r, a, http.StatusOK)
}
