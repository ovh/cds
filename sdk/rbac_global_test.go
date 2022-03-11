package sdk

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRbacGlobalInvalidGlobalRole(t *testing.T) {
	rb := RbacGlobal{
		Role:          "runWorkflow",
		RbacGroupsIDs: []int64{1},
		RbacUsersIDs:  []string{"aa-aa-aa"},
	}
	err := isValidRbacGlobal("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Rbac myRule: role runWorkflow is not allowed on a global permission")
}
func TestRbacGlobalInvalidGroupAndUsers(t *testing.T) {
	rb := RbacGlobal{
		Role:          RoleCreateProject,
		RbacGroupsIDs: []int64{},
		RbacUsersIDs:  []string{},
	}
	err := isValidRbacGlobal("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Rbac myRule: missing groups or users on global permission")
}

func TestRbacGlobalEmptyRole(t *testing.T) {
	rb := RbacGlobal{
		Role:          "",
		RbacGroupsIDs: []int64{1},
		RbacUsersIDs:  []string{},
	}
	err := isValidRbacGlobal("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Rbac myRule: role for global permission cannot be empty")
}
