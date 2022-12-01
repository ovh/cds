package api

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getVCSByIdentifier(ctx context.Context, projectKey string, vcsIdentifier string, opts ...gorpmapping.GetOptionFunc) (*sdk.VCSProject, error) {
	var vcsProject *sdk.VCSProject
	var err error
	if sdk.IsValidUUID(vcsIdentifier) {
		vcsProject, err = vcs.LoadVCSByID(ctx, api.mustDB(), projectKey, vcsIdentifier, opts...)
	} else {
		vcsProject, err = vcs.LoadVCSByProject(ctx, api.mustDB(), projectKey, vcsIdentifier, opts...)
	}
	if err != nil {
		return nil, err
	}
	return vcsProject, nil
}

func (api *API) postVCSProjectHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			project, err := project.Load(ctx, tx, pKey, project.LoadOptions.WithKeys)
			if err != nil {
				return sdk.WithStack(err)
			}

			var vcsProject sdk.VCSProject
			if err := service.UnmarshalRequest(ctx, req, &vcsProject); err != nil {
				return err
			}

			if err := vcsProject.Lint(*project); err != nil {
				return err
			}

			vcsProject.ProjectID = project.ID
			vcsProject.CreatedBy = getUserConsumer(ctx).GetUsername()

			if err := vcs.Insert(ctx, tx, &vcsProject); err != nil {
				return err
			}

			vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, pKey, vcsProject.Name)
			if err != nil {
				return err
			}

			if _, err := vcsClient.Repos(ctx); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			return service.WriteMarshal(w, req, vcsProject, http.StatusCreated)
		}
}

func (api *API) putVCSProjectHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]

			proj, err := project.Load(ctx, api.mustDB(), pKey, project.LoadOptions.WithKeys)
			if err != nil {
				return err
			}

			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}

			vcsOld, err := api.getVCSByIdentifier(ctx, pKey, vcsIdentifier)
			if err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			var vcsProject sdk.VCSProject
			if err := service.UnmarshalRequest(ctx, req, &vcsProject); err != nil {
				return err
			}

			vcsProject.ID = vcsOld.ID
			vcsProject.Created = vcsOld.Created
			vcsProject.CreatedBy = vcsOld.CreatedBy
			vcsProject.ProjectID = vcsOld.ProjectID

			if err := vcsProject.Lint(*proj); err != nil {
				return err
			}

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
	return service.RBAC(api.projectManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]

			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}

			vcsProject, err := api.getVCSByIdentifier(ctx, pKey, vcsIdentifier)
			if err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			project, err := project.Load(ctx, tx, pKey)
			if err != nil {
				return sdk.WithStack(err)
			}

			if err := vcs.Delete(tx, project.ID, vcsProject.Name); err != nil {
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
	return service.RBAC(api.projectRead),
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
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			vars := mux.Vars(r)
			pKey := vars["projectKey"]

			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}

			vcsProject, err := api.getVCSByIdentifier(ctx, pKey, vcsIdentifier)
			if err != nil {
				return err
			}

			vcsClear, err := vcs.LoadVCSByID(ctx, api.mustDB(), pKey, vcsProject.ID, gorpmapping.GetOptions.WithDecryption)
			if err != nil {
				return err
			}
			vcsProject.Auth.Username = vcsClear.Auth.Username
			vcsProject.Auth.SSHKeyName = vcsClear.Auth.SSHKeyName
			vcsProject.Auth.SSHUsername = vcsClear.Auth.SSHUsername
			vcsProject.Auth.SSHPort = vcsClear.Auth.SSHPort
			return service.WriteMarshal(w, r, vcsProject, http.StatusOK)
		}
}
