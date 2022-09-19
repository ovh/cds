package sdk

var (
	GlobalRoles = []string{GlobalRoleProjectCreate, RoleManagePermission}
)

type RBACGlobal struct {
	Role           string   `json:"role" db:"role"`
	RBACUsersName  []string `json:"users" db:"-"`
	RBACGroupsName []string `json:"groups" db:"-"`
	RBACUsersIDs   []string `json:"-" db:"-"`
	RBACGroupsIDs  []int64  `json:"-" db:"-"`
}

func isValidRBACGlobal(rbacName string, rg RBACGlobal) error {
	if len(rg.RBACGroupsIDs) == 0 && len(rg.RBACUsersIDs) == 0 {
		return NewErrorFrom(ErrInvalidData, "rbac %s: missing groups or users on global permission", rbacName)
	}
	if rg.Role == "" {
		return NewErrorFrom(ErrInvalidData, "rbac %s: role for global permission cannot be empty", rbacName)
	}
	roleFound := false
	for _, r := range GlobalRoles {
		if rg.Role == r {
			roleFound = true
			break
		}
	}
	if !roleFound {
		return NewErrorFrom(ErrInvalidData, "rbac %s: role %s is not allowed on a global permission", rbacName, rg.Role)
	}
	return nil
}
