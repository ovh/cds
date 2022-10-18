package rbac_test

import (
	"context"
	"fmt"
	"github.com/ovh/cds/engine/api/organization"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/yaml"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestImportRbacRegion(t *testing.T) {
	db, _ := test.SetupPG(t)

	_, err := db.Exec("DELETE FROM rbac")
	require.NoError(t, err)
	_, err = db.Exec("DELETE FROM region")
	require.NoError(t, err)

	grpName1 := sdk.RandomString(10)
	group1 := assets.InsertTestGroup(t, db, grpName1)

	user1, _ := assets.InsertLambdaUser(t, db)

	reg := sdk.Region{Name: sdk.RandomString(10)}
	require.NoError(t, region.Insert(context.TODO(), db, &reg))

	org, err := organization.LoadOrganizationByName(context.TODO(), db, "default")
	require.NoError(t, err)

	rbacYaml := `name: perm-%s
regions:
- role: %s
  region: %s
  users: [%s]
  groups: [%s]
  organizations: [default]`

	rbacYaml = fmt.Sprintf(rbacYaml, reg.Name, sdk.RegionRoleManage, reg.Name, user1.Username, group1.Name)
	var r sdk.RBAC
	require.NoError(t, yaml.Unmarshal([]byte(rbacYaml), &r))

	require.NoError(t, rbac.FillWithIDs(context.TODO(), db, &r))
	require.NoError(t, rbac.Insert(context.TODO(), db, &r))

	rbacRB, err := rbac.LoadRBACByName(context.TODO(), db, r.Name, rbac.LoadOptions.LoadRBACRegion)
	require.NoError(t, err)
	require.Equal(t, 1, len(rbacRB.Regions))
	require.Equal(t, reg.ID, rbacRB.Regions[0].RegionID)

	require.Equal(t, 1, len(rbacRB.Regions[0].RBACUsersIDs))
	require.Equal(t, 1, len(rbacRB.Regions[0].RBACGroupsIDs))
	require.Equal(t, 1, len(rbacRB.Regions[0].RBACOrganizationIDs))

	require.Equal(t, user1.ID, rbacRB.Regions[0].RBACUsersIDs[0])
	require.Equal(t, group1.ID, rbacRB.Regions[0].RBACGroupsIDs[0])
	require.Equal(t, org.ID, rbacRB.Regions[0].RBACOrganizationIDs[0])

}

func TestRbacRegionAllUsers(t *testing.T) {
	db, cache := test.SetupPG(t)

	_, err := db.Exec("DELETE FROM rbac")
	require.NoError(t, err)
	_, err = db.Exec("DELETE FROM region")
	require.NoError(t, err)

	user1, _ := assets.InsertLambdaUser(t, db)

	reg := sdk.Region{Name: sdk.RandomString(10)}
	require.NoError(t, region.Insert(context.TODO(), db, &reg))

	rbacYaml := `name: perm-%s
regions:
- role: %s
  region: %s
  all_users: true
  organizations: [default]`

	rbacYaml = fmt.Sprintf(rbacYaml, reg.Name, sdk.RegionRoleRead, reg.Name)
	var r sdk.RBAC
	require.NoError(t, yaml.Unmarshal([]byte(rbacYaml), &r))

	require.NoError(t, rbac.FillWithIDs(context.TODO(), db, &r))
	require.NoError(t, rbac.Insert(context.TODO(), db, &r))

	c := sdk.AuthConsumer{
		AuthConsumerUser: &sdk.AuthConsumerUser{
			AuthentifiedUserID: user1.ID,
			AuthentifiedUser:   user1,
		},
	}
	require.NoError(t, rbac.RegionRead(context.TODO(), &c, cache, db, map[string]string{"regionIdentifier": reg.Name}))
}
