package worker

import (
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func TestInsertWorker(t *testing.T) {
	db := test.SetupPG(t)

	w := &sdk.Worker{
		ID:   "foofoo",
		Name: "foo.bar.io",
	}

	err := InsertWorker(db, w, 0)
	if err != nil {
		t.Fatalf("Cannot insert worker: %s", err)
	}

}

func TestDeletetWorker(t *testing.T) {
	db := test.SetupPG(t)

	w := &sdk.Worker{
		ID:   "foofoo",
		Name: "foo.bar.io",
	}

	err := InsertWorker(db, w, 0)
	if err != nil {
		t.Fatalf("Cannot insert worker: %s", err)
	}

	err = DeleteWorker(db, w.ID)
	if err != nil {
		t.Fatalf("Cannot delete worker: %s", err)
	}
}

func TestLoadWorkers(t *testing.T) {
	db := test.SetupPG(t)

	w := &sdk.Worker{ID: "foo", Name: "aa.bar.io"}
	if err := InsertWorker(db, w, 0); err != nil {
		t.Fatalf("Cannot insert worker: %s", err)
	}
	w = &sdk.Worker{ID: "foo", Name: "zz.bar.io"}
	if err := InsertWorker(db, w, 0); err != nil {
		t.Fatalf("Cannot insert worker: %s", err)
	}
	w = &sdk.Worker{ID: "foo", Name: "bb.bar.io"}
	if err := InsertWorker(db, w, 0); err != nil {
		t.Fatalf("Cannot insert worker: %s", err)
	}
	w = &sdk.Worker{ID: "foo", Name: "aa.car.io"}
	if err := InsertWorker(db, w, 0); err != nil {
		t.Fatalf("Cannot insert worker: %s", err)
	}

	workers, err := LoadWorkers(db)
	if err != nil {
		t.Fatalf("Cannot load workers: %s", err)
	}

	if len(workers) != 4 {
		t.Fatalf("Expected 4 workers, got %d", 4)
	}

	order := []string{
		"aa.bar.io",
		"aa.car.io",
		"bb.bar.io",
		"zz.bar.io",
	}
	for i := range order {
		if order[i] != workers[i].Name {
			t.Fatalf("Expected %s, got %s\n", order[i], workers[i].Name)
		}
	}
}
