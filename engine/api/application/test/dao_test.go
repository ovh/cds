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

	test.NoError(t, application.InsertApplication(db, proj, &app))

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

	test.NoError(t, application.InsertApplication(db, proj, &app))

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

	test.NoError(t, application.InsertApplication(db, proj, &app))

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

	test.NoError(t, application.InsertApplication(db, proj, &app))

	u, _ := assets.InsertLambaUser(t, db, &proj.ProjectGroups[0].Group)

	test.NoError(t, application.AddGroup(db, proj, &app, proj.ProjectGroups...))

	actual, err := application.LoadByID(db, app.ID, u)
	assert.NoError(t, err)

	assert.Equal(t, app.Name, actual.Name)
	assert.Equal(t, proj.ID, actual.ProjectID)
	assert.Equal(t, proj.Key, actual.ProjectKey)
}
