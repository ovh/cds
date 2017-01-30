package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/hashicorp/hcl"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/msg"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func importNewEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	format := r.FormValue("format")

	proj, errProj := project.LoadProject(db, key, c.User, project.WithApplications(1), project.WithVariables())
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
		g, err := group.LoadGroup(db, eg.Group.Name)
		if err != nil {
			log.Warning("importNewEnvironmentHandler> Error on import : %s", err)
			return err
		}
		eg.Group = *g
	}

	allMsg := []msg.Message{}
	msgChan := make(chan msg.Message, 10)

	tx, errBegin := db.Begin()
	if errBegin != nil {
		log.Warning("importNewEnvironmentHandler: Cannot start transaction: %s\n", errBegin)
		return errBegin
	}

	defer tx.Rollback()

	if err := environment.Import(db, proj, env, msgChan); err != nil {
		log.Warning("importNewEnvironmentHandler> Error on import : %s", err)
		return err
	}

	close(msgChan)

	for msg := range msgChan {
		log.Debug("importNewEnvironmentHandler >>> %s", msg)
		allMsg = append(allMsg, msg)
	}

	al := r.Header.Get("Accept-Language")
	msgListString := []string{}

	fmt.Println(allMsg)

	for _, m := range allMsg {
		s := m.String(al)
		msgListString = append(msgListString, s)
	}

	if err := sanity.CheckProjectPipelines(db, proj); err != nil {
		log.Warning("importNewEnvironmentHandler> Cannot check warnings: %s\n", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("importNewEnvironmentHandler> Cannot commit transaction: %s\n", err)
		return err
	}

	return WriteJSON(w, r, msgListString, http.StatusOK)
}

func importIntoEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return errors.New("Not yet implemented")
}
