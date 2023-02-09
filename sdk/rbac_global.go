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
