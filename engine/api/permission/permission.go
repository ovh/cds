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

// ProjectPermission  Get the permission for the given project
func ProjectPermission(projectKey string, u *sdk.AuthentifiedUser) int {
	if u == nil || u.Admin() {
		return PermissionReadWriteExecute
	}

	return u.OldUserStruct.Permissions.ProjectsPerm[projectKey]
}

// WorkflowPermission  Get the permission for the given workflow
func WorkflowPermission(key string, name string, u *sdk.AuthentifiedUser) int {
	if u.Admin() {
		return PermissionReadWriteExecute
	}

	if perm, ok := u.OldUserStruct.Permissions.WorkflowsPerm[sdk.UserPermissionKey(key, name)]; ok {
		return perm
	}

	// By default all workflows are RO
	return PermissionRead
}

// AccessToProject check if we can access to the given project
func AccessToProject(key string, u *sdk.AuthentifiedUser, access int) bool {
	if u.Admin() {
		return true
	}
	return u.OldUserStruct.Permissions.ProjectsPerm[key] >= access
}

// AccessToWorkflow check access to a workflow
func AccessToWorkflow(key, name string, u *sdk.AuthentifiedUser, access int) bool {
	if u.Admin() {
		return true
	}

	for _, g := range u.OldUserStruct.Groups {
		if g.ID == SharedInfraGroupID {
			return true
		}
	}

	if u.OldUserStruct.Permissions.WorkflowsPerm[sdk.UserPermissionKey(key, name)] >= access {
		return true
	}
	return false
}

// AccessToWorkflowNode check rights on the given workflow node
func AccessToWorkflowNode(wf *sdk.Workflow, wn *sdk.Node, u *sdk.AuthentifiedUser, access int) bool {
	if wn == nil {
		return false
	}

	if u.Admin() {
		return true
	}

	if len(wn.Groups) > 0 {
		for _, g := range u.OldUserStruct.Groups {
			if g.ID == SharedInfraGroupID {
				return true
			}
			for _, grp := range wn.Groups {
				if g.ID == grp.Group.ID && grp.Permission >= access {
					return true
				}
			}
		}
		return false
	}

	return AccessToWorkflow(wf.ProjectKey, wf.Name, u, PermissionReadExecute)
}
