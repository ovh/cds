package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getWorkflowRunRetentionSchemaHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			return service.WriteJSON(w, sdk.GetProjectRunRetentionJsonSchema(), http.StatusOK)
		}
}

func (api *API) putWorkflowRunRetentionHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			var projectRunRetention sdk.ProjectRunRetention
			if err := service.UnmarshalBody(req, &projectRunRetention); err != nil {
				return err
			}

			if projectRunRetention.Retentions.DefaultRetention.DurationInDays < 0 {
				projectRunRetention.Retentions.DefaultRetention.DurationInDays = api.Config.WorkflowV2.WorkflowRunRetention
			}

			if projectRunRetention.Retentions.DefaultRetention.DurationInDays > api.Config.WorkflowV2.WorkflowRunMaxRetention {
				projectRunRetention.Retentions.DefaultRetention.DurationInDays = api.Config.WorkflowV2.WorkflowRunMaxRetention
			}

			for i := range projectRunRetention.Retentions.WorkflowRetentions {
				r := &projectRunRetention.Retentions.WorkflowRetentions[i]
				if r.DefaultRetention == nil {
					continue
				}
				if r.DefaultRetention.DurationInDays < 0 {
					r.DefaultRetention.DurationInDays = api.Config.WorkflowV2.WorkflowRunRetention
				}
				if r.DefaultRetention.DurationInDays > api.Config.WorkflowV2.WorkflowRunMaxRetention {
					r.DefaultRetention.DurationInDays = api.Config.WorkflowV2.WorkflowRunMaxRetention
				}
				for j := range r.Rules {
					g := &r.Rules[j]
					if g.DurationInDays < 0 {
						g.DurationInDays = api.Config.WorkflowV2.WorkflowRunRetention
					}
					if g.DurationInDays > api.Config.WorkflowV2.WorkflowRunMaxRetention {
						g.DurationInDays = api.Config.WorkflowV2.WorkflowRunMaxRetention
					}
				}
			}

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			retentionDB, err := project.LoadRunRetentionByProjectKey(ctx, api.mustDB(), proj.Key)
			if err != nil {
				return err
			}

			retentionDB.Retentions = projectRunRetention.Retentions

			tx, err := api.mustDB().Begin()
			if err != nil {
				return err
			}
			defer tx.Rollback()

			if err := project.UpdateRunRetention(ctx, tx, retentionDB); err != nil {
				return err
			}
			if err := tx.Commit(); err != nil {
				return err
			}
			return service.WriteJSON(w, retention, http.StatusOK)
		}
}

func (api *API) getWorkflowRunRetentionHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			retention, err := project.LoadRunRetentionByProjectKey(ctx, api.mustDB(), proj.Key)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, retention, http.StatusOK)
		}
}
