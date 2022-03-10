package sdk

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRbacGlobalInvalidGlobalRole(t *testing.T) {
	rb := RbacGlobal{
		AbstractRbac: AbstractRbac{
			Role:       "runWorkflow",
			RbacGroups: []RbacGroup{{GroupID: 1}},
			RbacUsers:  []RbacUser{{Name: "aa-aa-aa"}},
		},
	}
	err := isValidRbacGlobal("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Rbac myRule: role runWorkflow is not allowed on a global permission")
}
func TestRbacGlobalInvalidGroupAndUsers(t *testing.T) {
	rb := RbacGlobal{
		AbstractRbac: AbstractRbac{
			Role:       RoleCreateProject,
			RbacGroups: []RbacGroup{},
			RbacUsers:  []RbacUser{},
		},
	}
	err := isValidRbacGlobal("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Rbac myRule: missing groups or users on global permission")
}

func TestRbacGlobalEmptyRole(t *testing.T) {
	rb := RbacGlobal{
		AbstractRbac: AbstractRbac{
			Role:       "",
			RbacGroups: []RbacGroup{{GroupID: 1}},
			RbacUsers:  []RbacUser{},
		},
	}
	err := isValidRbacGlobal("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Rbac myRule: role for global permission cannot be empty")
}
