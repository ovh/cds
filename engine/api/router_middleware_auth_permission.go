package api

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/sdk"
)

// PermCheckFunc defines func call to check permission
type PermCheckFunc func(ctx context.Context, key string, perm int, routeVars map[string]string) error

func permissionFunc(api *API) map[string]PermCheckFunc {
	return map[string]PermCheckFunc{
		"permProjectKey":        api.checkProjectPermissions,
		"permWorkflowName":      api.checkWorkflowPermissions,
		"permGroupName":         api.checkGroupPermissions,
		"permModelID":           api.checkWorkerModelPermissions,
		"permActionName":        api.checkActionPermissions,
		"permActionBuiltinName": api.checkActionBuiltinPermissions,
		"permTemplateSlug":      api.checkTemplateSlugPermissions,
	}
}

func (api *API) checkPermission(ctx context.Context, routeVar map[string]string, permission int) error {
	for key, value := range routeVar {
		if permFunc, ok := permissionFunc(api)[key]; ok {
			if err := permFunc(ctx, value, permission, routeVar); err != nil {
				return err
			}
		}
	}
	return nil
}

func (api *API) checkProjectPermissions(ctx context.Context, key string, expectedPermissions int, routeVars map[string]string) error {
	groupIDs := sdk.GroupsToIDs(JWT(ctx).Groups)
	perms, err := loadPermissionsByGroupID(api.mustDB(), api.Cache, groupIDs...)
	if err != nil {
		return err
	}

	actualPermission, isGranted := perms.ProjectPermission(key)
	if isGranted && actualPermission >= expectedPermissions {
		return nil
	}

	return sdk.WrapError(sdk.ErrForbidden, "not authorized for project %s", key)
}

func (api *API) checkWorkflowPermissions(ctx context.Context, workflowName string, perm int, routeVars map[string]string) error {
	return sdk.WrapError(sdk.ErrForbidden, "not authorized for workflow %s, missing project key value", workflowName)
}

func (api *API) checkGroupPermissions(ctx context.Context, groupName string, perm int, routeVars map[string]string) error {
	return sdk.WrapError(sdk.ErrForbidden, "not authorized for group %s", groupName)
}

func (api *API) checkWorkerModelPermissions(ctx context.Context, key string, perm int, routeVars map[string]string) error {
	return sdk.WrapError(sdk.ErrForbidden, "not authorized for worker model %s", key)
}

func (api *API) checkActionPermissions(ctx context.Context, permActionName string, perm int, routeVars map[string]string) error {
	return sdk.WrapError(sdk.ErrForbidden, "not authorized for action %s", permActionName)
}

func (api *API) checkActionBuiltinPermissions(ctx context.Context, permActionBuiltinName string, perm int, routeVars map[string]string) error {
	return sdk.WrapError(sdk.ErrForbidden, "not authorized for action %s", permActionBuiltinName)
}

func (api *API) checkTemplateSlugPermissions(ctx context.Context, templateSlug string, permissionValue int, routeVars map[string]string) error {
	// try to get template for given path that match user's groups with/without admin grants
	groupName := routeVars["groupName"]

	if groupName == "" || templateSlug == "" {
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid given group or workflow template slug")
	}

	// check that group exists
	g, err := group.LoadGroup(api.mustDB(), groupName)
	if err != nil {
		return err
	}

	if permissionValue > permission.PermissionRead { // Only group administror or CDS administrator can update a template
		if !isGroupAdmin(ctx, g) && !isAdmin(ctx) {
			return sdk.WithStack(sdk.ErrForbidden)
		}
	} else {
		if !isGroupMember(ctx, g) && !isMaintainer(ctx) { // Only group member of CDS administrator can get a template
			return sdk.WithStack(sdk.ErrForbidden)
		}
	}

	wt, err := workflowtemplate.LoadBySlugAndGroupID(api.mustDB(), templateSlug, g.ID)
	if err != nil {
		return err
	}
	if wt == nil {
		return sdk.WithStack(sdk.ErrNotFound)
	}

	return nil
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
