package project

import (
	"testing"

	"github.com/go-gorp/gorp"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

func TestDelete(t *testing.T) {
	//covered by TestLoadAll
}

func TestDeleteByID(t *testing.T) {
	//covered by TestLoadAll
}

func TestExist(t *testing.T) {
	//covered by TestLoadAll
}

func TestLoadAll(t *testing.T) {
	db := test.SetupPG(t)

	Delete(db, "test_TestLoadAll")
	Delete(db, "test_TestLoadAll1")

	proj := sdk.Project{
		Key:  "test_TestLoadAll",
		Name: "test_TestLoadAll",
	}

	proj1 := sdk.Project{
		Key:  "test_TestLoadAll1",
		Name: "test_TestLoadAll1",
	}

	g := sdk.Group{
		Name: "test_TestLoadAll_group",
	}

	eg, _ := group.LoadGroup(db, g.Name)
	if eg != nil {
		g = *eg
	} else if err := group.InsertGroup(db, &g); err != nil {
		t.Fatalf("Cannot insert group : %s", err)
	}

	test.NoError(t, InsertProject(db, &proj))
	test.NoError(t, InsertProject(db, &proj1))
	test.NoError(t, group.InsertGroupInProject(db, proj.ID, g.ID, permission.PermissionReadWriteExecute))
	test.NoError(t, group.LoadGroupByProject(db, &proj))

	user.DeleteUserWithDependenciesByName(db, "test_TestLoadAll_admin")
	user.DeleteUserWithDependenciesByName(db, "test_TestLoadAll_user")

	u1, _ := InsertAdminUser(t, db, "test_TestLoadAll_admin")
	u2, _ := InsertLambaUser(t, db, "test_TestLoadAll_user", &proj.ProjectGroups[0].Group)

	actualGroups1, err := LoadAll(db, u1)
	test.NoError(t, err)
	assert.True(t, len(actualGroups1) > 1, "This should return more than one project")

	actualGroups2, err := LoadAll(db, u2)
	test.NoError(t, err)
	assert.True(t, len(actualGroups2) == 1, "This should return one project")

	ok, err := Exist(db, "test_TestLoadAll")
	test.NoError(t, err)
	assert.True(t, ok)

	Delete(db, "test_TestLoadAll")
	Delete(db, "test_TestLoadAll1")

}

// InsertAdminUser have to be used only for tests
func InsertAdminUser(t *testing.T, db *gorp.DbMap, s string) (*sdk.User, string) {
	password, hash, _ := user.GeneratePassword()
	u := &sdk.User{
		Admin:    true,
		Email:    "no-reply-" + s + "@corp.ovh.com",
		Username: s,
		Origin:   "local",
		Fullname: "Test " + s,
		Auth: sdk.Auth{
			EmailVerified:  true,
			HashedPassword: hash,
		},
	}
	user.InsertUser(db, u, &u.Auth)
	return u, password
}

// InsertLambaUser have to be used only for tests
func InsertLambaUser(t *testing.T, db gorp.SqlExecutor, s string, groups ...*sdk.Group) (*sdk.User, string) {
	password, hash, _ := user.GeneratePassword()
	u := &sdk.User{
		Admin:    false,
		Email:    "no-reply-" + s + "@corp.ovh.com",
		Username: s,
		Origin:   "local",
		Fullname: "Test " + s,
		Auth: sdk.Auth{
			EmailVerified:  true,
			HashedPassword: hash,
		},
	}
	user.InsertUser(db, u, &u.Auth)
	for _, g := range groups {
		group.InsertGroup(db, g)
		group.InsertUserInGroup(db, g.ID, u.ID, false)
	}
	return u, password
}
