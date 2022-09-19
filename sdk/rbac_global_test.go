package sdk

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRBACGlobalInvalidGlobalRole(t *testing.T) {
	rb := RBACGlobal{
		Role:          "runWorkflow",
		RBACGroupsIDs: []int64{1},
		RBACUsersIDs:  []string{"aa-aa-aa"},
	}
	err := isValidRBACGlobal("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "rbac myRule: role runWorkflow is not allowed on a global permission")
}
func TestRBACGlobalInvalidGroupAndUsers(t *testing.T) {
	rb := RBACGlobal{
		Role:          GlobalRoleProjectCreate,
		RBACGroupsIDs: []int64{},
		RBACUsersIDs:  []string{},
	}
	err := isValidRBACGlobal("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "rbac myRule: missing groups or users on global permission")
}

func TestRBACGlobalEmptyRole(t *testing.T) {
	rb := RBACGlobal{
		Role:          "",
		RBACGroupsIDs: []int64{1},
		RBACUsersIDs:  []string{},
	}
	err := isValidRBACGlobal("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "rbac myRule: role for global permission cannot be empty")
}
