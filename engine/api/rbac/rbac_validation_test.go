package rbac

import (
	"context"
	"fmt"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/organization"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInvalidRbacRegionGroups(t *testing.T) {
	ctx := context.TODO()
	db, _ := test.SetupPG(t)

	reg := sdk.Region{Name: sdk.RandomString(10)}
	require.NoError(t, region.Insert(ctx, db, &reg))

	org1 := sdk.Organization{Name: sdk.RandomString(10)}
	require.NoError(t, organization.Insert(ctx, db, &org1))

	org2 := sdk.Organization{Name: sdk.RandomString(10)}
	require.NoError(t, organization.Insert(ctx, db, &org2))

	grp1 := sdk.Group{Name: sdk.RandomString(10)}
	require.NoError(t, group.Insert(ctx, db, &grp1))
	grpOrg := group.GroupOrganization{OrganizationID: org2.ID, GroupID: grp1.ID}
	require.NoError(t, group.InsertGroupOrganization(ctx, db, &grpOrg))

	r := sdk.RBAC{
		Name: sdk.RandomString(10),
		Regions: []sdk.RBACRegion{
			{
				RBACGroupsIDs:       []int64{grp1.ID},
				RBACOrganizationIDs: []string{org1.ID},
				RegionID:            reg.ID,
				Role:                sdk.RegionRoleManage,
			},
		},
	}
	err := Insert(ctx, db, &r)
	require.Error(t, err)
	require.Contains(t, err.Error(), "some groups are not part of the allowed organizations")
}

func TestInvalidRbacRegionUsers(t *testing.T) {
	ctx := context.TODO()
	db, _ := test.SetupPG(t)

	reg := sdk.Region{Name: sdk.RandomString(10)}
	require.NoError(t, region.Insert(ctx, db, &reg))

	org1 := sdk.Organization{Name: sdk.RandomString(10)}
	require.NoError(t, organization.Insert(ctx, db, &org1))

	org2 := sdk.Organization{Name: sdk.RandomString(10)}
	require.NoError(t, organization.Insert(ctx, db, &org2))

	user1 := sdk.AuthentifiedUser{Fullname: sdk.RandomString(10), Username: sdk.RandomString(10)}
	require.NoError(t, user.Insert(ctx, db, &user1))
	userOrg := user.UserOrganization{OrganizationID: org2.ID, AuthentifiedUserID: user1.ID}
	require.NoError(t, user.InsertUserOrganization(ctx, db, &userOrg))

	r := sdk.RBAC{
		Name: sdk.RandomString(10),
		Regions: []sdk.RBACRegion{
			{
				RBACUsersIDs:        []string{user1.ID},
				RBACOrganizationIDs: []string{org1.ID},
				RegionID:            reg.ID,
				Role:                sdk.RegionRoleManage,
			},
		},
	}
	err := Insert(ctx, db, &r)
	require.Error(t, err)
	require.Contains(t, err.Error(), "some users are not part of the allowed organizations")
}

func TestRBACGlobalInvalidGlobalRole(t *testing.T) {
	rb := sdk.RBACGlobal{
		Role:          "runWorkflow",
		RBACGroupsIDs: []int64{1},
		RBACUsersIDs:  []string{"aa-aa-aa"},
	}
	err := isValidRBACGlobal("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "rbac myRule: role runWorkflow is not allowed on a global permission")
}
func TestRBACGlobalInvalidGroupAndUsers(t *testing.T) {
	rb := sdk.RBACGlobal{
		Role:          sdk.GlobalRoleProjectCreate,
		RBACGroupsIDs: []int64{},
		RBACUsersIDs:  []string{},
	}
	err := isValidRBACGlobal("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "rbac myRule: missing groups or users on global permission")
}

func TestRBACGlobalEmptyRole(t *testing.T) {
	rb := sdk.RBACGlobal{
		Role:          "",
		RBACGroupsIDs: []int64{1},
		RBACUsersIDs:  []string{},
	}
	err := isValidRBACGlobal("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "rbac myRule: role for global permission cannot be empty")
}

func TestRBACProjectInvalidRole(t *testing.T) {
	rb := sdk.RBACProject{
		RBACProjectKeys: []string{"foo"},
		AllUsers:        false,
		Role:            sdk.GlobalRoleProjectCreate,
		RBACGroupsIDs:   []int64{1},
		RBACUsersIDs:    []string{"aa-aa-aa"},
	}
	err := isValidRBACProject("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), fmt.Sprintf("rbac myRule: role %s is not allowed on a project permission", sdk.GlobalRoleProjectCreate))
}
func TestRBACProjectInvalidGroupAndUsers(t *testing.T) {
	rb := sdk.RBACProject{
		RBACProjectKeys: []string{"foo"},
		AllUsers:        false,
		Role:            sdk.ProjectRoleRead,
		RBACGroupsIDs:   []int64{},
		RBACUsersIDs:    []string{},
	}
	err := isValidRBACProject("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "rbac myRule: missing groups or users on project permission")
}
func TestRBACProjectInvalidGroupsAndUsers(t *testing.T) {
	rb := sdk.RBACProject{
		RBACProjectKeys: []string{"PROJ"},
		AllUsers:        false,
		Role:            sdk.ProjectRoleRead,
		RBACGroupsIDs:   []int64{},
		RBACUsersIDs:    []string{},
	}
	err := isValidRBACProject("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "rbac myRule: missing groups or users on project permission")
}
func TestRBACProjectEmptyRole(t *testing.T) {
	rb := sdk.RBACProject{
		RBACProjectKeys: []string{"foo"},
		AllUsers:        false,
		Role:            "",
		RBACGroupsIDs:   []int64{1},
		RBACUsersIDs:    []string{},
	}
	err := isValidRBACProject("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "rbac myRule: role for project permission cannot be empty")
}
func TestRBACProjectInvalidAllAndListOfGroups(t *testing.T) {
	rb := sdk.RBACProject{
		RBACProjectKeys: []string{"foo"},
		AllUsers:        true,
		Role:            sdk.ProjectRoleRead,
		RBACGroupsIDs:   []int64{1},
		RBACUsersIDs:    []string{},
	}
	err := isValidRBACProject("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "rbac myRule: cannot have a list of groups or users with flag allUsers")
}

func TestRBACWorkflowInvalidRole(t *testing.T) {
	rb := sdk.RBACWorkflow{
		RBACWorkflowsNames: []string{"foo"},
		AllUsers:           false,
		Role:               sdk.GlobalRoleProjectCreate,
		RBACGroupsIDs:      []int64{1},
		RBACUsersIDs:       []string{"aa-aa-aa"},
	}
	err := isValidRBACWorkflow("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), fmt.Sprintf("rbac myRule: role %s is not allowed on a workflow permission", sdk.GlobalRoleProjectCreate))
}
func TestRBACWorkflowInvalidGroupAndUsers(t *testing.T) {
	rb := sdk.RBACWorkflow{
		RBACWorkflowsNames: []string{"foo"},
		AllUsers:           false,
		Role:               sdk.WorkflowRoleTrigger,
		RBACGroupsIDs:      []int64{},
		RBACUsersIDs:       []string{},
	}
	err := isValidRBACWorkflow("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "rbac myRule: missing groups or users on workflow permission")
}
func TestRBACWorkflowInvalidGroupsAndUsers(t *testing.T) {
	rb := sdk.RBACWorkflow{
		RBACWorkflowsNames: []string{"PROJ"},
		AllUsers:           false,
		Role:               sdk.WorkflowRoleTrigger,
		RBACGroupsIDs:      []int64{},
		RBACUsersIDs:       []string{},
	}
	err := isValidRBACWorkflow("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "rbac myRule: missing groups or users on workflow permission")
}
func TestRBACWorkflowEmptyRole(t *testing.T) {
	rb := sdk.RBACWorkflow{
		RBACWorkflowsNames: []string{"foo"},
		AllUsers:           false,
		Role:               "",
		RBACGroupsIDs:      []int64{1},
		RBACUsersIDs:       []string{},
	}
	err := isValidRBACWorkflow("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "rbac myRule: role for workflow permission cannot be empty")
}
func TestRBACWorkflowInvalidAllAndListOfGroups(t *testing.T) {
	rb := sdk.RBACWorkflow{
		RBACWorkflowsNames: []string{"foo"},
		AllUsers:           true,
		Role:               sdk.WorkflowRoleTrigger,
		RBACGroupsIDs:      []int64{1},
		RBACUsersIDs:       []string{},
	}
	err := isValidRBACWorkflow("myRule", rb)
	require.Error(t, err)
	require.Contains(t, err.Error(), "rbac myRule: cannot have a list of groups or users with flag allUsers")
}
