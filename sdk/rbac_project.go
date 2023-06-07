package sdk

var (
	ProjectRoles = []string{ProjectRoleRead, ProjectRoleManage, ProjectRoleManageWorkerModel, ProjectRoleManageAction, ProjectRoleManageWorkflow}
)

type RBACProject struct {
	AllUsers        bool     `json:"all_users" db:"all_users"`
	Role            string   `json:"role" db:"role"`
	RBACProjectKeys []string `json:"projects,omitempty" db:"-"`
	RBACUsersName   []string `json:"users,omitempty" db:"-"`
	RBACGroupsName  []string `json:"groups,omitempty" db:"-"`

	RBACUsersIDs  []string `json:"-" db:"-"`
	RBACGroupsIDs []int64  `json:"-" db:"-"`
}
