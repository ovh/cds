package main

import (
	"net/http"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/sdk"
)

func getVariableTypeHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {
	WriteJSON(w, r, sdk.AvailableVariableType, http.StatusOK)
}

func getParameterTypeHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {
	WriteJSON(w, r, sdk.AvailableParameterType, http.StatusOK)
}
