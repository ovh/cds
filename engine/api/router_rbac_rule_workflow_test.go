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

func TestHasRoleWorkflowExecute(t *testing.T) {
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

	targetWorkflow := "my-workflow"

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcs := assets.InsertTestVCSProject(t, db, proj.ID, "myvcs", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcs.ID, "my/repo")

	tests := []struct {
		name   string
		rabc   string
		result bool
	}{
		{
			name: "user has direct right",
			rabc: fmt.Sprintf(`name: test-perm
workflows:
- role: trigger
  users: [%s]
  workflows: ["**/my-workflow"]
  project: %s`, user1.Username, proj.Key),
			result: true,
		},

		{
			name: "user has right through a group",
			rabc: fmt.Sprintf(`name: test-perm
workflows:
- role: trigger
  groups: [%s]
  workflows: [myvcs/**/my-workflow]
  project: %s`, g.Name, proj.Key),
			result: true,
		},

		{
			name: "all workflows are allowed on project",
			rabc: fmt.Sprintf(`name: test-perm
workflows:
- role: trigger
  groups: [%s]
  all_workflows: true
  project: %s`, g.Name, proj.Key),
			result: true,
		},

		{
			name: "all users are allowed on project",
			rabc: fmt.Sprintf(`name: test-perm
workflows:
- role: trigger
  all_users: true
  all_workflows: true
  project: %s`, proj.Key),
			result: true,
		},

		{
			name: "user does not have the right",
			rabc: fmt.Sprintf(`name: test-perm
workflows:
- role: trigger
  users: [%s]
  all_workflows: true
  project: %s`, user2.Username, proj.Key),
			result: false,
		},
		{
			name: "user has nodirect right - missing vcs and repo",
			rabc: fmt.Sprintf(`name: test-perm
workflows:
- role: trigger
  users: [%s]
  workflows: [my-workflow]
  project: %s`, user1.Username, proj.Key),
			result: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.Exec("DELETE FROM rbac")
			require.NoError(t, err)

			var r sdk.RBAC
			require.NoError(t, yaml.Unmarshal([]byte(tt.rabc), &r))

			rbacLoader := NewRBACLoader(api.mustDB())
			require.NoError(t, rbacLoader.FillRBACWithIDs(context.TODO(), &r))
			require.NoError(t, rbac.Insert(context.TODO(), db, &r))

			err = api.workflowTrigger(context.WithValue(context.TODO(), contextUserConsumer, &auth), map[string]string{
				"projectKey":           proj.Key,
				"vcsIdentifier":        vcs.Name,
				"repositoryIdentifier": repo.Name,
				"workflowName":         targetWorkflow,
			})
			if tt.result {
				require.NoError(t, err)
			} else {
				require.True(t, sdk.ErrorIs(err, sdk.ErrForbidden))
			}

		})
	}
}
