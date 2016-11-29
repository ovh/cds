package worker

import (
	"database/sql"
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

func insertUser(t *testing.T, db *sql.DB, name string) sdk.User {
	u := sdk.User{
		Username: name,
	}

	err := user.InsertUser(db, &u, nil)
	if err != nil {
		t.Fatalf("Cannot insert user: %s", err)
	}
	return u
}

func insertWorkerModel(t *testing.T, db *sql.DB, name string, userID int64) {
	m := sdk.Model{
		Name:    name,
		Type:    sdk.Docker,
		Image:   "foo/bar:3.4",
		OwnerID: userID,
	}

	err := InsertWorkerModel(db, &m)
	if err != nil {
		t.Fatalf("Cannot insert worker model: %s", err)
	}
}

func TestInsertWorkerModel(t *testing.T) {
	db := test.Setup("TestInsertWorkerModel", t)
	u := insertUser(t, db, "fakeUser")
	insertWorkerModel(t, db, "Foo", u.ID)
}

func TestLoadWorkerModel(t *testing.T) {
	db := test.Setup("TestLoadWorkerModel", t)

	u := insertUser(t, db, "fakeUser")

	insertWorkerModel(t, db, "Foo", u.ID)

	m, err := LoadWorkerModel(db, "Foo")
	if err != nil {
		t.Fatalf("Cannot load worker model: %s", err)
	}

	if m.Type != sdk.Docker {
		t.Fatalf("Unexpected model type '%s', wanted '%s'", m.Type, sdk.Docker)
	}

}

func insertCapacity(db *sql.DB, t *testing.T, modelID int64, capa sdk.Requirement) {

	err := InsertWorkerModelCapability(db, modelID, capa)
	if err != nil {
		t.Fatalf("Cannot insert worker model capacity: %s", err)
	}
}

func TestInsertWorkerModelCapacity(t *testing.T) {
	db := test.Setup("TestInsertWorkerModelCapacity", t)
	u := insertUser(t, db, "fakeUser")
	insertWorkerModel(t, db, "Foo", u.ID)

	m, err := LoadWorkerModel(db, "Foo")
	if err != nil {
		t.Fatalf("cannot load model: %s", err)
	}

	capa := sdk.Requirement{
		Name:  "Go",
		Type:  sdk.BinaryRequirement,
		Value: "1.5.1",
	}
	insertCapacity(db, t, m.ID, capa)

}
