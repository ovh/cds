package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
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

func (api *API) putVCSProjectHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(rbac.ProjectManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			vcsName := vars["vcsProjectName"]

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			var vcsProject sdk.VCSProject
			if err := service.UnmarshalRequest(ctx, req, &vcsProject); err != nil {
				return err
			}

			vcsOld, err := vcs.LoadVCSByProject(ctx, tx, pKey, vcsName, gorpmapping.GetOptions.WithDecryption)
			if err != nil {
				return err
			}

			vcsProject.ID = vcsOld.ID
			vcsProject.Created = vcsOld.Created
			vcsProject.CreatedBy = vcsOld.CreatedBy

			if err := vcs.Update(ctx, tx, &vcsProject); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			return service.WriteMarshal(w, req, vcsProject, http.StatusCreated)
		}
}

func (api *API) deleteVCSProjectHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(rbac.ProjectManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			vcsName := vars["vcsProjectName"]

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			project, err := project.Load(ctx, tx, pKey)
			if err != nil {
				return sdk.WithStack(err)
			}

			vcsOld, err := vcs.LoadVCSByProject(context.Background(), tx, project.Key, vcsName, gorpmapping.GetOptions.WithDecryption)
			if err != nil {
				return err
			}

			if err := vcs.Delete(tx, project.ID, vcsOld.Name); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			return nil
		}
}

// getVCSProjectAllHandler returns list of vcs of one project key
func (api *API) getVCSProjectAllHandler() ([]service.RbacChecker, service.Handler) {
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

func (api *API) getVCSProjectHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(rbac.ProjectManage),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			vars := mux.Vars(r)
			pKey := vars["projectKey"]
			vcsProjectName := vars["vcsProjectName"]

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			vcsProject, err := vcs.LoadVCSByProject(context.Background(), tx, pKey, vcsProjectName, gorpmapping.GetOptions.WithDecryption)
			if err != nil {
				return err
			}

			return service.WriteMarshal(w, r, vcsProject, http.StatusOK)
		}
}
