package api

import (
	"context"
	"fmt"
	"testing"

	"github.com/rockbears/yaml"

	"github.com/ovh/cds/engine/api/organization"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func TestRbacRegionAllUsers(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, err := db.Exec("DELETE FROM rbac")
	require.NoError(t, err)
	_, err = db.Exec("DELETE FROM region")
	require.NoError(t, err)

	org := sdk.Organization{
		Name: sdk.RandomString(10),
	}
	require.NoError(t, organization.Insert(context.TODO(), db, &org))
	t.Cleanup(func() {
		organization.Delete(db, org.ID)
	})

	user1, _ := assets.InsertLambdaUser(t, db)

	user2, _ := assets.InsertLambdaUserInOrganization(t, db, org.Name)

	reg := sdk.Region{Name: sdk.RandomString(10)}
	require.NoError(t, region.Insert(context.TODO(), db, &reg))

	rbacYaml := `name: perm-%s
regions:
- role: %s
  region: %s
  all_users: true
  organizations: [default]`

	rbacYaml = fmt.Sprintf(rbacYaml, reg.Name, sdk.RegionRoleList, reg.Name)
	var r sdk.RBAC
	require.NoError(t, yaml.Unmarshal([]byte(rbacYaml), &r))

	rbacLoader := NewRBACLoader(api.mustDB())
	require.NoError(t, rbacLoader.FillRBACWithIDs(context.TODO(), &r))
	require.NoError(t, rbac.Insert(context.TODO(), db, &r))

	c := sdk.AuthUserConsumer{
		AuthConsumerUser: sdk.AuthUserConsumerData{
			AuthentifiedUserID: user1.ID,
			AuthentifiedUser:   user1,
		},
	}
	require.NoError(t, api.regionRead(context.WithValue(context.TODO(), contextUserConsumer, &c),
		map[string]string{"regionIdentifier": reg.Name}))

	c2 := sdk.AuthUserConsumer{
		AuthConsumerUser: sdk.AuthUserConsumerData{
			AuthentifiedUserID: user2.ID,
			AuthentifiedUser:   user2,
		},
	}
	err = api.regionRead(context.WithValue(context.TODO(), contextUserConsumer, &c2), map[string]string{"regionIdentifier": reg.Name})
	require.True(t, sdk.ErrorIs(err, sdk.ErrForbidden))
}
