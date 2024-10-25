package api

import (
	"context"
	"fmt"
	"github.com/rockbears/yaml"
	"testing"

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

	user1, _ := assets.InsertLambdaUser(t, db)

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
	require.NoError(t, api.regionRead(context.TODO(), &c, api.Cache, db, map[string]string{"regionIdentifier": reg.Name}))
}
