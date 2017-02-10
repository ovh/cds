package worker

import (
	"testing"

	"github.com/go-gorp/gorp"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func deleteAllWorkerModel(t *testing.T, db gorp.SqlExecutor) {
	//Loading all models
	models, err := LoadWorkerModels(db)
	if err != nil {
		t.Fatalf("Error getting models : %s", err)
	}

	//Delete all of them
	for _, m := range models {
		if err := DeleteWorkerModel(db, m.ID); err != nil {
			t.Fatalf("Error deleting model : %s", err)
		}
	}
}

func insertGroup(t *testing.T, db gorp.SqlExecutor) *sdk.Group {
	g := &sdk.Group{
		Name: assets.RandomString(t, 10),
	}

	if err := group.InsertGroup(db, g); err != nil {
		t.Fatalf("Unable to create group %s", err)
	}

	return g
}

func insertWorkerModel(t *testing.T, db gorp.SqlExecutor, name string, groupID int64) *sdk.Model {
	m := sdk.Model{
		Name:    name,
		Type:    sdk.Docker,
		Image:   "foo/bar:3.4",
		GroupID: groupID,
		Capabilities: []sdk.Requirement{
			{
				Name:  "capa_1",
				Type:  sdk.BinaryRequirement,
				Value: "capa_1",
			},
		},
	}

	if err := InsertWorkerModel(db, &m); err != nil {
		t.Fatalf("Cannot insert worker model: %s", err)
	}

	assert.NotEqual(t, 0, m.ID)
	return &m
}

func TestInsertWorkerModel(t *testing.T) {
	db := test.SetupPG(t)
	deleteAllWorkerModel(t, db)

	g := insertGroup(t, db)

	m := insertWorkerModel(t, db, "Foo", g.ID)

	m1, err := LoadWorkerModelByID(db, m.ID)
	if err != nil {
		t.Fatalf("Cannot load worker model: %s", err)
	}
	assert.EqualValues(t, m, m1)

	m2, err := LoadWorkerModelsByGroup(db, g.ID)
	assert.EqualValues(t, []sdk.Model{*m}, m2)

	u, _ := assets.InsertLambaUser(t, db, g)

	m3, err := LoadWorkerModelsByUser(db, u.ID)
	assert.EqualValues(t, []sdk.Model{*m}, m3)

}

func TestLoadWorkerModel(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)
	deleteAllWorkerModel(t, db)

	g, err := group.LoadGroup(db, "shared.infra")
	if err != nil {
		t.Fatalf("Error : %s", err)
	}
	insertWorkerModel(t, db, "Foo", g.ID)

	m, err := LoadWorkerModelByName(db, "Foo")
	test.NoError(t, err)
	if err != nil {
		t.Fatalf("Cannot load worker model: %s", err)
	}
	assert.NotNil(t, m)
	assert.Equal(t, sdk.Docker, m.Type)

	m1, err := LoadSharedWorkerModels(db)
	if err != nil {
		t.Fatalf("Error : %s", err)
	}
	assert.EqualValues(t, []sdk.Model{*m}, m1)
}

func TestLoadWorkerModels(t *testing.T) {
	db := test.SetupPG(t)
	deleteAllWorkerModel(t, db)

	g := insertGroup(t, db)

	insertWorkerModel(t, db, "lol", g.ID)
	insertWorkerModel(t, db, "foo", g.ID)

	models, err := LoadWorkerModels(db)
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
	db := test.SetupPG(t, bootstrap.InitiliazeDB)
	deleteAllWorkerModel(t, db)

	g, err := group.LoadGroup(db, "shared.infra")
	if err != nil {
		t.Fatalf("Error : %s", err)
	}
	m := insertWorkerModel(t, db, "Foo", g.ID)

	capa, err := LoadWorkerModelCapabilities(db, m.ID)
	assert.EqualValues(t, m.Capabilities, capa)
}

func TestUpdateWorkerModel(t *testing.T) {
	db := test.SetupPG(t)
	deleteAllWorkerModel(t, db)

	g := insertGroup(t, db)

	m := insertWorkerModel(t, db, "lol", g.ID)
	m1 := *m
	m1.Capabilities = append(m1.Capabilities, sdk.Requirement{
		Name:  "Capa_2",
		Type:  sdk.BinaryRequirement,
		Value: "Capa_2",
	})

	if err := UpdateWorkerModel(db, m1); err != nil {
		t.Fatalf("Error : %s", err)
	}

	m3, err := LoadWorkerModelByName(db, "lol")
	test.NoError(t, err)
	if err != nil {
		t.Fatalf("Cannot load worker model: %s", err)
	}
	assert.NotNil(t, m)
	assert.Equal(t, sdk.Docker, m3.Type)
	assert.Equal(t, 2, len(m3.Capabilities))

}
