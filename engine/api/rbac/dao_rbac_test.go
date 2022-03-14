package rbac

import (
	"context"
	"fmt"
	"github.com/ovh/cds/engine/api/project"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

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

	users1, _ := assets.InsertLambdaUser(t, db)
	users2, _ := assets.InsertLambdaUser(t, db, group1)

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
`, proj1.Key, users1.Username, group1.Name, proj2.Key, group1.Name, proj2.Key, group1.Name)

	var r sdk.Rbac
	require.NoError(t, yaml.Unmarshal([]byte(perm), &r))

	err = fillWithIDs(context.TODO(), db, &r)
	require.NoError(t, err)

	require.NoError(t, Insert(context.Background(), db, &r))

	prjusers1, err := project.LoadRbacProjectIDsByUserID(context.TODO(), db, sdk.RoleRead, users1.ID)
	require.NoError(t, err)
	t.Logf("%+v", prjusers1)
	require.Len(t, prjusers1, 1)
	require.Equal(t, prjusers1[0].ID, proj1.ID)
	require.Equal(t, prjusers1[0].Key, proj1.Key)

	prjusers2, err := project.LoadRbacProjectIDsByUserID(context.TODO(), db, sdk.RoleManage, users2.ID)
	require.NoError(t, err)
	require.Len(t, prjusers2, 1)
	require.Equal(t, prjusers2[0].ID, proj2.ID)
	require.Equal(t, prjusers2[0].Key, proj2.Key)
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

	users1, _ := assets.InsertLambdaUser(t, db)

	perm := fmt.Sprintf(`name: perm-test
projects:
  - role: read
    users: [%s]
    groups: [%s]
    projects: [%s]
  - role: manage
    groups: [%s]
    projects: [%s]
globals:
  - role: create-project
    users: [%s]
`, users1.Username, group1.Name, proj1.Key, group1.Name, proj2.Key, users1.Username)

	var r sdk.Rbac
	require.NoError(t, yaml.Unmarshal([]byte(perm), &r))
	err = fillWithIDs(context.TODO(), db, &r)
	require.NoError(t, err)

	require.NoError(t, Insert(context.Background(), db, &r))

	rbacDB, err := LoadRbacByName(context.TODO(), db, r.Name, LoadOptions.Default)
	require.NoError(t, err)

	// Global part
	require.Equal(t, len(r.Globals), len(rbacDB.Globals))
	require.Equal(t, r.Globals[0].Role, rbacDB.Globals[0].Role)
	require.Equal(t, len(r.Globals[0].RbacUsersName), len(rbacDB.Globals[0].RbacUsersName))
	require.Equal(t, users1.Username, rbacDB.Globals[0].RbacUsersName[0])
	require.Equal(t, users1.ID, rbacDB.Globals[0].RbacUsersIDs[0])
	require.Equal(t, len(r.Globals[0].RbacUsersName), len(rbacDB.Globals[0].RbacUsersName))

	// Project part
	require.Equal(t, len(r.Projects), len(rbacDB.Projects))

	manageCheck := false
	readCheck := false
	for _, rp := range r.Projects {
		if rp.Role == "manage" {
			require.Equal(t, 1, len(rp.RbacGroupsName))
			require.Equal(t, 1, len(rp.RbacGroupsIDs))
			require.Equal(t, 1, len(rp.RbacProjectKeys))
			require.Equal(t, 1, len(rp.RbacProjectsIDs))
			require.Equal(t, proj2.Key, rp.RbacProjectKeys[0])
			require.Equal(t, proj2.ID, rp.RbacProjectsIDs[0])
			require.Equal(t, group1.Name, rp.RbacGroupsName[0])
			require.Equal(t, group1.ID, rp.RbacGroupsIDs[0])
			manageCheck = true
		}
		if rp.Role == "read" {
			require.Equal(t, 1, len(rp.RbacGroupsName))
			require.Equal(t, 1, len(rp.RbacGroupsIDs))
			require.Equal(t, 1, len(rp.RbacUsersIDs))
			require.Equal(t, 1, len(rp.RbacUsersName))
			require.Equal(t, 1, len(rp.RbacProjectKeys))
			require.Equal(t, 1, len(rp.RbacProjectsIDs))
			require.Equal(t, proj1.Key, rp.RbacProjectKeys[0])
			require.Equal(t, proj1.ID, rp.RbacProjectsIDs[0])
			require.Equal(t, group1.Name, rp.RbacGroupsName[0])
			require.Equal(t, group1.ID, rp.RbacGroupsIDs[0])
			require.Equal(t, users1.Username, rp.RbacUsersName[0])
			require.Equal(t, users1.ID, rp.RbacUsersIDs[0])
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

	users1, _ := assets.InsertLambdaUser(t, db)

	perm := fmt.Sprintf(`name: perm-test
projects:
  - role: read
    users: [%s]
    projects: [%s]

`, users1.Username, proj1.Key)

	var r sdk.Rbac
	require.NoError(t, yaml.Unmarshal([]byte(perm), &r))
	err = fillWithIDs(context.TODO(), db, &r)
	require.NoError(t, err)

	require.NoError(t, Insert(context.Background(), db, &r))

	rbacDB, err := LoadRbacByName(context.TODO(), db, r.Name, LoadOptions.Default)
	require.NoError(t, err)

	require.Equal(t, "read", rbacDB.Projects[0].Role)

	// Update change role
	permUpdated := fmt.Sprintf(`name: perm-test
projects:
  - role: manage
    users: [%s]
    projects: [%s]

`, users1.Username, proj1.Key)

	var rUpdated sdk.Rbac
	require.NoError(t, yaml.Unmarshal([]byte(permUpdated), &rUpdated))
	err = fillWithIDs(context.TODO(), db, &rUpdated)
	require.NoError(t, err)

	require.NoError(t, Update(context.TODO(), db, &rUpdated))

	rbacDBUpdate, err := LoadRbacByName(context.TODO(), db, r.Name, LoadOptions.Default)
	require.NoError(t, err)
	require.Equal(t, "manage", rbacDBUpdate.Projects[0].Role)

}
