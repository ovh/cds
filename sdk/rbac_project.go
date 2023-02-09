package sdk

var (
	ProjectRoles = []string{ProjectRoleRead, ProjectRoleManage, ProjectRoleManageWorkerModel}
)

type RBACProject struct {
	All             bool     `json:"all" db:"all"`
	Role            string   `json:"role" db:"role"`
	RBACProjectKeys []string `json:"projects,omitempty" db:"-"`
	RBACUsersName   []string `json:"users,omitempty" db:"-"`
	RBACGroupsName  []string `json:"groups,omitempty" db:"-"`

	RBACUsersIDs  []string `json:"-" db:"-"`
	RBACGroupsIDs []int64  `json:"-" db:"-"`
}
