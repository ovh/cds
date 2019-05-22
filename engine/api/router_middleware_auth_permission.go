package api

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
)

// PermCheckFunc defines func call to check permission
type PermCheckFunc func(key string, perm int, routeVars map[string]string, groupID ...int64) error

func permissionFunc(api *API) map[string]PermCheckFunc {
	return map[string]PermCheckFunc{
		"permProjectKey":        api.checkProjectPermissions,
		"permWorkflowName":      api.checkWorkflowPermissions,
		"permGroupName":         api.checkGroupPermissions,
		"permModelID":           api.checkWorkerModelPermissions,
		"permActionName":        api.checkActionPermissions,
		"permActionBuiltinName": api.checkActionBuiltinPermissions,
	}
}

func (api *API) checkPermission(ctx context.Context, routeVar map[string]string, permission int) error {
	token := JWT(ctx)
	groupIDs := sdk.GroupsToIDs(token.Groups)
	for key, value := range routeVar {
		if permFunc, ok := permissionFunc(api)[key]; ok {
			if err := permFunc(value, permission, routeVar, groupIDs...); err != nil {
				return err
			}
		}
	}
	return nil
}

func (api *API) checkProjectPermissions(key string, expectedPermissions int, routeVars map[string]string, groupID ...int64) error {
	perms, err := loadPermissionsByGroupID(api.mustDB(), api.Cache, groupID...)
	if err != nil {
		return err
	}

	actualPermission, isGranted := perms.ProjectPermission(key)
	if isGranted && actualPermission >= expectedPermissions {
		return nil
	}

	return sdk.WrapError(sdk.ErrForbidden, "not authorized for project %s", key)
}

func (api *API) checkWorkflowPermissions(workflowName string, perm int, routeVars map[string]string, groupID ...int64) error {
	return sdk.WrapError(sdk.ErrForbidden, "not authorized for workflow %s, missing project key value", workflowName)
}

func (api *API) checkGroupPermissions(groupName string, perm int, routeVars map[string]string, groupID ...int64) error {
	return sdk.WrapError(sdk.ErrForbidden, "not authorized for group %s", groupName)
}

func (api *API) checkWorkerModelPermissions(key string, perm int, routeVars map[string]string, groupID ...int64) error {
	return sdk.WrapError(sdk.ErrForbidden, "not authorized for worker model %s", key)
}

func (api *API) checkActionPermissions(permActionName string, perm int, routeVars map[string]string, groupID ...int64) error {
	return sdk.WrapError(sdk.ErrForbidden, "not authorized for action %s", permActionName)
}

func (api *API) checkActionBuiltinPermissions(permActionBuiltinName string, perm int, routeVars map[string]string, groupID ...int64) error {
	return sdk.WrapError(sdk.ErrForbidden, "not authorized for action %s", permActionBuiltinName)
}

// loadGroupPermissions retrieves all group memberships
func loadPermissionsByGroupID(db gorp.SqlExecutor, store cache.Store, groupID ...int64) (sdk.GroupPermissions, error) {
	var grpPerm sdk.GroupPermissions

	projectPermissions, err := project.FindPermissionByGroupID(db, groupID...)
	if err != nil {
		return grpPerm, err
	}
	grpPerm.Projects = projectPermissions
	return grpPerm, nil
}
