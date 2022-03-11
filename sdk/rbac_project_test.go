package sdk

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRbacProjectInvalidRole(t *testing.T) {
	rb := RbacProject{
		RbacProjectsIDs: []int64{1},
		All:             false,
		Role:            RoleCreateProject,
		RbacGroupsIDs:   []int64{1},
		RbacUsersIDs:    []string{"aa-aa-aa"},
	}
	err := isValidRbacProject("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Rbac myRule: role createProject is not allowed on a project permission")
}
func TestRbacProjectInvalidGroupAndUsers(t *testing.T) {
	rb := RbacProject{
		RbacProjectsIDs: []int64{1},
		All:             false,
		Role:            RoleRead,
		RbacGroupsIDs:   []int64{},
		RbacUsersIDs:    []string{},
	}
	err := isValidRbacProject("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Rbac myRule: missing groups or users on project permission")
}
func TestRbacProjectInvalidProjectIDs(t *testing.T) {
	rb := RbacProject{
		RbacProjectsIDs: []int64{},
		All:             false,
		Role:            RoleRead,
		RbacGroupsIDs:   []int64{1},
		RbacUsersIDs:    []string{},
	}
	err := isValidRbacProject("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Rbac myRule: must have at least 1 project on a project permission")
}
func TestRbacProjectEmptyRole(t *testing.T) {
	rb := RbacProject{
		RbacProjectsIDs: []int64{1},
		All:             false,
		Role:            "",
		RbacGroupsIDs:   []int64{1},
		RbacUsersIDs:    []string{},
	}
	err := isValidRbacProject("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Rbac myRule: role for project permission cannot be empty")
}
func TestRbacProjectInvalidAllAndListOfProject(t *testing.T) {
	rb := RbacProject{
		RbacProjectsIDs: []int64{1},
		All:             true,
		Role:            RoleRead,
		RbacGroupsIDs:   []int64{1},
		RbacUsersIDs:    []string{},
	}
	err := isValidRbacProject("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Rbac myRule: you can't have a list of project and the all flag checked on a project permission")
}
