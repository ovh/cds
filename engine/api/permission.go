package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// PermCheckFunc defines func call to check permission
type PermCheckFunc func(ctx context.Context, key string, permission int, routeVar map[string]string) bool

func permissionFunc(api *API) map[string]PermCheckFunc {
	return map[string]PermCheckFunc{
		"permProjectKey":      api.checkProjectPermissions,
		"permPipelineKey":     api.checkPipelinePermissions,
		"permApplicationName": api.checkApplicationPermissions,
		"permWorkflowName":    api.checkWorkflowPermissions,
		"permGroupName":       api.checkGroupPermissions,
		"permActionName":      api.checkActionPermissions,
		"permEnvironmentName": api.checkEnvironmentPermissions,
		"permModelID":         api.checkWorkerModelPermissions,
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

func (api *API) deletePermissionMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *HandlerConfig) (context.Context, error) {
	if req.Method == "POST" || req.Method == "PUT" || req.Method == "DELETE" {
		api.deleteUserPermissionCache(ctx, api.Cache)
	}
	return ctx, nil
}

func (api *API) authMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *HandlerConfig) (context.Context, error) {
	headers := req.Header

	// Check Token
	if h, ok := rc.Options["token"]; ok {
		headerSplitted := strings.Split(h, ":")
		receivedValue := req.Header.Get(headerSplitted[0])
		if receivedValue != headerSplitted[1] {
			return ctx, sdk.WrapError(sdk.ErrUnauthorized, "Router> Authorization denied on %s %s for %s", req.Method, req.URL, req.RemoteAddr)
		}
	}

	//Check Authentication
	if rc.Options["auth"] == "true" {
		switch headers.Get("User-Agent") {
		case sdk.HatcheryAgent:
			var err error
			ctx, err = auth.CheckHatcheryAuth(ctx, api.mustDB(), headers)
			if err != nil {
				return ctx, sdk.WrapError(sdk.ErrUnauthorized, "Router> Authorization denied on %s %s for %s agent %s : %s", req.Method, req.URL, req.RemoteAddr, getAgent(req), err)
			}
		case sdk.WorkerAgent:
			var err error
			ctx, err = auth.CheckWorkerAuth(ctx, api.mustDB(), api.Cache, headers)
			if err != nil {
				return ctx, sdk.WrapError(sdk.ErrUnauthorized, "Router> Authorization denied on %s %s for %s agent %s : %s", req.Method, req.URL, req.RemoteAddr, getAgent(req), err)
			}
		case sdk.ServiceAgent:
			var err error
			ctx, err = auth.CheckServiceAuth(ctx, api.mustDB(), api.Cache, headers)
			if err != nil {
				return ctx, sdk.WrapError(sdk.ErrUnauthorized, "Router> Authorization denied on %s %s for %s agent %s : %s", req.Method, req.URL, req.RemoteAddr, getAgent(req), err)
			}
		default:
			var err error
			ctx, err = api.Router.AuthDriver.CheckAuth(ctx, w, req)
			if err != nil {
				return ctx, sdk.WrapError(sdk.ErrUnauthorized, "Router> Authorization denied on %s %s for %s agent %s : %s", req.Method, req.URL, req.RemoteAddr, getAgent(req), err)
			}
		}
	}

	//Get the permission for either the hatchery, the worker or the user
	switch {
	case getHatchery(ctx) != nil:
		g, perm, err := loadPermissionsByGroupID(api.mustDB(), api.Cache, getHatchery(ctx).GroupID)
		if err != nil {
			return ctx, sdk.WrapError(sdk.ErrUnauthorized, "Router> cannot load group permissions for GroupID %d err:%s", getHatchery(ctx).GroupID, err)
		}
		getUser(ctx).Permissions = perm
		getUser(ctx).Groups = append(getUser(ctx).Groups, g)

	case getWorker(ctx) != nil:
		//Refresh the worker
		workerCtx := getWorker(ctx)
		if err := worker.RefreshWorker(api.mustDB(), workerCtx); err != nil {
			return ctx, sdk.WrapError(err, "Router> Unable to refresh worker")
		}

		if workerCtx.ModelID != 0 {
			// worker have a model, load model, then load model's group
			m, err := worker.LoadWorkerModelByID(api.mustDB(), workerCtx.ModelID)
			if err != nil {
				return ctx, sdk.WrapError(sdk.ErrUnauthorized, "Router> cannot load worker: %s - name:%s modelID:%d", err, workerCtx.Name, workerCtx.ModelID)
			}

			if m.GroupID == group.SharedInfraGroup.ID {
				// it's a shared.infra model, load group from token only: workerCtx.GroupID
				if err := api.setGroupsAndPermissionsFromGroupID(ctx, workerCtx.GroupID); err != nil {
					return ctx, sdk.WrapError(sdk.ErrUnauthorized, "Router> model shared.infra:%d", m.GroupID)
				}
			} else {
				// this model is not attached to shared.infra group, load group with m.GroupID
				if err := api.setGroupsAndPermissionsFromGroupID(ctx, m.GroupID); err != nil {
					return ctx, sdk.WrapError(sdk.ErrUnauthorized, "Router> model not shared.infra:%d", m.GroupID)
				}
			}
		} else {
			// worker does not have a model, take group from token only
			if err := api.setGroupsAndPermissionsFromGroupID(ctx, workerCtx.GroupID); err != nil {
				return ctx, sdk.WrapError(sdk.ErrUnauthorized, "Router> no model, worker not shared.infra:%d", workerCtx.GroupID)
			}
		}
	case getUser(ctx) != nil:
		if err := loadUserPermissions(api.mustDB(), api.Cache, getUser(ctx)); err != nil {
			return ctx, sdk.WrapError(sdk.ErrUnauthorized, "Router> Unable to load user %s permission: %s", getUser(ctx).ID, err)
		}
	}

	if rc.Options["auth"] != "true" {
		return ctx, nil
	}

	if getUser(ctx) == nil {
		return ctx, sdk.WrapError(sdk.ErrUnauthorized, "Router> Unable to find connected user")
	}

	if rc.Options["needHatchery"] == "true" && getHatchery(ctx) != nil {
		return ctx, nil
	}

	if rc.Options["needWorker"] == "true" {
		permissionOk := api.checkWorkerPermission(ctx, api.mustDB(), rc, mux.Vars(req))
		if !permissionOk {
			return ctx, sdk.WrapError(sdk.ErrForbidden, "Router> Worker not authorized")
		}
		return ctx, nil
	}

	if rc.Options["allowServices"] == "true" && getService(ctx) != nil {
		return ctx, nil
	}

	if getUser(ctx).Admin {
		return ctx, nil
	}

	if rc.Options["needAdmin"] != "true" {
		permissionOk := api.checkPermission(ctx, mux.Vars(req), getPermissionByMethod(req.Method, rc.Options["isExecution"] == "true"))
		if !permissionOk {
			return ctx, sdk.WrapError(sdk.ErrForbidden, "Router> User not authorized")
		}
	} else {
		return ctx, sdk.WrapError(sdk.ErrForbidden, "Router> User not authorized (needAdmin)")
	}

	if rc.Options["needUsernameOrAdmin"] == "true" && getUser(ctx).Username != mux.Vars(req)["username"] {
		// get / update / delete user -> for admin or current user
		// if not admin and currentUser != username in request -> ko
		return ctx, sdk.WrapError(sdk.ErrForbidden, "Router> User not authorized on this resource")
	}

	return ctx, nil

}

func (api *API) setGroupsAndPermissionsFromGroupID(ctx context.Context, groupID int64) error {
	g, perm, err := loadPermissionsByGroupID(api.mustDB(), api.Cache, groupID)
	if err != nil {
		return sdk.WrapError(sdk.ErrUnauthorized, "setGroupsAndPermissionsFromGroupID> cannot load permissions: %s", err)
	}
	getUser(ctx).Permissions = perm
	getUser(ctx).Groups = append(getUser(ctx).Groups, g)
	return err
}

func (api *API) checkWorkerPermission(ctx context.Context, db gorp.SqlExecutor, rc *HandlerConfig, routeVar map[string]string) bool {
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
			return ok
		}

		node, err := workflow.LoadNodeJobRun(db, api.Cache, id)
		if err != nil {
			log.Error("checkWorkerPermission> Unable to load job %d err:%v", id, err)
			return false
		}
		ok = node.Job.WorkerName == getWorker(ctx).Name && node.Job.WorkerID == getWorker(ctx).ID
		api.Cache.SetWithTTL(k, ok, 60*15)
		return ok
	}
	return true
}

func (api *API) checkPermission(ctx context.Context, routeVar map[string]string, permission int) bool {
	for _, g := range getUser(ctx).Groups {
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
	return getUser(ctx).Permissions.ProjectsPerm[projectKey] >= perm
}

func (api *API) checkPipelinePermissions(ctx context.Context, pipelineName string, perm int, routeVar map[string]string) bool {
	// Check if param key exist
	if projectKey, ok := routeVar["key"]; ok {
		switch perm {
		case permission.PermissionRead:
			return checkProjectReadPermission(ctx, projectKey)
		default:
			return getUser(ctx).Permissions.PipelinesPerm[sdk.UserPermissionKey(projectKey, pipelineName)] >= perm
		}
	} else {
		log.Warning("Wrong route configuration. need key parameter")
	}
	return false
}

func (api *API) checkEnvironmentPermissions(ctx context.Context, envName string, perm int, routeVar map[string]string) bool {
	// Check if param key exist
	if projectKey, ok := routeVar["key"]; ok {
		switch perm {
		case permission.PermissionRead:
			return checkProjectReadPermission(ctx, projectKey)
		default:
			return getUser(ctx).Permissions.EnvironmentsPerm[sdk.UserPermissionKey(projectKey, envName)] >= perm
		}
	} else {
		log.Warning("Wrong route configuration. need key parameter")
	}
	return false
}

func (api *API) checkWorkflowPermissions(ctx context.Context, workflowName string, perm int, routeVar map[string]string) bool {
	if projectKey, ok := routeVar["key"]; ok {
		// If need read permission, just check project read permission
		switch perm {
		case permission.PermissionRead:
			return checkProjectReadPermission(ctx, projectKey)
		default:
			return getUser(ctx).Permissions.WorkflowsPerm[sdk.UserPermissionKey(projectKey, workflowName)] >= perm
		}
	} else {
		log.Warning("Wrong route configuration. need key parameter")
	}
	return false
}

func checkProjectReadPermission(ctx context.Context, projectKey string) bool {
	return getUser(ctx).Permissions.ProjectsPerm[projectKey] >= permission.PermissionRead
}

func (api *API) checkApplicationPermissions(ctx context.Context, applicationName string, perm int, routeVar map[string]string) bool {
	// Check if param key exist
	if projectKey, ok := routeVar["key"]; ok {
		switch perm {
		case permission.PermissionRead:
			return checkProjectReadPermission(ctx, projectKey)
		default:
			return getUser(ctx).Permissions.ApplicationsPerm[sdk.UserPermissionKey(projectKey, applicationName)] >= perm
		}
	} else {
		log.Warning("Wrong route configuration. need key parameter")
	}
	return false
}

func (api *API) checkGroupPermissions(ctx context.Context, groupName string, permissionValue int, routeVar map[string]string) bool {
	for _, g := range getUser(ctx).Groups {
		if g.Name == groupName {

			if permissionValue == permission.PermissionRead {
				return true
			}

			for i := range g.Admins {
				if g.Admins[i].ID == getUser(ctx).ID {
					return true
				}
			}
		}
	}

	return false
}

func (api *API) checkActionPermissions(ctx context.Context, groupName string, permissionValue int, routeVar map[string]string) bool {
	if permissionValue == permission.PermissionRead {
		return true
	}

	if permissionValue != permission.PermissionRead && getUser(ctx).Admin {
		return true
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

	if getHatchery(ctx) != nil {
		return getHatchery(ctx).GroupID == group.SharedInfraGroup.ID || m.GroupID == getHatchery(ctx).GroupID
	}
	return api.checkWorkerModelPermissionsByUser(m, getUser(ctx), permissionValue)
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
