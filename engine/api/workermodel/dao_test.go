package workermodel_test

import (
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/test/assets"

	"github.com/go-gorp/gorp"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/sdk"
)

func deleteAllWorkerModel(t *testing.T, db gorp.SqlExecutor) {
	//Loading all models
	models, err := workermodel.LoadAll(db)
	if err != nil {
		t.Fatalf("Error getting models : %s", err)
	}

	//Delete all of them
	for _, m := range models {
		if err := workermodel.Delete(db, m.ID); err != nil {
			t.Fatalf("Error deleting model : %s", err)
		}
	}
}

func insertGroup(t *testing.T, db gorp.SqlExecutor) *sdk.Group {
	g := &sdk.Group{
		Name: "test-group-model",
	}

	g1, _ := group.LoadGroup(db, g.Name)
	if g1 != nil {
		group.DeleteGroupAndDependencies(db, g1)
	}

	if err := group.InsertGroup(db, g); err != nil {
		t.Fatalf("Unable to create group %s", err)
	}

	return g
}

func insertWorkerModel(t *testing.T, db gorp.SqlExecutor, name string, groupID int64) *sdk.Model {
	m := sdk.Model{
		Name: name,
		Type: sdk.Docker,
		ModelDocker: sdk.ModelDocker{
			Image: "foo/bar:3.4",
		},
		GroupID: groupID,
		RegisteredCapabilities: sdk.RequirementList{
			{
				Name:  "capa_1",
				Type:  sdk.BinaryRequirement,
				Value: "capa_1",
			},
		},
		UserLastModified: time.Now(),
	}

	if err := workermodel.Insert(db, &m); err != nil {
		t.Fatalf("Cannot insert worker model: %s", err)
	}

	assert.NotEqual(t, 0, m.ID)
	return &m
}

func TestInsertWorkerModel(t *testing.T) {
	db, store, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	deleteAllWorkerModel(t, db)

	g := insertGroup(t, db)

	m := insertWorkerModel(t, db, "Foo", g.ID)

	m1, err := workermodel.LoadByID(db, m.ID)
	if err != nil {
		t.Fatalf("Cannot load worker model: %s", err)
	}
	m1.Group = nil

	// lastregistration is LOCALTIMESTAMP (at sql insert)
	// set it manually to allow use EqualValues on others fields
	m.LastRegistration = m1.LastRegistration
	m.UserLastModified = m1.UserLastModified

	assert.EqualValues(t, m, m1)

	group.InsertGroup(db, g)
	u, _ := assets.InsertLambdaUser(db, g)
	group.InsertUserInGroup(db, g.ID, u.OldUserStruct.ID, false)

	m3, err := workermodel.LoadAllByUser(db, store, u, nil)
	if err != nil {
		t.Fatalf("Cannot load worker model by user: %s", err)
	}
	m3u := m3[0]
	m3u.Group = nil

	m.UserLastModified = m3u.UserLastModified
	m.LastRegistration = m3u.LastRegistration

	assert.EqualValues(t, *m, m3u)
}

func TestLoadWorkerModel(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	deleteAllWorkerModel(t, db)

	g, err := group.LoadGroup(db, "shared.infra")
	if err != nil {
		t.Fatalf("Error : %s", err)
	}
	insertWorkerModel(t, db, "Foo", g.ID)

	m, err := workermodel.LoadByNameAndGroupID(db, "Foo", g.ID)
	test.NoError(t, err)
	if err != nil {
		t.Fatalf("Cannot load worker model: %s", err)
	}
	assert.NotNil(t, m)
	assert.Equal(t, sdk.Docker, m.Type)

	_, errNotExist := workermodel.LoadByNameAndGroupID(db, "NotExisting", g.ID)
	assert.True(t, sdk.ErrorIs(errNotExist, sdk.ErrNoWorkerModel))
}

func TestLoadWorkerModelsByNameAndGroupIDs(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	deleteAllWorkerModel(t, db)

	g1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	insertWorkerModel(t, db, "SameName", g1.ID)
	insertWorkerModel(t, db, "SameName", g2.ID)
	insertWorkerModel(t, db, "DiffName", g2.ID)

	wms, err := workermodel.LoadAllByNameAndGroupIDs(db, "SameName", []int64{g1.ID})
	test.NoError(t, err)
	assert.Equal(t, 1, len(wms))

	wms, err = workermodel.LoadAllByNameAndGroupIDs(db, "SameName", []int64{g1.ID, g2.ID})
	test.NoError(t, err)
	assert.Equal(t, 2, len(wms))

	wms, err = workermodel.LoadAllByNameAndGroupIDs(db, "DiffName", []int64{g1.ID, g2.ID})
	test.NoError(t, err)
	assert.Equal(t, 1, len(wms))

	wms, err = workermodel.LoadAllByNameAndGroupIDs(db, "Unknown", []int64{g1.ID, g2.ID})
	test.NoError(t, err)
	assert.Equal(t, 0, len(wms))
}

func TestLoadWorkerModels(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	deleteAllWorkerModel(t, db)

	g := insertGroup(t, db)

	insertWorkerModel(t, db, "lol", g.ID)
	insertWorkerModel(t, db, "foo", g.ID)

	models, err := workermodel.LoadAll(db)
	if err != nil {
		t.Fatalf("Cannot load worker model: %s", err)
	}

	if len(models) != 2 {
		t.Fatalf("Expected 2 models, got %d", len(models))
	}

	for _, m := range models {
		if m.Type != sdk.Docker {
			t.Fatalf("Unexpected model type '%s', wanted '%s'", m.Type, sdk.Docker)
		}
	}
}

func TestLoadWorkerModelsWithFilter(t *testing.T) {
	db, store, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	deleteAllWorkerModel(t, db)

	g := insertGroup(t, db)

	insertWorkerModel(t, db, "lol", g.ID)
	insertWorkerModel(t, db, "foo", g.ID)

	opts := workermodel.StateError
	models, err := workermodel.LoadAllByUser(db, store, &sdk.User{Admin: true}, &opts)
	if err != nil {
		t.Fatalf("Cannot load worker model: %s", err)
	}

	if len(models) != 0 {
		t.Fatalf("Expected 0 models, got %d", len(models))
	}
}

func TestLoadWorkerModelsByUserAndBinary(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	deleteAllWorkerModel(t, db)
	g := insertGroup(t, db)
	u, _ := assets.InsertLambdaUser(db)
	group.InsertUserInGroup(db, g.ID, u.OldUserStruct.ID, false)

	insertWorkerModel(t, db, "lol", g.ID)
	insertWorkerModel(t, db, "foo", g.ID)

	models, err := workermodel.LoadAllByUserAndBinary(db, u, "capa_1")
	if err != nil {
		t.Fatalf("Cannot load worker model: %s", err)
	}

	if len(models) != 2 {
		t.Fatalf("Expected 2 models, got %d", len(models))
	}

	for _, m := range models {
		if m.Type != sdk.Docker {
			t.Fatalf("Unexpected model type '%s', wanted '%s'", m.Type, sdk.Docker)
		}
	}
}

func TestLoadWorkerModelCapabilities(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	deleteAllWorkerModel(t, db)

	g, err := group.LoadGroup(db, "shared.infra")
	if err != nil {
		t.Fatalf("Error : %s", err)
	}
	m := insertWorkerModel(t, db, "Foo", g.ID)

	capa, err := workermodel.LoadCapabilities(db, m.ID)
	if err != nil {
		t.Fatalf("Cannot load worker model capabilities: %s", err)
	}
	assert.EqualValues(t, m.RegisteredCapabilities, capa)
}

func TestUpdateWorkerModel(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	deleteAllWorkerModel(t, db)

	g := insertGroup(t, db)

	m := insertWorkerModel(t, db, "lol", g.ID)
	m1 := *m
	m1.RegisteredCapabilities = append(m1.RegisteredCapabilities, sdk.Requirement{
		Name:  "Capa_2",
		Type:  sdk.BinaryRequirement,
		Value: "Capa_2",
	})

	if err := workermodel.UpdateDB(db, &m1); err != nil {
		t.Fatalf("Error : %s", err)
	}

	m3, err := workermodel.LoadByNameAndGroupID(db, "lol", g.ID)
	test.NoError(t, err)
	if err != nil {
		t.Fatalf("Cannot load worker model: %s", err)
	}
	assert.NotNil(t, m)
	assert.Equal(t, sdk.Docker, m3.Type)
	assert.Equal(t, 2, len(m3.RegisteredCapabilities))
}

func TestLoadWorkerModelsForGroupIDs(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	deleteAllWorkerModel(t, db)

	g1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	m1 := insertWorkerModel(t, db, sdk.RandomString(10), g1.ID)
	m2 := insertWorkerModel(t, db, sdk.RandomString(10), g2.ID)
	m3 := sdk.Model{
		Name:             sdk.RandomString(10),
		Type:             sdk.Docker,
		ModelDocker:      sdk.ModelDocker{Image: "foo/bar:3.4"},
		GroupID:          g2.ID,
		UserLastModified: time.Now(),
		Disabled:         true,
	}
	if err := workermodel.Insert(db, &m3); err != nil {
		t.Fatalf("cannot insert worker model: %s", err)
	}

	models, err := workermodel.LoadAllActiveAndNotDeprecatedForGroupIDs(db, []int64{g1.ID})
	if err != nil {
		t.Fatalf("cannot load worker model: %s", err)
	}
	assert.Equal(t, 1, len(models))
	found := make([]bool, 2)
	for i := range models {
		if models[i].ID == m1.ID {
			found[0] = true
		}
		if models[i].ID == m2.ID {
			found[1] = true
		}
	}
	assert.True(t, found[0], "Model for group %s not found in result", g1.Name)
	assert.False(t, found[1], "Model for group %s should not be in result", g2.Name)

	models, err = workermodel.LoadAllActiveAndNotDeprecatedForGroupIDs(db, []int64{g1.ID, g2.ID})
	if err != nil {
		t.Fatalf("cannot load worker model: %s", err)
	}
	assert.Equal(t, 2, len(models))
	found = make([]bool, 2)
	for i := range models {
		if models[i].ID == m1.ID {
			found[0] = true
		}
		if models[i].ID == m2.ID {
			found[1] = true
		}
	}
	assert.True(t, found[0], "Model for group %s not found in result", g1.Name)
	assert.True(t, found[1], "Model for group %s not found in result", g2.Name)
}
