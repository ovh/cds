package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) postVCSProjectHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(rbac.ProjectManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			project, err := project.Load(ctx, tx, pKey)
			if err != nil {
				return sdk.WithStack(err)
			}

			var vcsProject sdk.VCSProject
			if err := service.UnmarshalRequest(ctx, req, &vcsProject); err != nil {
				return err
			}

			vcsProject.ProjectID = project.ID
			vcsProject.CreatedBy = getAPIConsumer(ctx).GetUsername()

			if err := vcs.Insert(ctx, tx, &vcsProject); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			return service.WriteMarshal(w, req, vcsProject, http.StatusCreated)
		}
}

func (api *API) getVCSProjectHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(rbac.ProjectManage),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			vars := mux.Vars(r)
			pKey := vars["projectKey"]

			vcsProjects, err := vcs.LoadAllVCSByProject(ctx, api.mustDB(), pKey)
			if err != nil {
				return sdk.WrapError(err, "unable to load vcs server on project %s", pKey)
			}

			return service.WriteJSON(w, vcsProjects, http.StatusOK)
		}
}
