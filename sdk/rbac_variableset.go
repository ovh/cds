package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

var (
	VariableSetRoleUse        = "use"
	VariableSetRoleManageItem = "manage-item"
	VariableSetRoles          = []string{VariableSetRoleManageItem, VariableSetRoleUse}
)

type RBACVariableSet struct {
	AllUsers             bool                 `json:"all_users" db:"all_users"`
	Role                 string               `json:"role" db:"role"`
	ProjectKey           string               `json:"project" db:"project_key"`
	RBACUsersName        []string             `json:"users,omitempty" db:"-"`
	RBACGroupsName       []string             `json:"groups,omitempty" db:"-"`
	RBACVariableSetNames RBACVariableSetNames `json:"variablesets,omitempty" db:"variablesets"`
	AllVariableSets      bool                 `json:"all_variablesets" db:"all_variablesets"`
	RBACVCSUsers         RBACVCSUsers         `json:"vcs_users,omitempty" db:"vcs_users"`

	RBACUsersIDs  []string `json:"-" db:"-"`
	RBACGroupsIDs []int64  `json:"-" db:"-"`
}

type RBACVariableSetNames []string

func (rwn RBACVariableSetNames) Value() (driver.Value, error) {
	names, err := json.Marshal(rwn)
	return names, WrapError(err, "cannot marshal RBACVariableSetNames")
}

// Scan action.
func (rwn *RBACVariableSetNames) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal(source, rwn), "cannot unmarshal RBACVariableSetNamesPoufpa")
}
