package permission

import (
	"github.com/ovh/cds/sdk"
)

const (
	// PermissionRead  read permission on the resource
	PermissionRead = 4
	// PermissionReadExecute  read & execute permission on the resource
	PermissionReadExecute = 5
	// PermissionReadWriteExecute read/execute/write permission on the resource
	PermissionReadWriteExecute = 7
)

var (
	// SharedInfraGroupID must be init from elsewhere with group.SharedInfraGroup
	SharedInfraGroupID int64

	// DefaultGroupID same as SharedInfraGroupID
	DefaultGroupID int64
)

// ApplicationPermission  Get the permission for the given application
func ApplicationPermission(key string, appName string, u *sdk.User) int {
	if u.Admin {
		return PermissionReadWriteExecute
	}

	return u.Permissions.ApplicationsPerm[sdk.UserPermissionKey(key, appName)]
}

// ProjectPermission  Get the permission for the given project
func ProjectPermission(projectKey string, u *sdk.User) int {
	if u.Admin || u == nil {
		return PermissionReadWriteExecute
	}

	return u.Permissions.ProjectsPerm[projectKey]
}

// WorkflowPermission  Get the permission for the given workflow
func WorkflowPermission(key string, name string, u *sdk.User) int {
	if u.Admin {
		return PermissionReadWriteExecute
	}

	return u.Permissions.WorkflowsPerm[sdk.UserPermissionKey(key, name)]
}

// PipelinePermission  Get the permission for the given pipeline
func PipelinePermission(key string, name string, u *sdk.User) int {
	if u.Admin {
		return PermissionReadWriteExecute
	}

	return u.Permissions.PipelinesPerm[sdk.UserPermissionKey(key, name)]
}

// EnvironmentPermission  Get the permission for the given environment
func EnvironmentPermission(key string, name string, u *sdk.User) int {
	if u.Admin {
		return PermissionReadWriteExecute
	}
	return u.Permissions.EnvironmentsPerm[sdk.UserPermissionKey(key, name)]
}

// AccessToApplication check if we can modify the given application
func AccessToApplication(key string, name string, u *sdk.User, access int) bool {
	if u.Admin {
		return true
	}

	return u.Permissions.ApplicationsPerm[sdk.UserPermissionKey(key, name)] >= access
}

// AccessToPipeline check if we can modify the given pipeline
func AccessToPipeline(key string, env, pip string, u *sdk.User, access int) bool {
	if u.Admin {
		return true
	}

	for _, g := range u.Groups {
		if g.ID == SharedInfraGroupID {
			return true
		}
	}

	if u.Permissions.PipelinesPerm[sdk.UserPermissionKey(key, pip)] >= access {
		if env != sdk.DefaultEnv.Name {
			return AccessToEnvironment(key, env, u, access)
		}
		return true
	}

	return false
}

// AccessToEnvironment check if we can modify the given environment
func AccessToEnvironment(key, env string, u *sdk.User, access int) bool {
	if env == "" || env == sdk.DefaultEnv.Name {
		return true
	}

	if u.Admin {
		return true
	}

	for _, g := range u.Groups {
		if g.ID == SharedInfraGroupID {
			return true
		}
	}

	return u.Permissions.EnvironmentsPerm[sdk.UserPermissionKey(key, env)] >= access
}
