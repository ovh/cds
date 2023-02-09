package rbac

import (
	"context"
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
