package sdk

var (
	RegionRoles = StringSlice{RegionRoleExecute, RegionRoleRead, RegionRoleManage}
)

type RBACRegion struct {
	ID                int64    `json:"-" db:"id"`
	RbacID            string   `json:"-"  db:"rbac_id"`
	Role              string   `json:"role" db:"role"`
	RegionID          string   `json:"region_id" db:"region_id"`
	AllUsers          bool     `json:"all_users" db:"all_users"`
	RBACOrganizations []string `json:"organizations" db:"-"`
	RBACUsersName     []string `json:"users" db:"-"`
	RBACGroupsName    []string `json:"groups" db:"-"`
	RegionName        string   `json:"region" db:"-"`

	RBACUsersIDs        []string `json:"-" db:"-"`
	RBACGroupsIDs       []int64  `json:"-" db:"-"`
	RBACOrganizationIDs []string `json:"-" db:"-"`
}

func isValidRBACRegion(rbacName string, rbacRegion RBACRegion) error {
	if rbacRegion.RegionID == "" {
		return NewErrorFrom(ErrInvalidData, "rbac %s: missing region", rbacName)
	}

	// Check role
	if rbacRegion.Role == "" {
		return NewErrorFrom(ErrInvalidData, "rbac %s: role for region permission cannot be empty", rbacName)
	}
	if !RegionRoles.Contains(rbacRegion.Role) {
		return NewErrorFrom(ErrInvalidData, "rbac %s: role %s is not allowed on a region permission", rbacName, rbacRegion.Role)
	}
	return nil
}
