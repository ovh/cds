package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/purge"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/glob"
	cdslog "github.com/ovh/cds/sdk/log"
)

func (api *API) getWorkflowRunRetentionSchemaHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			return service.WriteJSON(w, sdk.GetProjectRunRetentionJsonSchema(), http.StatusOK)
		}
}

func (api *API) postWorkflowRunRetentionDryRunHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			reportID := sdk.UUID()
			api.GoRoutines.Exec(context.Background(), "manual-dry-run-purge-"+pKey, func(ctx context.Context) {
				ctx = context.WithValue(ctx, cdslog.Project, pKey)
				time.Sleep(1 * time.Second)
				if err := purge.ApplyRunRetentionOnProject(ctx, api.mustDB(), api.Cache, pKey, purge.PurgeOption{DisabledDryRun: false, ReportID: reportID}); err != nil {
					log.ErrorWithStackTrace(ctx, err)
				}
			})

			return service.WriteJSON(w, sdk.StartRunResponse{ReportID: reportID}, http.StatusOK)
		}
}

func (api *API) postWorkflowRunRetentionStartHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			reportID := sdk.UUID()
			api.GoRoutines.Exec(context.Background(), "manual-purge-"+pKey, func(ctx context.Context) {
				ctx = context.WithValue(ctx, cdslog.Project, pKey)
				time.Sleep(1 * time.Second)
				if err := purge.ApplyRunRetentionOnProject(ctx, api.mustDB(), api.Cache, pKey, purge.PurgeOption{DisabledDryRun: true, ReportID: reportID}); err != nil {
					log.ErrorWithStackTrace(ctx, err)
				}
			})

			return service.WriteJSON(w, sdk.StartRunResponse{ReportID: reportID}, http.StatusOK)
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

			// Set default retention
			if projectRunRetention.Retentions.DefaultRetention.DurationInDays <= 0 {
				projectRunRetention.Retentions.DefaultRetention.DurationInDays = api.Config.WorkflowV2.WorkflowRunRetentionDefaultDays
			}

			if projectRunRetention.Retentions.DefaultRetention.DurationInDays > api.Config.WorkflowV2.WorkflowRunMaxRetention {
				projectRunRetention.Retentions.DefaultRetention.DurationInDays = api.Config.WorkflowV2.WorkflowRunMaxRetention
			}
			if projectRunRetention.Retentions.DefaultRetention.Count <= 0 {
				projectRunRetention.Retentions.DefaultRetention.Count = api.Config.WorkflowV2.WorkflowRunRetentionDefaultCount
			}

			// Check value for each workflow
			for i := range projectRunRetention.Retentions.WorkflowRetentions {
				r := &projectRunRetention.Retentions.WorkflowRetentions[i]
				if r.DefaultRetention == nil {
					continue
				}
				if r.DefaultRetention.DurationInDays <= 0 {
					r.DefaultRetention.DurationInDays = api.Config.WorkflowV2.WorkflowRunRetentionDefaultDays
				}
				if r.DefaultRetention.DurationInDays > api.Config.WorkflowV2.WorkflowRunMaxRetention {
					r.DefaultRetention.DurationInDays = api.Config.WorkflowV2.WorkflowRunMaxRetention
				}
				if r.DefaultRetention.Count <= 0 {
					r.DefaultRetention.Count = api.Config.WorkflowV2.WorkflowRunRetentionDefaultCount
				}
				for j := range r.Rules {
					g := &r.Rules[j]
					if g.DurationInDays <= 0 {
						g.DurationInDays = api.Config.WorkflowV2.WorkflowRunRetentionDefaultDays
					}
					if g.DurationInDays > api.Config.WorkflowV2.WorkflowRunMaxRetention {
						g.DurationInDays = api.Config.WorkflowV2.WorkflowRunMaxRetention
					}
					if g.Count <= 0 {
						g.Count = api.Config.WorkflowV2.WorkflowRunRetentionDefaultCount
					}
				}
			}

			// Check all git ref expressions
			defaultBranch := "refs/heads/my/branch"
			for _, wr := range projectRunRetention.Retentions.WorkflowRetentions {
				for _, data := range wr.Rules {
					if _, err := glob.New(data.GitRef).MatchString(defaultBranch); err != nil {
						return sdk.NewErrorFrom(sdk.ErrInvalidData, "wrong expression %q for workflow %q", data.GitRef, wr.Workflow)
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
			return service.WriteJSON(w, retentionDB, http.StatusOK)
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
