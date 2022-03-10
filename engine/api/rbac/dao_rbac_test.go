package rbac

import (
	"context"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLoadRbacProject(t *testing.T) {
	// user1  can read (proj1)
	// Group 1 can read (proj1,prj2), manage(prj2)
	// user2  in group 1

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
	users2, _ := assets.InsertLambdaUser(t, db, group1)

	r := sdk.Rbac{
		Name: "perm-test",
		Projects: []sdk.RbacProject{
			{
				Projects: []sdk.RbacProjectIdentifiers{
					{
						ProjectID: proj1.ID,
					},
				},
				AbstractRbac: sdk.AbstractRbac{
					Role: sdk.RoleRead,
					RbacUsers: []sdk.RbacUser{
						{
							UserID: users1.ID,
						},
					},
					RbacGroups: []sdk.RbacGroup{
						{
							GroupID: group1.ID,
						},
					},
				},
			},
			{
				Projects: []sdk.RbacProjectIdentifiers{
					{
						ProjectID: proj2.ID,
					},
				},
				AbstractRbac: sdk.AbstractRbac{
					Role: sdk.RoleRead,
					RbacGroups: []sdk.RbacGroup{
						{
							GroupID: group1.ID,
						},
					},
				},
			},
			{
				Projects: []sdk.RbacProjectIdentifiers{
					{
						ProjectID: proj2.ID,
					},
				},
				AbstractRbac: sdk.AbstractRbac{
					Role: sdk.RoleManage,
					RbacGroups: []sdk.RbacGroup{
						{
							GroupID: group1.ID,
						},
					},
				},
			},
		},
	}
	require.NoError(t, Insert(context.Background(), db, &r))

	prjusers1, err := LoadRbacProjectIDsByUserID(context.TODO(), db, sdk.RoleRead, users1.ID)
	require.NoError(t, err)
	require.Len(t, prjusers1, 1)
	require.Equal(t, prjusers1[0].ID, proj1.ID)

	prjusers2, err := LoadRbacProjectIDsByUserID(context.TODO(), db, sdk.RoleManage, users2.ID)
	require.NoError(t, err)
	require.Len(t, prjusers2, 1)
	require.Equal(t, prjusers2[0].ID, proj2.ID)
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

	r := sdk.Rbac{
		Name: "perm-test",
		Projects: []sdk.RbacProject{
			{
				Projects: []sdk.RbacProjectIdentifiers{
					{
						ProjectID: proj1.ID,
					},
				},
				AbstractRbac: sdk.AbstractRbac{
					Role: sdk.RoleRead,
					RbacUsers: []sdk.RbacUser{
						{
							UserID: users1.ID,
						},
					},
					RbacGroups: []sdk.RbacGroup{
						{
							GroupID: group1.ID,
						},
					},
				},
			},
			{
				Projects: []sdk.RbacProjectIdentifiers{
					{
						ProjectID: proj2.ID,
					},
				},
				AbstractRbac: sdk.AbstractRbac{
					Role: sdk.RoleRead,
					RbacGroups: []sdk.RbacGroup{
						{
							GroupID: group1.ID,
						},
					},
				},
			},
			{
				Projects: []sdk.RbacProjectIdentifiers{
					{
						ProjectID: proj2.ID,
					},
				},
				AbstractRbac: sdk.AbstractRbac{
					Role: sdk.RoleManage,
					RbacGroups: []sdk.RbacGroup{
						{
							GroupID: group1.ID,
						},
					},
				},
			},
		},
		Globals: []sdk.RbacGlobal{
			{
				AbstractRbac: sdk.AbstractRbac{
					Role: sdk.RoleCreateProject,
					RbacUsers: []sdk.RbacUser{
						{
							UserID: users1.ID,
						},
					},
				},
			},
		},
	}
	require.NoError(t, Insert(context.Background(), db, &r))

	rbacDB, err := LoadRbacByName(context.TODO(), db, r.Name, LoadOptions.Default)
	require.NoError(t, err)
	require.Equal(t, len(r.Globals), len(rbacDB.Globals))
	require.Equal(t, r.Globals[0].Role, rbacDB.Globals[0].Role)
	require.Equal(t, len(r.Globals[0].RbacUsers), len(rbacDB.Globals[0].RbacUsers))
	require.Equal(t, len(r.Projects), len(rbacDB.Projects))
	require.Equal(t, len(r.Projects), len(rbacDB.Projects))
}
