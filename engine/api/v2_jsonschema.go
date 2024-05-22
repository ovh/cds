package api

import (
	"context"
	"fmt"
	"strings"

	"net/http"

	"github.com/gorilla/mux"
	"github.com/sguiheux/jsonschema"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/plugin"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getJsonSchemaHandler() ([]service.RbacChecker, service.Handler) {
	return nil,
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			t := vars["type"]

			u := getUserConsumer(ctx)

			var schema *jsonschema.Schema
			switch t {
			case sdk.EntityTypeWorkerModel:
				schema = sdk.GetWorkerModelJsonSchema()
			case sdk.EntityTypeAction, sdk.EntityTypeWorkflow, sdk.EntityTypeJob, sdk.EntityTypeWorkflowTemplate:
				actionNames, err := getActionNames(ctx, api.mustDB(), u)
				if err != nil {
					return err
				}
				regNames, err := getRegionNames(ctx, api.mustDB(), u)
				if err != nil {
					return err
				}
				wmNames, err := getWorkerModelNames(ctx, api.mustDB(), u)
				if err != nil {
					return err
				}

				switch t {
				case sdk.EntityTypeWorkflow:
					schema = sdk.GetWorkflowJsonSchema(actionNames, regNames, wmNames)
				case sdk.EntityTypeAction:
					schema = sdk.GetActionJsonSchema(actionNames)
				case sdk.EntityTypeJob:
					schema = sdk.GetJobJsonSchema(actionNames, regNames, wmNames)
				case sdk.EntityTypeWorkflowTemplate:
					schema = sdk.GetWorkflowTemplateJsonSchema()
				}
			}
			return service.WriteJSON(w, schema, http.StatusOK)
		}
}

func getWorkerModelNames(ctx context.Context, db gorp.SqlExecutor, u *sdk.AuthUserConsumer) ([]string, error) {
	if u == nil {
		return nil, nil
	}
	pKeys, err := rbac.LoadAllProjectKeysAllowed(ctx, db, sdk.ProjectRoleRead, u.AuthConsumerUser.AuthentifiedUserID)
	if err != nil {
		return nil, err
	}
	entities, err := entity.UnsafeLoadAllByTypeAndProjectKeys(ctx, db, sdk.EntityTypeWorkerModel, pKeys)
	if err != nil {
		return nil, err
	}
	wmNames := make([]string, 0, len(entities))
	for _, wm := range entities {
		wmNames = append(wmNames, fmt.Sprintf("%s/%s/%s/%s@%s", wm.ProjectKey, wm.VCSName, wm.RepoName, wm.Name, wm.Ref))
		shortRef := strings.TrimPrefix(strings.TrimPrefix(wm.Ref, sdk.GitRefBranchPrefix), sdk.GitRefTagPrefix)
		wmNames = append(wmNames, fmt.Sprintf("%s/%s/%s/%s@%s", wm.ProjectKey, wm.VCSName, wm.RepoName, wm.Name, shortRef))
		wmNames = append(wmNames, fmt.Sprintf("%s/%s/%s/%s", wm.ProjectKey, wm.VCSName, wm.RepoName, wm.Name))
	}
	return wmNames, nil
}

func getRegionNames(ctx context.Context, db gorp.SqlExecutor, u *sdk.AuthUserConsumer) ([]string, error) {
	if u == nil {
		return nil, nil
	}
	rbacRegion, err := rbac.LoadRegionIDsByRoleAndUserID(ctx, db, sdk.RegionRoleExecute, u.AuthConsumerUser.AuthentifiedUserID)
	if err != nil {
		return nil, err
	}
	regIDs := make([]string, 0, len(rbacRegion))
	for _, r := range rbacRegion {
		regIDs = append(regIDs, r.RegionID)
	}
	regs, err := region.LoadRegionByIDs(ctx, db, regIDs)
	if err != nil {
		return nil, err
	}
	regNames := make([]string, 0, len(regs))
	for _, r := range regs {
		regNames = append(regNames, r.Name)
	}
	return regNames, nil
}

func getActionNames(ctx context.Context, db gorp.SqlExecutor, u *sdk.AuthUserConsumer) ([]string, error) {
	// Load available action
	var actionNames []string
	if u != nil {
		keys, err := rbac.LoadAllProjectKeysAllowed(ctx, db, sdk.ProjectRoleRead, u.AuthConsumerUser.AuthentifiedUserID)
		if err != nil {
			return nil, err
		}
		actionFullNames, err := entity.UnsafeLoadAllByTypeAndProjectKeys(ctx, db, sdk.EntityTypeAction, keys)
		if err != nil {
			return nil, err
		}
		for _, an := range actionFullNames {
			shortRef := strings.TrimPrefix(strings.TrimPrefix(an.Ref, sdk.GitRefBranchPrefix), sdk.GitRefTagPrefix)
			actionNames = append(actionNames, fmt.Sprintf("%s/%s/%s/%s@%s", an.ProjectKey, an.VCSName, an.RepoName, an.Name, shortRef))
		}
	}
	// Load action plugin
	pls, err := plugin.LoadAllByType(ctx, db, sdk.GRPCPluginAction)
	if err != nil {
		return nil, err
	}
	for _, p := range pls {
		actionNames = append(actionNames, p.Name)
	}
	return actionNames, nil
}
