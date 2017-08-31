package api

import (
	"net/http"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/sdk"
)

// ConfigUserHandler return url of CDS UI
func ConfigUserHandler(router *Router) Handler {
	return func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
		return WriteJSON(w, r, map[string]string{sdk.ConfigURLUIKey: router.Cfg.URL.UI}, http.StatusOK)
	}
}
