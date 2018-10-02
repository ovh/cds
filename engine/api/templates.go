package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/engine/service"
)

func (api *API) getTemplatesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, workflowtemplate.GetAll(), http.StatusOK)
	}
}

func (api *API) executeTemplateHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		tmpls := workflowtemplate.GetAll()

		id, err := requestVarInt(r, "id")
		if err != nil {
			return sdk.ErrNotFound
		}
		if len(tmpls) <= int(id) {
			return sdk.ErrNotFound
		}

		tmpl := tmpls[id]

		res, err := tmpl.Execute()
		if err != nil {
			return err
		}

		return service.WriteJSON(w, res, http.StatusOK)
	}
}
