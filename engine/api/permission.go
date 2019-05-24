package api

import (
	"context"
	"strconv"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// PermCheckFunc defines func call to check permission
type PermCheckFunc func(ctx context.Context, key string, perm int, routeVars map[string]string) error

func permissionFunc(api *API) map[string]PermCheckFunc {
	return map[string]PermCheckFunc{
		"permProjectKey":        api.checkProjectPermissions,
		"permWorkflowName":      api.checkWorkflowPermissions,
		"permGroupName":         api.checkGroupPermissions,
		"permModelID":           api.checkWorkerModelIDPermissions,
		"permModelName":         api.checkWorkerModelPermissions,
		"permActionName":        api.checkActionPermissions,
		"permActionBuiltinName": api.checkActionBuiltinPermissions,
	}
}

func getPermissionByMethod(method string, isExecution bool) int {
	switch method {
	case "POST":
		if isExecution {
			return permission.PermissionReadExecute
		}
		return permission.PermissionReadWriteExecute
	case "PUT":
		return permission.PermissionReadWriteExecute
	case "DELETE":
		return permission.PermissionReadWriteExecute
	default:
		return permission.PermissionRead
	}
}

func (api *API) deprecatedSetGroupsAndPermissionsFromGroupID(ctx context.Context, groupID int64) error {
	g, perm, err := loadPermissionsByGroupID(api.mustDB(), api.Cache, groupID)
	if err != nil {
		return sdk.WrapError(sdk.ErrUnauthorized, "deprecatedSetGroupsAndPermissionsFromGroupID> cannot load permissions: %s", err)
	}
	deprecatedGetUser(ctx).Permissions = perm
	deprecatedGetUser(ctx).Groups = append(deprecatedGetUser(ctx).Groups, g)
	return err
}

func (api *API) checkWorkerPermission(ctx context.Context, db gorp.SqlExecutor, rc *service.HandlerConfig, routeVar map[string]string) bool {
	if getWorker(ctx) == nil {
		log.Error("checkWorkerPermission> no worker in ctx")
		return false
	}

	idS, ok := routeVar["permID"]
	if !ok {
		return true
	}

	id, err := strconv.ParseInt(idS, 10, 64)
	if err != nil {
		log.Error("checkWorkerPermission> Unable to parse permID:%s err:%v", idS, err)
		return false
	}

	//IF it is POSTEXECUTE, it means that the job is must be taken by the worker
	if rc.Options["isExecution"] == "true" {
		k := cache.Key("workers", getWorker(ctx).ID, "perm", idS)
		if api.Cache.Get(k, &ok) {
			if ok {
				return ok
			}
		}

		runNodeJob, err := workflow.LoadNodeJobRun(db, api.Cache, id)
		if err != nil {
			log.Error("checkWorkerPermission> Unable to load job %d err:%v", id, err)
			return false
		}

		ok = runNodeJob.ID == getWorker(ctx).ActionBuildID
		api.Cache.SetWithTTL(k, ok, 60*15)
		if !ok {
			log.Error("checkWorkerPermission> actionBuildID:%v runNodeJob.ID:%v", getWorker(ctx).ActionBuildID, runNodeJob.ID)
		}
		return ok
	}
	return true
}

func (api *API) checkPermission(ctx context.Context, routeVar map[string]string, permission int) error {
	// FIXME to remove with new auth, by pass only used for workers
	for _, g := range deprecatedGetUser(ctx).Groups {
		if group.SharedInfraGroup != nil && g.Name == group.SharedInfraGroup.Name {
			return nil
		}
	}

	for key, value := range routeVar {
		if permFunc, ok := permissionFunc(api)[key]; ok {
			if err := permFunc(ctx, value, permission, routeVar); err != nil {
				return err
			}
		}
	}

	return nil
}

func (api *API) checkProjectPermissions(ctx context.Context, projectKey string, perm int, routeVars map[string]string) error {
	if permission.PermissionReadExecute == perm && getService(ctx) != nil {
		return nil
	}

	if deprecatedGetUser(ctx).Permissions.ProjectsPerm[projectKey] >= perm {
		return nil
	}

	return sdk.WrapError(sdk.ErrForbidden, "user not authorized for project %s", projectKey)
}

func (api *API) checkWorkflowPermissions(ctx context.Context, workflowName string, perm int, routeVars map[string]string) error {
	if projectKey, ok := routeVars["key"]; ok {
		switch perm {
		case permission.PermissionRead:
			// If need read permission, just check project read permission
			if checkProjectReadPermission(ctx, projectKey) {
				return nil
			}
			return sdk.WrapError(sdk.ErrForbidden, "user not authorized for workflow %s", workflowName)
		default:
			wPerm, has := deprecatedGetUser(ctx).Permissions.WorkflowsPerm[sdk.UserPermissionKey(projectKey, workflowName)]
			if !has {
				return sdk.WithStack(sdk.ErrNotFound)
			}
			if wPerm >= perm {
				return nil
			}
			return sdk.WrapError(sdk.ErrForbidden, "user not authorized for workflow %s", workflowName)
		}
	}

	return sdk.WrapError(sdk.ErrForbidden, "user not authorized for workflow %s, missing project key value", workflowName)
}

func checkProjectReadPermission(ctx context.Context, projectKey string) bool {
	return deprecatedGetUser(ctx).Permissions.ProjectsPerm[projectKey] >= permission.PermissionRead
}

func (api *API) checkGroupPermissions(ctx context.Context, groupName string, permissionValue int, routeVar map[string]string) error {
	for _, g := range deprecatedGetUser(ctx).Groups {
		if g.Name == groupName {
			if permissionValue == permission.PermissionRead {
				return nil
			}

			for i := range g.Admins {
				if g.Admins[i].ID == deprecatedGetUser(ctx).ID {
					return nil
				}
			}
		}
	}

	return sdk.WrapError(sdk.ErrForbidden, "user not authorized for group %s", groupName)
}

func (api *API) checkWorkerModelPermissions(ctx context.Context, modelName string, permissionValue int, routeVars map[string]string) error {
	// try to get worker model for given path that match user's groups with/without admin grants
	groupName := routeVars["groupName"]

	if groupName == "" || modelName == "" {
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid given group or worker model name")
	}

	u := deprecatedGetUser(ctx)

	// check that group exists
	g, err := group.LoadGroup(api.mustDB(), groupName)
	if err != nil {
		return err
	}

	if permissionValue > permission.PermissionRead {
		if err := group.CheckUserIsGroupAdmin(g, u); err != nil {
			return err
		}
	} else {
		if err := group.CheckUserIsGroupMember(g, u); err != nil {
			return err
		}
	}

	m, err := workermodel.LoadByNameAndGroupID(api.mustDB(), modelName, g.ID)
	if err != nil {
		return err
	}
	if m == nil {
		return sdk.WithStack(sdk.ErrNoWorkerModel)
	}

	return nil
}

// This will works only for hatchery.
func (api *API) checkWorkerModelIDPermissions(ctx context.Context, modelID string, permissionValue int, routeVar map[string]string) error {
	id, err := strconv.ParseInt(modelID, 10, 64)
	if err != nil {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given worker model id %s", modelID)
	}

	m, err := workermodel.LoadByID(api.mustDB(), id)
	if err != nil {
		return err
	}

	h := getHatchery(ctx)
	if h == nil || h.GroupID == nil {
		return sdk.NewErrorFrom(sdk.ErrForbidden, "user not authorized for worker model %s", modelID)
	}

	if *h.GroupID == group.SharedInfraGroup.ID || m.GroupID == *h.GroupID {
		return nil
	}

	return sdk.NewErrorFrom(sdk.ErrForbidden, "user not authorized for worker model %s", modelID)
}

func (api *API) checkActionPermissions(ctx context.Context, actionName string, permissionValue int, routeVars map[string]string) error {
	// try to get action for given path that match user's groups with/without admin grants
	groupName := routeVars["groupName"]

	if groupName == "" || actionName == "" {
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid given group or action name")
	}

	u := deprecatedGetUser(ctx)

	// check that group exists
	g, err := group.LoadGroup(api.mustDB(), groupName)
	if err != nil {
		return err
	}

	if permissionValue > permission.PermissionRead {
		if err := group.CheckUserIsGroupAdmin(g, u); err != nil {
			return err
		}
	} else {
		if err := group.CheckUserIsGroupMember(g, u); err != nil {
			return err
		}
	}

	a, err := action.LoadTypeDefaultByNameAndGroupID(api.mustDB(), actionName, g.ID)
	if err != nil {
		return err
	}
	if a == nil {
		return sdk.WithStack(sdk.ErrNoAction)
	}

	return nil
}

func (api *API) checkActionBuiltinPermissions(ctx context.Context, actionName string, permissionValue int, routeVars map[string]string) error {
	// try to get action for given name
	if actionName == "" {
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid given action name")
	}

	a, err := action.LoadByTypesAndName(api.mustDB(), []string{sdk.BuiltinAction, sdk.PluginAction}, actionName)
	if err != nil {
		return err
	}
	if a == nil {
		return sdk.WithStack(sdk.ErrNoAction)
	}

	return nil
}
