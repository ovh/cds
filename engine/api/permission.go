package api

import (
	"context"
	"strconv"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// PermCheckFunc defines func call to check permission
type PermCheckFunc func(ctx context.Context, key string, permission int, routeVar map[string]string) bool

func permissionFunc(api *API) map[string]PermCheckFunc {
	return map[string]PermCheckFunc{
		"permProjectKey":   api.checkProjectPermissions,
		"permWorkflowName": api.checkWorkflowPermissions,
		"permGroupName":    api.checkGroupPermissions,
		"permModelID":      api.checkWorkerModelPermissions,
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
		return ok
	}
	return true
}

func (api *API) checkPermission(ctx context.Context, routeVar map[string]string, permission int) bool {
	for _, g := range deprecatedGetUser(ctx).Groups {
		if group.SharedInfraGroup != nil && g.Name == group.SharedInfraGroup.Name {
			return true
		}
	}

	permissionOk := true
	for key, value := range routeVar {
		if permFunc, ok := permissionFunc(api)[key]; ok {
			permissionOk = permFunc(ctx, value, permission, routeVar)
			if !permissionOk {
				return permissionOk
			}
		}
	}
	return permissionOk
}

func (api *API) checkProjectPermissions(ctx context.Context, projectKey string, perm int, routeVar map[string]string) bool {
	if permission.PermissionReadExecute == perm && getService(ctx) != nil {
		return true
	}
	return deprecatedGetUser(ctx).Permissions.ProjectsPerm[projectKey] >= perm
}

func (api *API) checkWorkflowPermissions(ctx context.Context, workflowName string, perm int, routeVar map[string]string) bool {
	if projectKey, ok := routeVar["key"]; ok {
		// If need read permission, just check project read permission
		switch perm {
		case permission.PermissionRead:
			return checkProjectReadPermission(ctx, projectKey)
		default:
			return deprecatedGetUser(ctx).Permissions.WorkflowsPerm[sdk.UserPermissionKey(projectKey, workflowName)] >= perm
		}
	} else {
		log.Warning("Wrong route configuration. need key parameter")
	}
	return false
}

func checkProjectReadPermission(ctx context.Context, projectKey string) bool {
	return deprecatedGetUser(ctx).Permissions.ProjectsPerm[projectKey] >= permission.PermissionRead
}

func (api *API) checkGroupPermissions(ctx context.Context, groupName string, permissionValue int, routeVar map[string]string) bool {
	for _, g := range deprecatedGetUser(ctx).Groups {
		if g.Name == groupName {

			if permissionValue == permission.PermissionRead {
				return true
			}

			for i := range g.Admins {
				if g.Admins[i].ID == deprecatedGetUser(ctx).ID {
					return true
				}
			}
		}
	}

	return false
}

func (api *API) checkWorkerModelPermissions(ctx context.Context, modelID string, permissionValue int, routeVar map[string]string) bool {
	id, err := strconv.ParseInt(modelID, 10, 64)
	if err != nil {
		log.Warning("checkWorkerModelPermissions> modelID is not an integer: %s", err)
		return false
	}

	m, err := worker.LoadWorkerModelByID(api.mustDB(), id)
	if err != nil {
		log.Warning("checkWorkerModelPermissions> unable to load model by id %s: %s", modelID, err)
		return false
	}

	h := getHatchery(ctx)
	if h != nil && h.GroupID != nil {
		return *h.GroupID == group.SharedInfraGroup.ID || m.GroupID == *h.GroupID
	}
	return api.checkWorkerModelPermissionsByUser(m, deprecatedGetUser(ctx), permissionValue)
}

func (api *API) checkWorkerModelPermissionsByUser(m *sdk.Model, u *sdk.User, permissionValue int) bool {
	if u.Admin {
		return true
	}

	for _, g := range u.Groups {
		if g.ID == m.GroupID {
			for _, a := range g.Admins {
				if a.ID == u.ID {
					return true
				}
			}

			if permissionValue == permission.PermissionRead {
				return true
			}
		}
	}
	return false
}
