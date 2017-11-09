package project_test

import (
	"testing"

	"github.com/go-gorp/gorp"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

func TestInsertProject(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
	project.Delete(db, cache, "key")

	u, _ := assets.InsertAdminUser(db)

	proj := sdk.Project{
		Name: "test proj",
		Key:  "key",
	}
	assert.NoError(t, project.Insert(db, cache, &proj, u))
}

func TestInsertProject_withWrongKey(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
	u, _ := assets.InsertAdminUser(db)

	proj := sdk.Project{
		Name: "test proj",
		Key:  "error key",
	}

	assert.Error(t, project.Insert(db, cache, &proj, u))
}

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
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	project.Delete(db, cache, "test_TestLoadAll")
	project.Delete(db, cache, "test_TestLoadAll1")

	proj := sdk.Project{
		Key:  "test_TestLoadAll",
		Name: "test_TestLoadAll",
		Metadata: map[string]string{
			"data1": "value1",
			"data2": "value2",
		},
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

	test.NoError(t, project.Insert(db, cache, &proj, nil))
	test.NoError(t, project.Insert(db, cache, &proj1, nil))
	test.NoError(t, group.InsertGroupInProject(db, proj.ID, g.ID, permission.PermissionReadWriteExecute))
	test.NoError(t, group.LoadGroupByProject(db, &proj))

	user.DeleteUserWithDependenciesByName(db, "test_TestLoadAll_admin")
	user.DeleteUserWithDependenciesByName(db, "test_TestLoadAll_user")

	u1, _ := InsertAdminUser(t, db, "test_TestLoadAll_admin")
	u2, _ := InsertLambdaUser(t, db, "test_TestLoadAll_user", &proj.ProjectGroups[0].Group)

	actualGroups1, err := project.LoadAll(db, cache, u1)
	test.NoError(t, err)
	assert.True(t, len(actualGroups1) > 1, "This should return more than one project")

	for _, p := range actualGroups1 {
		if p.Name == "test_TestLoadAll" {
			t.Log(p)
			assert.EqualValues(t, proj.Metadata, p.Metadata)
		}
	}

	actualGroups2, err := project.LoadAll(db, cache, u2)
	test.NoError(t, err)
	assert.True(t, len(actualGroups2) == 1, "This should return one project")

	ok, err := project.Exist(db, "test_TestLoadAll")
	test.NoError(t, err)
	assert.True(t, ok)

	project.Delete(db, cache, "test_TestLoadAll")
	project.Delete(db, cache, "test_TestLoadAll1")

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

// InsertLambdaUser have to be used only for tests
func InsertLambdaUser(t *testing.T, db gorp.SqlExecutor, s string, groups ...*sdk.Group) (*sdk.User, string) {
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
		u.Groups = append(u.Groups, *g)
	}
	return u, password
}
