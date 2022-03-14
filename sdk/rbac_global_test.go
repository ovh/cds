package sdk

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRbacGlobalInvalidGlobalRole(t *testing.T) {
	rb := RBACGlobal{
		Role:          "runWorkflow",
		RBACGroupsIDs: []int64{1},
		RBACUsersIDs:  []string{"aa-aa-aa"},
	}
	err := isValidRbacGlobal("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "rbac myRule: role runWorkflow is not allowed on a global permission")
}
func TestRbacGlobalInvalidGroupAndUsers(t *testing.T) {
	rb := RBACGlobal{
		Role:          RoleCreateProject,
		RBACGroupsIDs: []int64{},
		RBACUsersIDs:  []string{},
	}
	err := isValidRbacGlobal("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "rbac myRule: missing groups or users on global permission")
}

func TestRbacGlobalEmptyRole(t *testing.T) {
	rb := RBACGlobal{
		Role:          "",
		RBACGroupsIDs: []int64{1},
		RBACUsersIDs:  []string{},
	}
	err := isValidRbacGlobal("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "rbac myRule: role for global permission cannot be empty")
}
