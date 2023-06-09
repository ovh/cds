package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

var (
	WorkflowRoleExecute = "execute"
	WorkflowRoles       = []string{WorkflowRoleExecute}
)

type RBACWorkflow struct {
	AllUsers           bool              `json:"all_users" db:"all_users"`
	Role               string            `json:"role" db:"role"`
	ProjectKey         string            `json:"project" db:"project_key"`
	RBACUsersName      []string          `json:"users,omitempty" db:"-"`
	RBACGroupsName     []string          `json:"groups,omitempty" db:"-"`
	RBACWorkflowsNames RBACWorkflowNames `json:"workflows,omitempty" db:"workflows"`
	AllWorkflows       bool              `json:"all_workflows" db:"all_workflows"`

	RBACUsersIDs  []string `json:"-" db:"-"`
	RBACGroupsIDs []int64  `json:"-" db:"-"`
}

type RBACWorkflowNames []string

func (rwn RBACWorkflowNames) Value() (driver.Value, error) {
	names, err := json.Marshal(rwn)
	return names, WrapError(err, "cannot marshal RBACWorkflowNames")
}

// Scan action.
func (rwn *RBACWorkflowNames) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal(source, rwn), "cannot unmarshal RBACWorkflowNames")
}
