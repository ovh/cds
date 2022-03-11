package sdk

var (
	GlobalRoles = []string{RoleCreateProject, RoleManagePermission}
)

type RbacGlobal struct {
	Role           string   `json:"role" db:"role" yaml:"role"`
	RbacUsersName  []string `json:"users" db:"-" yaml:"users"`
	RbacGroupsName []string `json:"groups" db:"-" yaml:"groups"`

	RbacUsersIDs  []string `json:"-" db:"-" yaml:"-"`
	RbacGroupsIDs []int64  `json:"-" db:"-" yaml:"-"`
}

func isValidRbacGlobal(rbacName string, rg RbacGlobal) error {
	if len(rg.RbacGroupsIDs) == 0 && len(rg.RbacUsersIDs) == 0 {
		return WrapError(ErrInvalidData, "Rbac %s: missing groups or users on global permission", rbacName)
	}
	if rg.Role == "" {
		return WrapError(ErrInvalidData, "Rbac %s: role for global permission cannot be empty", rbacName)
	}
	roleFound := false
	for _, r := range GlobalRoles {
		if rg.Role == r {
			roleFound = true
			break
		}
	}
	if !roleFound {
		return WrapError(ErrInvalidData, "Rbac %s: role %s is not allowed on a global permission", rbacName, rg.Role)
	}
	return nil
}
