package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/build"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/log"

	"github.com/ovh/cds/sdk"
)

// notifHandler is call from worker (through API) and post to notification system
func notifHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	// Get action build ID in URL
	vars := mux.Vars(r)
	id := vars["actionBuildId"]

	// Load Queue
	ab, err := build.LoadActionBuild(db, id)
	if err != nil {
		log.Warning("PostNotifHandler> Cannot load build %s from db: %s\n", id, err)
		WriteError(w, r, sdk.ErrNotFound)
		return
	}

	idWorker, err := worker.FindBuildingWorker(db, id)
	if err != nil {
		log.Warning("PostNotifHandler> cannot load calling worker: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

	if idWorker != c.WorkerID {
		log.Warning("PostNotifHandler> this worker (%s) doesn't work on actionBuildId: %s\n", idWorker, id)
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("PostNotifHandler> Cannot read body: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

	// Unmarshal into notif
	var notif sdk.Notif

	err = json.Unmarshal([]byte(data), &notif)
	if err != nil {
		log.Warning("notifHandler> Cannot unmarshal Result: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}
	notification.SendBuiltinNotif(db, &ab, notif)
}
