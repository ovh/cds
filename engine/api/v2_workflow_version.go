package api

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) deleteWorkflowVersionHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.workflowTrigger),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WrapError(sdk.ErrForbidden, "no user consumer")
			}

			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			repositoryIdentifier, err := url.PathUnescape(vars["repositoryIdentifier"])
			if err != nil {
				return sdk.WithStack(err)
			}
			workflowName := vars["workflow"]
			version := vars["version"]

			workflowVersion, err := workflow_v2.LoadWorkflowVersion(ctx, api.mustDB(), pKey, vcsIdentifier, repositoryIdentifier, workflowName, version)
			if err != nil {
				return err
			}
			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint
			if err := workflow_v2.DeleteWorkflowVersion(ctx, tx, workflowVersion); err != nil {
				return err
			}
			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}
			return service.WriteJSON(w, nil, http.StatusOK)

		}
}

func (api *API) getWorkflowVersionHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.workflowTrigger),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WrapError(sdk.ErrForbidden, "no user consumer")
			}

			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			repositoryIdentifier, err := url.PathUnescape(vars["repositoryIdentifier"])
			if err != nil {
				return sdk.WithStack(err)
			}
			workflowName := vars["workflow"]
			version := vars["version"]

			workflowVersion, err := workflow_v2.LoadWorkflowVersion(ctx, api.mustDB(), pKey, vcsIdentifier, repositoryIdentifier, workflowName, version)
			if err != nil {
				return err
			}

			return service.WriteJSON(w, workflowVersion, http.StatusOK)

		}
}

func (api *API) getWorkflowVersionsHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.workflowTrigger),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WrapError(sdk.ErrForbidden, "no user consumer")
			}

			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			repositoryIdentifier, err := url.PathUnescape(vars["repositoryIdentifier"])
			if err != nil {
				return sdk.WithStack(err)
			}
			workflowName := vars["workflow"]

			versions, err := workflow_v2.LoadAllVerionsByWorkflow(ctx, api.mustDB(), pKey, vcsIdentifier, repositoryIdentifier, workflowName)
			if err != nil {
				return err
			}

			return service.WriteJSON(w, versions, http.StatusOK)
		}
}
