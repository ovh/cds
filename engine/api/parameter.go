package main

import (
	"net/http"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/sdk"
)

func getVariableTypeHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	return WriteJSON(w, r, sdk.AvailableVariableType, http.StatusOK)
}

func getParameterTypeHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	return WriteJSON(w, r, sdk.AvailableParameterType, http.StatusOK)
}
