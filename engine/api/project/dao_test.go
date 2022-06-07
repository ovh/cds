package project_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	project.Delete(db, "key")

	proj := sdk.Project{
		Name: "test proj",
		Key:  "key",
	}
	assert.NoError(t, project.Insert(db, &proj))
}

func TestInsertProject_withWrongKey(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	proj := sdk.Project{
		Name: "test proj",
		Key:  "error key",
	}

	assert.Error(t, project.Insert(db, &proj))
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
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	_ = event.Initialize(context.Background(), db.DbMap, cache, nil)

	proj := &sdk.Project{
		Key:  sdk.RandomString(10),
		Name: sdk.RandomString(10),
	}
	require.NoError(t, project.Insert(db, proj))

	g := assets.InsertGroup(t, db)
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionReadWriteExecute,
	}))

	app := &sdk.Application{
		Name:               sdk.RandomString(10),
		RepositoryFullname: "ovh/cds",
	}
	test.NoError(t, application.Insert(db, *proj, app))

	projs, err := project.LoadAllByRepoAndGroupIDs(context.TODO(), db, []int64{g.ID}, "ovh/cds")
	require.NoError(t, err)
	require.Len(t, projs, 1)
	assert.Equal(t, proj.ID, projs[0].ID)
}

func TestLoadAll(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	project.Delete(db, "test_TestLoadAll1")
	project.Delete(db, "test_TestLoadAll2")

	proj1 := &sdk.Project{
		Key:  "test_TestLoadAll1",
		Name: "test_TestLoadAll1",
		Metadata: map[string]string{
			"data1": "value1",
			"data2": "value2",
		},
	}
	require.NoError(t, project.Insert(db, proj1))

	proj2 := sdk.Project{
		Key:  "test_TestLoadAll2",
		Name: "test_TestLoadAll2",
	}
	require.NoError(t, project.Insert(db, &proj2))

	g := sdk.Group{Name: sdk.RandomString(10)}
	require.NoError(t, group.Insert(context.TODO(), db, &g))

	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g.ID,
		ProjectID: proj1.ID,
		Role:      sdk.PermissionReadWriteExecute,
	}))

	proj1, _ = project.LoadByID(db, proj1.ID, project.LoadOptions.WithGroups)

	allProjects, err := project.LoadAll(nil, db, cache)
	require.NoError(t, err)
	assert.True(t, len(allProjects) > 1, "This should return more than one project")
	var foundProj1, foundProj2 bool
	for _, p := range allProjects {
		if p.Name == proj1.Name {
			foundProj1 = true
		}
		if p.Name == proj2.Name {
			foundProj2 = true
		}
		if p.Name == "test_TestLoadAll1" {
			assert.EqualValues(t, proj1.Metadata, p.Metadata)
		}
	}
	assert.True(t, foundProj1, "Project 1 should be in list")
	assert.True(t, foundProj2, "Project 2 should be in list")

	groupProjects, err := project.LoadAllByGroupIDs(context.TODO(), db, cache, []int64{g.ID})
	require.NoError(t, err)
	assert.True(t, len(groupProjects) == 1, "This should return only one project")
	assert.Equal(t, proj1.Name, groupProjects[0].Name)

	ok, err := project.Exist(db, "test_TestLoadAll1")
	require.NoError(t, err)
	assert.True(t, ok)

	assert.NoError(t, project.Delete(db, "test_TestLoadAll1"))
	assert.NoError(t, project.Delete(db, "test_TestLoadAll2"))
}
