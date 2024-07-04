package api

import (
	"context"
	"fmt"
	"testing"

	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/rockbears/yaml"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/sdk"
)

func TestHasRoleVariableSetExecute(t *testing.T) {
	api, db, _ := newTestAPI(t)

	g := assets.InsertGroup(t, db)

	user1, _ := assets.InsertLambdaUser(t, db, g)
	user2, _ := assets.InsertLambdaUser(t, db, g)
	auth := sdk.AuthUserConsumer{
		AuthConsumerUser: sdk.AuthUserConsumerData{
			AuthentifiedUser: &sdk.AuthentifiedUser{
				ID: user1.ID,
			},
		},
	}

	targetVariableSet := "vs1"

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	tests := []struct {
		name   string
		rbac   string
		result bool
	}{
		{
			name: "user has direct right",
			rbac: fmt.Sprintf(`name: test-perm
variablesets:
- role: manage
  users: [%s]
  variablesets: [vs1]
  project: %s`, user1.Username, proj.Key),
			result: true,
		},

		{
			name: "user has right through a group",
			rbac: fmt.Sprintf(`name: test-perm
variablesets:
- role: manage
  groups: [%s]
  variablesets: [vs1]
  project: %s`, g.Name, proj.Key),
			result: true,
		},

		{
			name: "all variablesets are allowed on project",
			rbac: fmt.Sprintf(`name: test-perm
variablesets:
- role: manage
  groups: [%s]
  all_variablesets: true
  project: %s`, g.Name, proj.Key),
			result: true,
		},

		{
			name: "all users are allowed on project",
			rbac: fmt.Sprintf(`name: test-perm
variablesets:
- role: manage
  all_users: true
  all_variablesets: true
  project: %s`, proj.Key),
			result: true,
		},

		{
			name: "user does not have the right",
			rbac: fmt.Sprintf(`name: test-perm
variablesets:
- role: manage
  users: [%s]
  all_workflows: true
  project: %s`, user2.Username, proj.Key),
			result: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.Exec("DELETE FROM rbac")
			require.NoError(t, err)

			var r sdk.RBAC
			require.NoError(t, yaml.Unmarshal([]byte(tt.rbac), &r))

			rbacLoader := NewRBACLoader(api.mustDB())
			require.NoError(t, rbacLoader.FillRBACWithIDs(context.TODO(), &r))
			require.NoError(t, rbac.Insert(context.TODO(), db, &r))

			err = api.variableSetManage(context.TODO(), &auth, api.Cache, api.mustDB(), map[string]string{
				"projectKey":      proj.Key,
				"variableSetName": targetVariableSet,
			})
			if tt.result {
				require.NoError(t, err)
			} else {
				require.True(t, sdk.ErrorIs(err, sdk.ErrForbidden))
			}

		})
	}
}
