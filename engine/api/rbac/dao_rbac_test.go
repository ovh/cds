package rbac_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func TestLoadRbacProject(t *testing.T) {
	// user1  can read (proj1)
	// Group 1 can read (proj1,prj2), manage(prj2)
	// user2  in group 1
	db, cache := test.SetupPG(t)

	_, err := db.Exec("DELETE FROM rbac")
	require.NoError(t, err)

	key1 := sdk.RandomString(10)
	proj1 := assets.InsertTestProject(t, db, cache, key1, key1)

	key2 := sdk.RandomString(10)
	proj2 := assets.InsertTestProject(t, db, cache, key2, key2)

	grpName1 := sdk.RandomString(10)
	group1 := assets.InsertTestGroup(t, db, grpName1)

	user1, _ := assets.InsertLambdaUser(t, db)
	user2, _ := assets.InsertLambdaUser(t, db, group1)

	perm := fmt.Sprintf(`name: perm-test
projects:
  - role: read
    projects: [%s]
    users: [%s]
    groups: [%s]
  - role: read
    projects: [%s]
    groups: [%s]
  - role: manage
    projects: [%s]
    groups: [%s]
`, proj1.Key, user1.Username, group1.Name, proj2.Key, group1.Name, proj2.Key, group1.Name)

	var r sdk.RBAC
	require.NoError(t, yaml.Unmarshal([]byte(perm), &r))

	r.Projects[0].RBACUsersIDs = []string{user1.ID}
	r.Projects[0].RBACGroupsIDs = []int64{group1.ID}
	r.Projects[1].RBACGroupsIDs = []int64{group1.ID}
	r.Projects[2].RBACGroupsIDs = []int64{group1.ID}

	require.NoError(t, rbac.Insert(context.Background(), db, &r))

	projectKeysForUser1, err := rbac.LoadAllProjectKeysAllowed(context.TODO(), db, sdk.ProjectRoleRead, user1.ID)
	require.NoError(t, err)
	t.Logf("%+v", projectKeysForUser1)
	require.Len(t, projectKeysForUser1, 1)
	require.Equal(t, projectKeysForUser1[0], proj1.Key)

	projectKeysForUser2, err := rbac.LoadAllProjectKeysAllowed(context.TODO(), db, sdk.ProjectRoleManage, user2.ID)
	require.NoError(t, err)
	require.Len(t, projectKeysForUser2, 1)
	require.Equal(t, projectKeysForUser2[0], proj2.Key)
}

func TestLoadRbac(t *testing.T) {
	db, cache := test.SetupPG(t)

	_, err := db.Exec("DELETE FROM rbac WHERE name = 'perm-test'")
	require.NoError(t, err)

	key1 := sdk.RandomString(10)
	proj1 := assets.InsertTestProject(t, db, cache, key1, key1)

	key2 := sdk.RandomString(10)
	proj2 := assets.InsertTestProject(t, db, cache, key2, key2)

	grpName1 := sdk.RandomString(10)
	group1 := assets.InsertTestGroup(t, db, grpName1)

	user1, _ := assets.InsertLambdaUser(t, db)

	perm := fmt.Sprintf(`name: perm-test
projects:
  - role: read
    users: [%s]
    groups: [%s]
    projects: [%s]
  - role: manage
    groups: [%s]
    projects: [%s]
global:
  - role: create-project
    users: [%s]
`, user1.Username, group1.Name, proj1.Key, group1.Name, proj2.Key, user1.Username)

	var r sdk.RBAC
	require.NoError(t, yaml.Unmarshal([]byte(perm), &r))

	r.Projects[0].RBACUsersIDs = []string{user1.ID}
	r.Projects[0].RBACGroupsIDs = []int64{group1.ID}
	r.Projects[1].RBACGroupsIDs = []int64{group1.ID}
	r.Global[0].RBACUsersIDs = []string{user1.ID}

	require.NoError(t, rbac.Insert(context.Background(), db, &r))

	rbacDB, err := rbac.LoadRBACByName(context.TODO(), db, r.Name, rbac.LoadOptions.Default)
	require.NoError(t, err)

	// Global part
	require.Equal(t, len(r.Global), len(rbacDB.Global))
	require.Equal(t, r.Global[0].Role, rbacDB.Global[0].Role)
	require.Equal(t, user1.ID, rbacDB.Global[0].RBACUsersIDs[0])

	// Project part
	require.Equal(t, len(r.Projects), len(rbacDB.Projects))

	manageCheck := false
	readCheck := false
	for _, rp := range r.Projects {
		if rp.Role == "manage" {
			require.Equal(t, 1, len(rp.RBACGroupsName))
			require.Equal(t, 1, len(rp.RBACGroupsIDs))
			require.Equal(t, 1, len(rp.RBACProjectKeys))
			require.Equal(t, proj2.Key, rp.RBACProjectKeys[0])
			require.Equal(t, group1.Name, rp.RBACGroupsName[0])
			require.Equal(t, group1.ID, rp.RBACGroupsIDs[0])
			manageCheck = true
		}
		if rp.Role == "read" {
			require.Equal(t, 1, len(rp.RBACGroupsName))
			require.Equal(t, 1, len(rp.RBACGroupsIDs))
			require.Equal(t, 1, len(rp.RBACUsersIDs))
			require.Equal(t, 1, len(rp.RBACUsersName))
			require.Equal(t, 1, len(rp.RBACProjectKeys))
			require.Equal(t, proj1.Key, rp.RBACProjectKeys[0])
			require.Equal(t, group1.Name, rp.RBACGroupsName[0])
			require.Equal(t, group1.ID, rp.RBACGroupsIDs[0])
			require.Equal(t, user1.Username, rp.RBACUsersName[0])
			require.Equal(t, user1.ID, rp.RBACUsersIDs[0])
			readCheck = true
		}
	}
	require.True(t, manageCheck)
	require.True(t, readCheck)
}

func TestUpdateRbac(t *testing.T) {
	db, cache := test.SetupPG(t)

	_, err := db.Exec("DELETE FROM rbac WHERE name = 'perm-test'")
	require.NoError(t, err)

	key1 := sdk.RandomString(10)
	proj1 := assets.InsertTestProject(t, db, cache, key1, key1)

	user1, _ := assets.InsertLambdaUser(t, db)

	perm := fmt.Sprintf(`name: perm-test
projects:
  - role: read
    users: [%s]
    projects: [%s]

`, user1.Username, proj1.Key)

	var r sdk.RBAC
	require.NoError(t, yaml.Unmarshal([]byte(perm), &r))

	r.Projects[0].RBACUsersIDs = []string{user1.ID}

	require.NoError(t, rbac.Insert(context.Background(), db, &r))

	rbacDB, err := rbac.LoadRBACByName(context.TODO(), db, r.Name, rbac.LoadOptions.Default)
	require.NoError(t, err)

	require.Equal(t, "read", rbacDB.Projects[0].Role)

	// Update change role
	permUpdated := fmt.Sprintf(`name: perm-test
projects:
  - role: manage
    users: [%s]
    projects: [%s]

`, user1.Username, proj1.Key)

	var rUpdated sdk.RBAC
	require.NoError(t, yaml.Unmarshal([]byte(permUpdated), &rUpdated))

	rUpdated.ID = rbacDB.ID
	rUpdated.Projects[0].RBACUsersIDs = []string{user1.ID}

	require.NoError(t, rbac.Update(context.TODO(), db, &rUpdated))

	rbacDBUpdate, err := rbac.LoadRBACByName(context.TODO(), db, r.Name, rbac.LoadOptions.Default)
	require.NoError(t, err)
	require.Equal(t, "manage", rbacDBUpdate.Projects[0].Role)

}
