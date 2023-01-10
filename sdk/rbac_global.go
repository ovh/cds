package sdk

var (
	GlobalRoles = []string{GlobalRoleProjectCreate, GlobalRoleManagePermission, GlobalRoleManageOrganization, GlobalRoleManageRegion, GlobalRoleManageUser, GlobalRoleManageGroup, GlobalRoleManageHatchery}
)

type RBACGlobal struct {
	Role           string   `json:"role" db:"role"`
	RBACUsersName  []string `json:"users,omitempty" db:"-"`
	RBACGroupsName []string `json:"groups,omitempty" db:"-"`
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

	if !IsInArray(rg.Role, GlobalRoles) {
		return NewErrorFrom(ErrInvalidData, "rbac %s: role %s is not allowed on a global permission", rbacName, rg.Role)
	}
	return nil
}
