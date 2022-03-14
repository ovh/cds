package sdk

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRbacProjectInvalidRole(t *testing.T) {
	rb := RBACProject{
		RBACProjectsIDs: []int64{1},
		All:             false,
		Role:            RoleCreateProject,
		RBACGroupsIDs:   []int64{1},
		RBACUsersIDs:    []string{"aa-aa-aa"},
	}
	err := isValidRbacProject("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), fmt.Sprintf("rbac myRule: role %s is not allowed on a project permission", RoleCreateProject))
}
func TestRbacProjectInvalidGroupAndUsers(t *testing.T) {
	rb := RBACProject{
		RBACProjectsIDs: []int64{1},
		All:             false,
		Role:            RoleRead,
		RBACGroupsIDs:   []int64{},
		RBACUsersIDs:    []string{},
	}
	err := isValidRbacProject("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "rbac myRule: missing groups or users on project permission")
}
func TestRbacProjectInvalidProjectIDs(t *testing.T) {
	rb := RBACProject{
		RBACProjectsIDs: []int64{},
		All:             false,
		Role:            RoleRead,
		RBACGroupsIDs:   []int64{1},
		RBACUsersIDs:    []string{},
	}
	err := isValidRbacProject("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "rbac myRule: must have at least 1 project on a project permission")
}
func TestRbacProjectEmptyRole(t *testing.T) {
	rb := RBACProject{
		RBACProjectsIDs: []int64{1},
		All:             false,
		Role:            "",
		RBACGroupsIDs:   []int64{1},
		RBACUsersIDs:    []string{},
	}
	err := isValidRbacProject("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "rbac myRule: role for project permission cannot be empty")
}
func TestRbacProjectInvalidAllAndListOfProject(t *testing.T) {
	rb := RBACProject{
		RBACProjectsIDs: []int64{1},
		All:             true,
		Role:            RoleRead,
		RBACGroupsIDs:   []int64{1},
		RBACUsersIDs:    []string{},
	}
	err := isValidRbacProject("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "rbac myRule: you can't have a list of project and the all flag checked on a project permission")
}
