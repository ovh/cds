package main

import (
	"strconv"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// PermCheckFunc defines func call to check permission
type PermCheckFunc func(key string, c *businesscontext.Ctx, permission int, routeVar map[string]string) bool

var permissionMapFunction = initPermissionFunc()

func initPermissionFunc() map[string]PermCheckFunc {
	return map[string]PermCheckFunc{
		"permProjectKey":      checkProjectPermissions,
		"permPipelineKey":     checkPipelinePermissions,
		"permApplicationName": checkApplicationPermissions,
		"appID":               checkApplicationIDPermissions,
		"permGroupName":       checkGroupPermissions,
		"permActionName":      checkActionPermissions,
		"permEnvironmentName": checkEnvironmentPermissions,
		"permModelID":         checkWorkerModelPermissions,
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

func checkWorkerPermission(db gorp.SqlExecutor, rc *HandlerConfig, routeVar map[string]string, c *businesscontext.Ctx) bool {
	if c.Worker == nil {
		return false
	}

	idS, ok := routeVar["permID"]
	if !ok {
		return true
	}

	id, err := strconv.ParseInt(idS, 10, 64)
	if err != nil {
		log.Error("checkWorkerPermission> Unable to parse permID=%s", idS)
		return false
	}

	//IF it is POSTEXECUTE, it means that the job is must be taken by the worker
	if rc.isExecution {
		node, err := workflow.LoadNodeJobRun(db, id)
		if err != nil {
			log.Error("checkWorkerPermission> Unable to load job %d", id)
			return false
		}
		return node.Job.WorkerName == c.Worker.Name && node.Job.WorkerID == c.Worker.ID
	}
	return true
}

func checkPermission(routeVar map[string]string, c *businesscontext.Ctx, permission int) bool {
	for _, g := range c.User.Groups {
		if group.SharedInfraGroup != nil && g.Name == group.SharedInfraGroup.Name {
			return true
		}
	}

	permissionOk := true
	for key, value := range routeVar {
		if permFunc, ok := permissionMapFunction[key]; ok {
			log.Debug("Check permission for %s", key)
			permissionOk = permFunc(value, c, permission, routeVar)
			if !permissionOk {
				return permissionOk
			}
		}
	}
	return permissionOk
}

func checkProjectPermissions(projectKey string, c *businesscontext.Ctx, permission int, routeVar map[string]string) bool {
	if c.User.Groups != nil {
		for _, g := range c.User.Groups {
			for _, p := range g.ProjectGroups {
				if projectKey == p.Project.Key && p.Permission >= permission {
					return true
				}
			}
		}
	}
	log.Warning("Access denied. user %s on project %s", c.User.Username, projectKey)
	return false
}

func checkPipelinePermissions(pipelineName string, c *businesscontext.Ctx, permission int, routeVar map[string]string) bool {
	// Check if param key exist
	if projectKey, ok := routeVar["key"]; ok {
		for _, g := range c.User.Groups {
			for _, p := range g.PipelineGroups {
				if pipelineName == p.Pipeline.Name && p.Permission >= permission && projectKey == p.Pipeline.ProjectKey {
					return true
				}
			}
		}
		log.Warning("Access denied. user %s on pipeline %s", c.User.Username, pipelineName)
	} else {
		log.Warning("Wrong route configuration. need key parameter")
	}
	return false
}

func checkEnvironmentPermissions(envName string, c *businesscontext.Ctx, permission int, routeVar map[string]string) bool {
	// Check if param key exist
	if projectKey, ok := routeVar["key"]; ok {
		if c.User.Groups != nil {
			for _, g := range c.User.Groups {
				for _, p := range g.EnvironmentGroups {
					if envName == p.Environment.Name && p.Permission >= permission && projectKey == p.Environment.ProjectKey {
						return true
					}
				}
			}
		}
		log.Warning("Access denied. user %s on environment %s", c.User.Username, envName)
	} else {
		log.Warning("Wrong route configuration. need key parameter")
	}
	return false
}

func checkApplicationPermissions(applicationName string, c *businesscontext.Ctx, permission int, routeVar map[string]string) bool {
	// Check if param key exist
	if projectKey, ok := routeVar["key"]; ok {
		if c.User.Groups != nil {
			for _, g := range c.User.Groups {
				for _, a := range g.ApplicationGroups {
					if applicationName == a.Application.Name && a.Permission >= permission && projectKey == a.Application.ProjectKey {
						return true
					}
				}
			}
		}
		log.Warning("Access denied. user %s on application %s", c.User.Username, applicationName)
	} else {
		log.Warning("Wrong route configuration. need key parameter")
	}
	return false
}

func checkApplicationIDPermissions(appIDS string, c *businesscontext.Ctx, permission int, routeVar map[string]string) bool {
	appID, err := strconv.ParseInt(appIDS, 10, 64)
	if err != nil {
		log.Warning("checkApplicationIDPermissions> appID (%s) is not an integer: %s", appIDS, err)
		return false
	}

	if c.User.Groups != nil {
		for _, g := range c.User.Groups {
			for _, a := range g.ApplicationGroups {
				if appID == a.Application.ID && a.Permission >= permission {
					return true
				}
			}
		}
	}

	log.Warning("Access denied. user %s on application %s", c.User.Username, appIDS)
	return false
}

func checkGroupPermissions(groupName string, c *businesscontext.Ctx, permissionValue int, routeVar map[string]string) bool {
	for _, g := range c.User.Groups {
		if g.Name == groupName {

			if permissionValue == permission.PermissionRead {
				return true
			}

			for i := range g.Admins {
				if g.Admins[i].ID == c.User.ID {
					return true
				}
			}
		}
	}

	return false
}

func checkActionPermissions(groupName string, c *businesscontext.Ctx, permissionValue int, routeVar map[string]string) bool {
	if permissionValue == permission.PermissionRead {
		return true
	}

	if permissionValue != permission.PermissionRead && c.User.Admin {
		return true
	}

	return false
}

func checkWorkerModelPermissions(modelID string, c *businesscontext.Ctx, permissionValue int, routeVar map[string]string) bool {
	id, err := strconv.ParseInt(modelID, 10, 64)
	if err != nil {
		log.Warning("checkWorkerModelPermissions> modelID is not an integer: %s", err)
		return false
	}

	db := database.DB()
	if db == nil {
		return false
	}

	m, err := worker.LoadWorkerModelByID(database.DBMap(db), id)
	if err != nil {
		log.Warning("checkWorkerModelPermissions> unable to load model by id %s: %s", modelID, err)
		return false
	}

	if c.Hatchery != nil {
		return c.Hatchery.GroupID == group.SharedInfraGroup.ID || m.GroupID == c.Hatchery.GroupID
	}
	return checkWorkerModelPermissionsByUser(m, c.User, permissionValue)
}

func checkWorkerModelPermissionsByUser(m *sdk.Model, u *sdk.User, permissionValue int) bool {
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
