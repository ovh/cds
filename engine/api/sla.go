package main

import (
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/context"
)

func slaHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {
	vars := mux.Vars(r)
	date := vars["date"]

	if date == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	query := `UPDATE sla SET count = count + 1 WHERE "date" = $1`
	_, err := db.Exec(query, date)
	if err == nil {
		return
	}

	query = `INSERT INTO sla ("date", "count") VALUES ($1, $2)`
	_, err = db.Exec(query, date, 1)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
