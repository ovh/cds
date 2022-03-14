package sdk

var (
	ProjectRoles = []string{RoleRead, RoleManage, RoleDelete}
)

type RBACProject struct {
	All             bool     `json:"all" db:"all" yaml:"all"`
	Role            string   `json:"role" db:"role" yaml:"role"`
	RBACProjectKeys []string `json:"projects" db:"-" yaml:"projects"`
	RBACUsersName   []string `json:"users" db:"-" yaml:"users"`
	RBACGroupsName  []string `json:"groups" db:"-" yaml:"groups"`

	RBACProjectsIDs []int64  `json:"-" db:"-" yaml:"-"`
	RBACUsersIDs    []string `json:"-" db:"-" yaml:"-"`
	RBACGroupsIDs   []int64  `json:"-" db:"-" yaml:"-"`
}

func isValidRbacProject(rbacName string, rp RBACProject) error {
	// Check empty group and users
	if len(rp.RBACGroupsIDs) == 0 && len(rp.RBACUsersIDs) == 0 {
		return NewErrorFrom(ErrInvalidData, "rbac %s: missing groups or users on project permission", rbacName)
	}

	// Check role
	if rp.Role == "" {
		return NewErrorFrom(ErrInvalidData, "rbac %s: role for project permission cannot be empty", rbacName)
	}
	roleFound := false
	for _, r := range ProjectRoles {
		if r == rp.Role {
			roleFound = true
			break
		}
	}
	if !roleFound {
		return NewErrorFrom(ErrInvalidData, "rbac %s: role %s is not allowed on a project permission", rbacName, rp.Role)
	}

	// Check project_ids and all flag
	if len(rp.RBACProjectsIDs) == 0 && !rp.All {
		return NewErrorFrom(ErrInvalidData, "rbac %s: must have at least 1 project on a project permission", rbacName)
	}
	if len(rp.RBACProjectsIDs) > 0 && rp.All {
		return NewErrorFrom(ErrInvalidData, "rbac %s: you can't have a list of project and the all flag checked on a project permission", rbacName)
	}
	return nil
}
