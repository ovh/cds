package sdk

var (
	RegionRoles = StringSlice{RegionRoleExecute, RegionRoleList, RegionRoleManage}
)

type RBACRegion struct {
	ID                int64        `json:"-" db:"id"`
	RbacID            string       `json:"-"  db:"rbac_id"`
	Role              string       `json:"role" db:"role"`
	RegionID          string       `json:"region_id" db:"region_id"`
	AllUsers          bool         `json:"all_users,omitempty" db:"all_users"`
	RBACOrganizations []string     `json:"organizations,omitempty" db:"-"`
	RBACUsersName     []string     `json:"users,omitempty" db:"-"`
	RBACGroupsName    []string     `json:"groups,omitempty" db:"-"`
	RegionName        string       `json:"region" db:"-"`
	RBACVCSUsers      RBACVCSUsers `json:"vcs_users,omitempty" db:"vcs_users"`

	RBACUsersIDs        []string `json:"-" db:"-"`
	RBACGroupsIDs       []int64  `json:"-" db:"-"`
	RBACOrganizationIDs []string `json:"-" db:"-"`
}
