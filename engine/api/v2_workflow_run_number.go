package api

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

type workflowRunNumberTarget struct {
	projectKey   string
	vcsServerID  string
	repo         *sdk.ProjectRepository
	workflowName string
}

// loadWorkflowRunNumberTarget resolves the VCS and repository of the route and checks the workflow exists in it.
func (api *API) loadWorkflowRunNumberTarget(ctx context.Context, req *http.Request) (*workflowRunNumberTarget, error) {
	vars := mux.Vars(req)
	pKey := vars["projectKey"]
	vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
	if err != nil {
		return nil, sdk.NewError(sdk.ErrWrongRequest, err)
	}
	repositoryIdentifier, err := url.PathUnescape(vars["repositoryIdentifier"])
	if err != nil {
		return nil, sdk.NewError(sdk.ErrWrongRequest, err)
	}
	workflowName := vars["workflow"]

	vcsProject, err := api.getVCSByIdentifier(ctx, pKey, vcsIdentifier)
	if err != nil {
		return nil, err
	}
	repo, err := api.getRepositoryByIdentifier(ctx, vcsProject.ID, repositoryIdentifier)
	if err != nil {
		return nil, err
	}
	exists, err := entity.ExistsByTypeAndName(api.mustDB(), repo.ID, sdk.EntityTypeWorkflow, workflowName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, sdk.NewErrorFrom(sdk.ErrNotFound, "workflow %s not found in repository %s", workflowName, repo.Name)
	}
	return &workflowRunNumberTarget{projectKey: pKey, vcsServerID: vcsProject.ID, repo: repo, workflowName: workflowName}, nil
}

func (api *API) getWorkflowRunNumberHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			target, err := api.loadWorkflowRunNumberTarget(ctx, req)
			if err != nil {
				return err
			}
			value, err := workflow_v2.LoadWorkflowRunNumber(api.mustDB(), target.repo.ID, target.workflowName)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, sdk.V2WorkflowRunNumber{Value: value}, http.StatusOK)
		}
}

func (api *API) putWorkflowRunNumberHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			target, err := api.loadWorkflowRunNumberTarget(ctx, req)
			if err != nil {
				return err
			}

			var runNumber sdk.V2WorkflowRunNumber
			if err := service.UnmarshalBody(req, &runNumber); err != nil {
				return err
			}
			if runNumber.Value < 0 {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "run number must be positive, got %d", runNumber.Value)
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			if err := workflow_v2.LockWorkflowRunNumber(tx, target.repo.ID, target.workflowName); err != nil {
				return err
			}
			// Run numbers are unique per workflow: below the highest existing one, the next runs would collide.
			maxRunNumber, err := workflow_v2.LoadMaxRunNumber(tx, target.projectKey, target.vcsServerID, target.repo.ID, target.workflowName)
			if err != nil {
				return err
			}
			if runNumber.Value < maxRunNumber {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "run number %d is lower than the highest existing run number %d", runNumber.Value, maxRunNumber)
			}
			if err := workflow_v2.SetWorkflowRunNumber(tx, target.repo.ID, target.workflowName, runNumber.Value); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}
			return service.WriteJSON(w, runNumber, http.StatusOK)
		}
}
