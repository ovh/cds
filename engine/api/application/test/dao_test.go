package test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func TestLoadByNameAsAdmin(t *testing.T) {
	db := test.SetupPG(t)
	key := assets.RandomString(t, 10)
	proj := assets.InsertTestProject(t, db, key, key)
	app := sdk.Application{
		Name: "my-app",
	}

	test.NoError(t, application.Insert(db, proj, &app))

	actual, err := application.LoadByName(db, key, "my-app", nil)
	test.NoError(t, err)

	assert.Equal(t, app.Name, actual.Name)
	assert.Equal(t, proj.ID, actual.ProjectID)
	assert.Equal(t, proj.Key, actual.ProjectKey)
}

func TestLoadByNameAsUser(t *testing.T) {
	db := test.SetupPG(t)
	key := assets.RandomString(t, 10)
	proj := assets.InsertTestProject(t, db, key, key)
	app := sdk.Application{
		Name: "my-app",
	}

	test.NoError(t, application.Insert(db, proj, &app))

	u, _ := assets.InsertLambaUser(t, db, &proj.ProjectGroups[0].Group)

	test.NoError(t, application.AddGroup(db, proj, &app, proj.ProjectGroups...))

	actual, err := application.LoadByName(db, key, "my-app", u)
	assert.NoError(t, err)

	assert.Equal(t, app.Name, actual.Name)
	assert.Equal(t, proj.ID, actual.ProjectID)
	assert.Equal(t, proj.Key, actual.ProjectKey)
}

func TestLoadByIDAsAdmin(t *testing.T) {
	db := test.SetupPG(t)
	key := assets.RandomString(t, 10)
	proj := assets.InsertTestProject(t, db, key, key)
	app := sdk.Application{
		Name: "my-app",
	}

	test.NoError(t, application.Insert(db, proj, &app))

	actual, err := application.LoadByID(db, app.ID, nil)
	test.NoError(t, err)

	assert.Equal(t, app.Name, actual.Name)
	assert.Equal(t, proj.ID, actual.ProjectID)
	assert.Equal(t, proj.Key, actual.ProjectKey)
}

func TestLoadByIDAsUser(t *testing.T) {
	db := test.SetupPG(t)
	key := assets.RandomString(t, 10)
	proj := assets.InsertTestProject(t, db, key, key)
	app := sdk.Application{
		Name: "my-app",
	}

	test.NoError(t, application.Insert(db, proj, &app))

	u, _ := assets.InsertLambaUser(t, db, &proj.ProjectGroups[0].Group)

	test.NoError(t, application.AddGroup(db, proj, &app, proj.ProjectGroups...))

	actual, err := application.LoadByID(db, app.ID, u)
	assert.NoError(t, err)

	assert.Equal(t, app.Name, actual.Name)
	assert.Equal(t, proj.ID, actual.ProjectID)
	assert.Equal(t, proj.Key, actual.ProjectKey)
}

func TestLoadAllAsAdmin(t *testing.T) {
	db := test.SetupPG(t)
	key := assets.RandomString(t, 10)
	proj := assets.InsertTestProject(t, db, key, key)
	app := sdk.Application{
		Name: "my-app",
		Metadata: sdk.Metadata{
			"bla": "bla",
		},
	}

	app2 := sdk.Application{
		Name: "my-app2",
		Metadata: sdk.Metadata{
			"bla": "bla",
		},
	}

	test.NoError(t, application.Insert(db, proj, &app))
	test.NoError(t, application.Insert(db, proj, &app2))

	actual, err := application.LoadAll(db, proj.Key, nil)
	test.NoError(t, err)

	assert.Equal(t, 2, len(actual))

	for _, a := range actual {
		assert.EqualValues(t, app.Metadata, a.Metadata)
		assert.EqualValues(t, app2.Metadata, a.Metadata)
	}
}

func TestLoadAllAsUser(t *testing.T) {
	db := test.SetupPG(t)
	key := assets.RandomString(t, 10)
	proj := assets.InsertTestProject(t, db, key, key)
	app := sdk.Application{
		Name: "my-app",
	}

	app2 := sdk.Application{
		Name: "my-app2",
	}

	test.NoError(t, application.Insert(db, proj, &app))
	test.NoError(t, application.Insert(db, proj, &app2))

	u, _ := assets.InsertLambaUser(t, db, &proj.ProjectGroups[0].Group)

	test.NoError(t, application.AddGroup(db, proj, &app, proj.ProjectGroups...))

	actual, err := application.LoadAll(db, proj.Key, u)
	test.NoError(t, err)

	assert.Equal(t, 1, len(actual))
}
