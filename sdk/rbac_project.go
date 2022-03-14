package sdk

var (
	ProjectRoles = []string{RoleRead, RoleManage, RoleDelete}
)

type RbacProject struct {
	All             bool     `json:"all" db:"all" yaml:"all"`
	Role            string   `json:"role" db:"role" yaml:"role"`
	RbacProjectKeys []string `json:"projects" db:"-" yaml:"projects"`
	RbacUsersName   []string `json:"users" db:"-" yaml:"users"`
	RbacGroupsName  []string `json:"groups" db:"-" yaml:"groups"`

	RbacProjectsIDs []int64  `json:"-" db:"-" yaml:"-"`
	RbacUsersIDs    []string `json:"-" db:"-" yaml:"-"`
	RbacGroupsIDs   []int64  `json:"-" db:"-" yaml:"-"`
}

func isValidRbacProject(rbacName string, rp RbacProject) error {
	// Check empty group and users
	if len(rp.RbacGroupsIDs) == 0 && len(rp.RbacUsersIDs) == 0 {
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
	if len(rp.RbacProjectsIDs) == 0 && !rp.All {
		return NewErrorFrom(ErrInvalidData, "rbac %s: must have at least 1 project on a project permission", rbacName)
	}
	if len(rp.RbacProjectsIDs) > 0 && rp.All {
		return NewErrorFrom(ErrInvalidData, "rbac %s: you can't have a list of project and the all flag checked on a project permission", rbacName)
	}
	return nil
}
