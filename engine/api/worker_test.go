package main

import (
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/testwithdb"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
)

func Test_workerCheckingHandler(t *testing.T) {
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := testwithdb.SetupPG(t, bootstrap.InitiliazeDB)
	assert.NoError(t, err)
	dbmap := database.DBMap(db)

	//1. Load all workers and hatcheries
	workers, err := worker.LoadWorkers(db)
	if err != nil {
		t.Fatal(err)
	}
	hs, err := hatchery.LoadHatcheries(db)
	if err != nil {
		t.Fatalf("Unable to load hatcheries : %s", err)
	}
	//2. Delete all workers and hatcheries
	for _, w := range workers {
		if err := worker.DeleteWorker(db, w.ID); err != nil {
			t.Fatal(err)
		}
	}
	for _, h := range hs {
		err := hatchery.DeleteHatchery(db, h.ID, 0)
		if err != nil {
			t.Fatalf("Unable to delete hatcheries : %s", err)
		}
	}

	//3. Create model
	g, err := group.LoadGroup(db, "shared.infra")
	if err != nil {
		t.Fatalf("Error getting group : %s", err)
	}
	model, _ := worker.LoadWorkerModelByName(dbmap, "Test1")
	if model == nil {
		model = &sdk.Model{
			Name:    "Test1",
			GroupID: g.ID,
			Type:    sdk.Docker,
			Image:   "buildpack-deps:jessie",
			Capabilities: []sdk.Requirement{
				{
					Name:  "capa1",
					Type:  sdk.BinaryRequirement,
					Value: "1",
				},
			},
		}

		if err := worker.InsertWorkerModel(dbmap, model); err != nil {
			t.Fatalf("Error inserting model : %s", err)
		}
	}

	//4. Registering worker
	h := sdk.Hatchery{
		Name:    "test-hatchery",
		GroupID: g.ID,
		UID:     "UUID",
	}
	if err := hatchery.InsertHatchery(db, &h); err != nil {
		t.Fatalf("Error inserting hatchery : %s", err)
	}

	if err := worker.InsertToken(db, g.ID, "test-key", sdk.Persistent); err != nil {
		t.Fatalf("Error inserting token : %s", err)
	}

	workr, err := worker.RegisterWorker(db, "test-worker", "test-key", model.ID, &h, nil)
	if err != nil {
		t.Fatalf("Error Registering worker : %s", err)
	}

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local", TTL: 30})
	router = &Router{authDriver, mux.NewRouter(), "/Test_workerCheckingHandler"}
	router.init()

	//Prepare request
	uri := router.getRoute("POST", workerCheckingHandler, nil)
	if uri == "" {
		t.FailNow()
	}
	req := testwithdb.NewAuthentifiedRequestFromWorker(t, workr, "POST", uri, nil)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	workers, err = worker.LoadWorkers(db)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range workers {
		assert.Equal(t, sdk.StatusChecking, w.Status)
	}

}

func Test_workerWaitingHandler(t *testing.T) {
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := testwithdb.SetupPG(t, bootstrap.InitiliazeDB)
	assert.NoError(t, err)
	dbmap := database.DBMap(db)

	//1. Load all workers and hatcheries
	workers, err := worker.LoadWorkers(db)
	if err != nil {
		t.Fatal(err)
	}
	hs, err := hatchery.LoadHatcheries(db)
	if err != nil {
		t.Fatalf("Unable to load hatcheries : %s", err)
	}
	//2. Delete all workers and hatcheries
	for _, w := range workers {
		if err := worker.DeleteWorker(db, w.ID); err != nil {
			t.Fatal(err)
		}
	}
	for _, h := range hs {
		err := hatchery.DeleteHatchery(db, h.ID, 0)
		if err != nil {
			t.Fatalf("Unable to delete hatcheries : %s", err)
		}
	}

	//3. Create model
	g, err := group.LoadGroup(db, "shared.infra")
	if err != nil {
		t.Fatalf("Error getting group : %s", err)
	}
	model, _ := worker.LoadWorkerModelByName(dbmap, "Test1")
	if model == nil {
		model = &sdk.Model{
			Name:    "Test1",
			GroupID: g.ID,
			Type:    sdk.Docker,
			Image:   "buildpack-deps:jessie",
			Capabilities: []sdk.Requirement{
				{
					Name:  "capa1",
					Type:  sdk.BinaryRequirement,
					Value: "1",
				},
			},
		}

		if err := worker.InsertWorkerModel(dbmap, model); err != nil {
			t.Fatalf("Error inserting model : %s", err)
		}
	}

	//4. Registering worker
	h := sdk.Hatchery{
		Name:    "test-hatchery",
		GroupID: g.ID,
		UID:     "UUID",
	}
	if err := hatchery.InsertHatchery(db, &h); err != nil {
		t.Fatalf("Error inserting hatchery : %s", err)
	}

	if err := worker.InsertToken(db, g.ID, "test-key", sdk.Persistent); err != nil {
		t.Fatalf("Error inserting token : %s", err)
	}

	workr, err := worker.RegisterWorker(db, "test-worker", "test-key", model.ID, &h, nil)
	if err != nil {
		t.Fatalf("Error Registering worker : %s", err)
	}

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local", TTL: 30})
	router = &Router{authDriver, mux.NewRouter(), "/Test_workerWaitingHandler"}
	router.init()

	//Prepare request
	uri := router.getRoute("POST", workerWaitingHandler, nil)
	if uri == "" {
		t.FailNow()
	}
	req := testwithdb.NewAuthentifiedRequestFromWorker(t, workr, "POST", uri, nil)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	workers, err = worker.LoadWorkers(db)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range workers {
		assert.Equal(t, sdk.StatusWaiting, w.Status)
	}

}
