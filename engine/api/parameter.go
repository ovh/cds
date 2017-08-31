package api

import (
	"net/http"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/sdk"
)

func getVariableTypeHandler(router *Router) Handler {
	return func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
		return WriteJSON(w, r, sdk.AvailableVariableType, http.StatusOK)
	}
}

func getParameterTypeHandler(router *Router) Handler {
	return func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
		return WriteJSON(w, r, sdk.AvailableParameterType, http.StatusOK)
	}
}
