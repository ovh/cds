package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/template"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func getBuildTemplates(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	var tpl []sdk.Template
	WriteJSON(w, r, tpl, http.StatusOK)
}

func getDeployTemplates(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	var tpl []sdk.Template
	WriteJSON(w, r, tpl, http.StatusOK)
}

func applyTemplateHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]

	// Get data in body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	var app sdk.Application
	err = json.Unmarshal(data, &app)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	p, err := project.LoadProject(db, projectKey, c.User)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	err = group.LoadGroupByProject(db, p)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("applyTemplateHandler> Cannot start tx: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	// check application name pattern
	regexp := regexp.MustCompile(sdk.NamePattern)
	if !regexp.MatchString(app.Name) {
		log.Warning("applyTemplateHandler: Application name %s do not respect pattern %s", app.Name, sdk.NamePattern)
		WriteError(w, r, sdk.ErrInvalidApplicationPattern)
		return
	}

	err = template.ApplyTemplate(tx, p, &app)
	if err != nil {
		log.Warning("applyTemplateHandler> Cannot apply template: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("applyTemplateHandler> Cannot commit tx: %s\n", err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, app, http.StatusOK)
}
