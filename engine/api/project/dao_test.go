package project_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func TestInsertProject(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	project.Delete(db, cache, "key")

	proj := sdk.Project{
		Name: "test proj",
		Key:  "key",
	}
	assert.NoError(t, project.Insert(db, cache, &proj))
}

func TestInsertProject_withWrongKey(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	proj := sdk.Project{
		Name: "test proj",
		Key:  "error key",
	}

	assert.Error(t, project.Insert(db, cache, &proj))
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

func TestLoadAllByRepo(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	event.Initialize(event.KafkaConfig{}, cache)

	app, _ := application.LoadByName(db, cache, "TestLoadAllByRepo", "TestLoadAllByRepo")
	if app != nil {
		application.DeleteApplication(db, app.ID)
	}
	project.Delete(db, cache, "TestLoadAllByRepo")
	defer project.Delete(db, cache, "TestLoadAllByRepo")
	proj := sdk.Project{
		Key:  "TestLoadAllByRepo",
		Name: "TestLoadAllByRepo",
	}

	g := sdk.Group{
		Name: "test_TestLoadAll_group",
	}

	eg, _ := group.LoadByName(context.TODO(), db, g.Name)
	if eg != nil {
		g = *eg
	} else if err := group.InsertGroup(db, &g); err != nil {
		t.Fatalf("Cannot insert group : %s", err)
	}

	app = &sdk.Application{
		Name:               "TestLoadAllByRepo",
		RepositoryFullname: "ovh/cds",
	}

	test.NoError(t, project.Insert(db, cache, &proj))
	test.NoError(t, group.InsertGroupInProject(db, proj.ID, g.ID, sdk.PermissionReadWriteExecute))
	test.NoError(t, group.LoadGroupByProject(db, &proj))

	u, _ := assets.InsertLambdaUser(db, &proj.ProjectGroups[0].Group)

	test.NoError(t, application.Insert(db, cache, &proj, app))

	projs, err := project.LoadAllByRepoAndGroupIDs(db, cache, u.GetGroupIDs(), "ovh/cds")
	assert.NoError(t, err)
	assert.Len(t, projs, 1)
}

func TestLoadAll(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

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

	eg, _ := group.LoadByName(context.TODO(), db, g.Name)
	if eg != nil {
		g = *eg
	} else if err := group.InsertGroup(db, &g); err != nil {
		t.Fatalf("Cannot insert group : %s", err)
	}

	test.NoError(t, project.Insert(db, cache, &proj))
	test.NoError(t, project.Insert(db, cache, &proj1))
	test.NoError(t, group.InsertGroupInProject(db, proj.ID, g.ID, sdk.PermissionReadWriteExecute))
	test.NoError(t, group.LoadGroupByProject(db, &proj))

	u2, _ := assets.InsertLambdaUser(db, &proj.ProjectGroups[0].Group)

	actualGroups1, err := project.LoadAll(nil, db, cache)
	test.NoError(t, err)
	assert.True(t, len(actualGroups1) > 1, "This should return more than one project")

	for _, p := range actualGroups1 {
		if p.Name == "test_TestLoadAll" {
			assert.EqualValues(t, proj.Metadata, p.Metadata)
		}
	}

	actualGroups2, err := project.LoadAllByGroupIDs(nil, db, cache, u2.GetGroupIDs())
	t.Log(actualGroups2)
	test.NoError(t, err)
	assert.True(t, len(actualGroups2) == 1, "This should return one project")

	ok, err := project.Exist(db, "test_TestLoadAll")
	test.NoError(t, err)
	assert.True(t, ok)

	project.Delete(db, cache, "test_TestLoadAll")
	project.Delete(db, cache, "test_TestLoadAll1")

}
