package api

import (
	"context"
	"fmt"
	"testing"

	"github.com/rockbears/yaml"

	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func TestRbacEntityRead(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, err := db.Exec("DELETE FROM rbac")
	require.NoError(t, err)

	user1, _ := assets.InsertLambdaUser(t, db)
	p := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	rbacYaml := `name: perm-%s
projects:
- role: %s
  projects: [%s]
  users: [%s]`

	rbacYaml = fmt.Sprintf(rbacYaml, sdk.RandomString(10), sdk.ProjectRoleRead, p.Key, user1.Username)
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
	require.NoError(t, api.entityRead(context.TODO(), &c, api.Cache, db, map[string]string{"projectKey": p.Key}))

	cNo := sdk.AuthUserConsumer{
		AuthConsumerUser: sdk.AuthUserConsumerData{
			AuthentifiedUserID: "aa",
			AuthentifiedUser: &sdk.AuthentifiedUser{
				ID:       "aa",
				Username: "bb",
			},
		},
	}
	err = api.entityRead(context.TODO(), &cNo, api.Cache, db, map[string]string{"projectKey": p.Key})
	require.True(t, sdk.ErrorIs(err, sdk.ErrForbidden))
}
