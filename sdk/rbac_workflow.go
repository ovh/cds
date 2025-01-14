package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

var (
	WorkflowRoleTrigger = "trigger"
	WorkflowRoles       = []string{WorkflowRoleTrigger}
)

type RBACWorkflow struct {
	AllUsers           bool              `json:"all_users" db:"all_users"`
	Role               string            `json:"role" db:"role"`
	ProjectKey         string            `json:"project" db:"project_key"`
	RBACUsersName      []string          `json:"users,omitempty" db:"-"`
	RBACGroupsName     []string          `json:"groups,omitempty" db:"-"`
	RBACWorkflowsNames RBACWorkflowNames `json:"workflows,omitempty" db:"workflows"`
	AllWorkflows       bool              `json:"all_workflows" db:"all_workflows"`
	RBACVCSUsers       RBACVCSUsers      `json:"vcs_users,omitempty" db:"vcs_users"`

	RBACUsersIDs  []string `json:"-" db:"-"`
	RBACGroupsIDs []int64  `json:"-" db:"-"`
}

type RBACVCSUsers []RBACVCSUser

func (rwn RBACVCSUsers) Value() (driver.Value, error) {
	names, err := json.Marshal(rwn)
	return names, WrapError(err, "cannot marshal RBACVCSUsers")
}

// Scan action.
func (rwn *RBACVCSUsers) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal(source, rwn), "cannot unmarshal RBACVCSUsers")
}

type RBACVCSUser struct {
	VCSServer   string `json:"server"`
	VCSUsername string `json:"username"`
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
