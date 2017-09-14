package worker

import (
	"testing"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func TestInsertWorker(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	workers, err := LoadWorkers(db)
	test.NoError(t, err)
	for _, w := range workers {
		DeleteWorker(db, w.ID)
	}

	w := &sdk.Worker{
		ID:   "foofoo",
		Name: "foo.bar.io",
	}

	if err := InsertWorker(db, w, 0); err != nil {
		t.Fatalf("Cannot insert worker: %s", err)
	}

}

func TestDeletetWorker(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	workers, errl := LoadWorkers(db)
	test.NoError(t, errl)
	for _, w := range workers {
		DeleteWorker(db, w.ID)
	}

	w := &sdk.Worker{
		ID:   "foofoo_to_delete",
		Name: "foo.bar.io",
	}

	if err := InsertWorker(db, w, 0); err != nil {
		t.Fatalf("Cannot insert worker: %s", err)
	}

	if err := DeleteWorker(db, w.ID); err != nil {
		t.Fatalf("Cannot delete worker: %s", err)
	}
}

func TestLoadWorkers(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	workers, errl := LoadWorkers(db)
	test.NoError(t, errl)
	for _, w := range workers {
		DeleteWorker(db, w.ID)
	}

	w := &sdk.Worker{ID: "foo1", Name: "aa.bar.io"}
	if err := InsertWorker(db, w, 0); err != nil {
		t.Fatalf("Cannot insert worker: %s", err)
	}
	w = &sdk.Worker{ID: "foo2", Name: "zz.bar.io"}
	if err := InsertWorker(db, w, 0); err != nil {
		t.Fatalf("Cannot insert worker: %s", err)
	}
	w = &sdk.Worker{ID: "foo3", Name: "bb.bar.io"}
	if err := InsertWorker(db, w, 0); err != nil {
		t.Fatalf("Cannot insert worker: %s", err)
	}
	w = &sdk.Worker{ID: "foo4", Name: "aa.car.io"}
	if err := InsertWorker(db, w, 0); err != nil {
		t.Fatalf("Cannot insert worker: %s", err)
	}

	var errlw error
	workers, errlw = LoadWorkers(db)
	if errlw != nil {
		t.Fatalf("Cannot load workers: %s", errlw)
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
