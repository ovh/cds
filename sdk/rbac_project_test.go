package sdk

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRBACProjectInvalidRole(t *testing.T) {
	rb := RBACProject{
		RBACProjectKeys: []string{"foo"},
		All:             false,
		Role:            RoleCreateProject,
		RBACGroupsIDs:   []int64{1},
		RBACUsersIDs:    []string{"aa-aa-aa"},
	}
	err := isValidRBACProject("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), fmt.Sprintf("rbac myRule: role %s is not allowed on a project permission", RoleCreateProject))
}
func TestRBACProjectInvalidGroupAndUsers(t *testing.T) {
	rb := RBACProject{
		RBACProjectKeys: []string{"foo"},
		All:             false,
		Role:            RoleRead,
		RBACGroupsIDs:   []int64{},
		RBACUsersIDs:    []string{},
	}
	err := isValidRBACProject("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "rbac myRule: missing groups or users on project permission")
}
func TestRBACProjectInvalidProjectKeys(t *testing.T) {
	rb := RBACProject{
		RBACProjectKeys: []string{},
		All:             false,
		Role:            RoleRead,
		RBACGroupsIDs:   []int64{1},
		RBACUsersIDs:    []string{},
	}
	err := isValidRBACProject("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "rbac myRule: must have at least 1 project on a project permission")
}
func TestRBACProjectEmptyRole(t *testing.T) {
	rb := RBACProject{
		RBACProjectKeys: []string{"foo"},
		All:             false,
		Role:            "",
		RBACGroupsIDs:   []int64{1},
		RBACUsersIDs:    []string{},
	}
	err := isValidRBACProject("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "rbac myRule: role for project permission cannot be empty")
}
func TestRBACProjectInvalidAllAndListOfProject(t *testing.T) {
	rb := RBACProject{
		RBACProjectKeys: []string{"foo"},
		All:             true,
		Role:            RoleRead,
		RBACGroupsIDs:   []int64{1},
		RBACUsersIDs:    []string{},
	}
	err := isValidRBACProject("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "rbac myRule: you can't have a list of project and the all flag checked on a project permission")
}
