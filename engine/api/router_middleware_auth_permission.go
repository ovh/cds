package api

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/api/workflowtemplate"
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
	perms, err := loadPermissionsByGroupID(ctx, api.mustDB(), api.Cache, groupIDs...)
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

func (api *API) checkGroupPermissions(ctx context.Context, groupName string, permissionValue int, routeVars map[string]string) error {
	if groupName == "" {
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid given group name")
	}

	// check that group exists
	g, err := group.LoadByName(ctx, api.mustDB(), groupName, group.LoadOptions.WithMembers)
	if err != nil {
		return sdk.WrapError(err, "cannot get group for name %s", groupName)
	}

	log.Debug("groups> %v", g.Members)

	if permissionValue > permission.PermissionRead { // Only group administror or CDS administrator can update a group or its dependencies
		if !isGroupAdmin(ctx, g) && !isMaintainer(ctx) {
			return sdk.WithStack(sdk.ErrForbidden)
		}
	} else {
		if !isGroupMember(ctx, g) && !isMaintainer(ctx) { // Only group member of CDS administrator can get a group or its dependencies
			return sdk.WithStack(sdk.ErrForbidden)
		}
	}

	return nil
}

func (api *API) checkWorkerModelPermissions(ctx context.Context, modelName string, perm int, routeVars map[string]string) error {
	if modelName == "" {
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid worker model name")
	}

	g, err := group.LoadByName(ctx, api.mustDB(), routeVars["permGroupName"])
	if err != nil {
		return err
	}

	wm, err := workermodel.LoadByNameAndGroupID(api.mustDB(), modelName, g.ID)
	if err != nil {
		return err
	}
	if wm == nil {
		return sdk.WithStack(sdk.ErrNotFound)
	}

	return nil
}

func (api *API) checkActionPermissions(ctx context.Context, actionName string, perm int, routeVars map[string]string) error {
	if actionName == "" {
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid action name")
	}

	g, err := group.LoadByName(ctx, api.mustDB(), routeVars["permGroupName"])
	if err != nil {
		return err
	}

	a, err := action.LoadTypeDefaultByNameAndGroupID(ctx, api.mustDB(), actionName, g.ID)
	if err != nil {
		return err
	}
	if a == nil {
		return sdk.WithStack(sdk.ErrNotFound)
	}

	return nil
}

func (api *API) checkActionBuiltinPermissions(ctx context.Context, permActionBuiltinName string, perm int, routeVars map[string]string) error {
	return sdk.WrapError(sdk.ErrForbidden, "not authorized for action %s", permActionBuiltinName)
}

func (api *API) checkTemplateSlugPermissions(ctx context.Context, templateSlug string, permissionValue int, routeVars map[string]string) error {
	if templateSlug == "" {
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid workflow template slug")
	}

	g, err := group.LoadByName(ctx, api.mustDB(), routeVars["permGroupName"])
	if err != nil {
		return err
	}

	wt, err := workflowtemplate.LoadBySlugAndGroupID(ctx, api.mustDB(), templateSlug, g.ID)
	if err != nil {
		return err
	}
	if wt == nil {
		return sdk.WithStack(sdk.ErrNotFound)
	}

	return nil
}

// loadGroupPermissions retrieves all group memberships
func loadPermissionsByGroupID(ctx context.Context, db gorp.SqlExecutor, store cache.Store, groupID ...int64) (sdk.GroupPermissions, error) {
	var grpPerm sdk.GroupPermissions

	projectPermissions, err := project.FindPermissionByGroupID(ctx, db, groupID...)
	if err != nil {
		return grpPerm, err
	}
	grpPerm.Projects = projectPermissions
	return grpPerm, nil
}
