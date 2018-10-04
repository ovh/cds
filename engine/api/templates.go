package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getTemplatesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		ts, err := workflowtemplate.GetAll(api.mustDB())
		if err != nil {
			return err
		}

		return service.WriteJSON(w, ts, http.StatusOK)
	}
}

func (api *API) postTemplateHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		t := &sdk.WorkflowTemplate{}
		if err := UnmarshalBody(r, t); err != nil {
			return sdk.WrapError(err, "Unmarshall body error")
		}

		if err := t.ValidateStruct(); err != nil {
			return err
		}

		if err := workflowtemplate.InsertWorkflow(api.mustDB(), t); err != nil {
			return err
		}

		return service.WriteJSON(w, t, http.StatusOK)
	}
}

func (api *API) executeTemplateHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, err := requestVarInt(r, "id")
		if err != nil {
			return sdk.ErrNotFound
		}
		t, err := workflowtemplate.GetByID(api.mustDB(), id)
		if err != nil {
			return err
		}
		if t == nil {
			return sdk.ErrNotFound
		}

		var req sdk.WorkflowTemplateRequest
		if err := UnmarshalBody(r, &req); err != nil {
			return sdk.WrapError(err, "Unmarshall body error")
		}

		if err := t.CheckParams(req); err != nil {
			return err
		}

		res, err := workflowtemplate.Execute(t, req)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, res, http.StatusOK)
	}
}
