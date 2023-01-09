package sdk

var (
	ProjectRoles = []string{ProjectRoleRead, ProjectRoleManage, ProjectRoleManageWorkerModel}
)

type RBACProject struct {
	All             bool     `json:"all" db:"all"`
	Role            string   `json:"role" db:"role"`
	RBACProjectKeys []string `json:"projects" db:"-"`
	RBACUsersName   []string `json:"users" db:"-"`
	RBACGroupsName  []string `json:"groups" db:"-"`

	RBACUsersIDs  []string `json:"-" db:"-"`
	RBACGroupsIDs []int64  `json:"-" db:"-"`
}

func isValidRBACProject(rbacName string, rbacProject RBACProject) error {
	// Check empty group and users
	if len(rbacProject.RBACGroupsIDs) == 0 && len(rbacProject.RBACUsersIDs) == 0 {
		return NewErrorFrom(ErrInvalidData, "rbac %s: missing groups or users on project permission", rbacName)
	}

	// Check role
	if rbacProject.Role == "" {
		return NewErrorFrom(ErrInvalidData, "rbac %s: role for project permission cannot be empty", rbacName)
	}
	roleFound := false
	for _, r := range ProjectRoles {
		if r == rbacProject.Role {
			roleFound = true
			break
		}
	}
	if !roleFound {
		return NewErrorFrom(ErrInvalidData, "rbac %s: role %s is not allowed on a project permission", rbacName, rbacProject.Role)
	}

	// Check project_key and all flag
	if len(rbacProject.RBACProjectKeys) == 0 && !rbacProject.All {
		return NewErrorFrom(ErrInvalidData, "rbac %s: must have at least 1 project on a project permission", rbacName)
	}
	if len(rbacProject.RBACProjectKeys) > 0 && rbacProject.All {
		return NewErrorFrom(ErrInvalidData, "rbac %s: you can't have a list of project and the all flag checked on a project permission", rbacName)
	}
	return nil
}
