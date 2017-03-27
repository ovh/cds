package main

import (
	"net/http"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/sdk"
)

// ConfigUserHandler return url of CDS UI
func ConfigUserHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return WriteJSON(w, r, map[string]string{sdk.ConfigURLUIKey: baseURL}, http.StatusOK)
}
