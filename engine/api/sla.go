package main

import (
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/sdk"
)

func slaHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	date := vars["date"]

	if date == "" {
		return sdk.ErrWrongRequest
	}

	query := `UPDATE sla SET count = count + 1 WHERE "date" = $1`
	_, err := db.Exec(query, date)
	if err == nil {
		return nil
	}

	query = `INSERT INTO sla ("date", "count") VALUES ($1, $2)`
	_, err = db.Exec(query, date, 1)
	if err != nil {
		return err
	}

	return nil
}
