package sdk

var (
	ProjectRoles = []string{ProjectRoleRead, ProjectRoleManage, ProjectRoleManageNotification, ProjectRoleManageWorkerModel, ProjectRoleManageAction, ProjectRoleManageWorkflow, ProjectRoleManageWorkflowTemplate, ProjectRoleManageVariableSet}
)

type RBACProject struct {
	AllUsers        bool         `json:"all_users" db:"all_users"`
	AllVCSUsers     bool         `json:"all_vcs_users" db:"all_vcs_users"`
	Role            string       `json:"role" db:"role"`
	RBACProjectKeys []string     `json:"projects,omitempty" db:"-"`
	RBACUsersName   []string     `json:"users,omitempty" db:"-"`
	RBACGroupsName  []string     `json:"groups,omitempty" db:"-"`
	RBACVCSUsers    RBACVCSUsers `json:"vcs_users,omitempty" db:"vcs_users"`

	RBACUsersIDs  []string `json:"-" db:"-"`
	RBACGroupsIDs []int64  `json:"-" db:"-"`
}
