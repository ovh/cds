package main

import (
	"net/http"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getUserWarnings(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {

	al := r.Header.Get("Accept-Language")

	var warnings []sdk.Warning
	var err error
	if c.User.Admin {
		warnings, err = sanity.LoadAllWarnings(db, al)
	} else {
		warnings, err = sanity.LoadUserWarnings(db, al, c.User.ID)
	}
	if err != nil {
		log.Warning("getUserWarnings> Cannot load user %d warnings: %s\n", c.User.ID, err)
		return err

	}

	return WriteJSON(w, r, warnings, http.StatusOK)
}
