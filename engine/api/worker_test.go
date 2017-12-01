package api

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/token"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
)

func Test_workerCheckingHandler(t *testing.T) {
	api, _, router := newTestAPI(t, bootstrap.InitiliazeDB)

	//1. Load all workers and hatcheries
	workers, err := worker.LoadWorkers(api.mustDB())
	if err != nil {
		t.Fatal(err)
	}
	hs, err := hatchery.LoadHatcheries(api.mustDB())
	if err != nil {
		t.Fatalf("Unable to load hatcheries : %s", err)
	}
	//2. Delete all workers and hatcheries
	for _, w := range workers {
		if err := worker.DeleteWorker(api.mustDB(), w.ID); err != nil {
			t.Fatal(err)
		}
	}
	for _, h := range hs {
		err := hatchery.DeleteHatchery(api.mustDB(), h.ID, 0)
		if err != nil {
			t.Fatalf("Unable to delete hatcheries : %s", err)
		}
	}

	//3. Create model
	g, err := group.LoadGroup(api.mustDB(), "shared.infra")
	if err != nil {
		t.Fatalf("Error getting group : %s", err)
	}
	model, _ := worker.LoadWorkerModelByName(api.mustDB(), "Test1")
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

		if err := worker.InsertWorkerModel(api.mustDB(), model); err != nil {
			t.Fatalf("Error inserting model : %s", err)
		}
	}

	//4. Registering worker
	h := sdk.Hatchery{
		Name:    "test-hatchery",
		GroupID: g.ID,
		UID:     "UUID",
	}
	if err := hatchery.InsertHatchery(api.mustDB(), &h); err != nil {
		t.Fatalf("Error inserting hatchery : %s", err)
	}

	if err := token.InsertToken(api.mustDB(), g.ID, "test-key", sdk.Persistent); err != nil {
		t.Fatalf("Error inserting token : %s", err)
	}

	workr, err := worker.RegisterWorker(api.mustDB(), "test-worker", "test-key", model.ID, &h, nil)
	if err != nil {
		t.Fatalf("Error Registering worker : %s", err)
	}

	//Prepare request
	uri := router.GetRoute("POST", api.workerCheckingHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequestFromWorker(t, workr, "POST", uri, nil)
	req.Header.Set("User-Agent", string(sdk.WorkerAgent))

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 204, w.Code)

	workers, err = worker.LoadWorkers(api.mustDB())
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range workers {
		assert.Equal(t, sdk.StatusChecking, w.Status)
	}

}

func Test_workerWaitingHandler(t *testing.T) {
	api, _, router := newTestAPI(t, bootstrap.InitiliazeDB)

	//1. Load all workers and hatcheries
	workers, errlw := worker.LoadWorkers(api.mustDB())
	if errlw != nil {
		t.Fatal(errlw)
	}
	hs, errlh := hatchery.LoadHatcheries(api.mustDB())
	if errlh != nil {
		t.Fatalf("Unable to load hatcheries : %s", errlh)
	}
	//2. Delete all workers and hatcheries
	for _, w := range workers {
		if err := worker.DeleteWorker(api.mustDB(), w.ID); err != nil {
			t.Fatal(err)
		}
	}
	for _, h := range hs {
		err := hatchery.DeleteHatchery(api.mustDB(), h.ID, 0)
		if err != nil {
			t.Fatalf("Unable to delete hatcheries : %s", err)
		}
	}

	//3. Create model
	g, err := group.LoadGroup(api.mustDB(), "shared.infra")
	if err != nil {
		t.Fatalf("Error getting group : %s", err)
	}
	model, _ := worker.LoadWorkerModelByName(api.mustDB(), "Test1")
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

		if err := worker.InsertWorkerModel(api.mustDB(), model); err != nil {
			t.Fatalf("Error inserting model : %s", err)
		}
	}

	//4. Registering worker
	h := sdk.Hatchery{
		Name:    "test-hatchery",
		GroupID: g.ID,
		UID:     "UUID",
	}
	if err := hatchery.InsertHatchery(api.mustDB(), &h); err != nil {
		t.Fatalf("Error inserting hatchery : %s", err)
	}

	if err := token.InsertToken(api.mustDB(), g.ID, "test-key", sdk.Persistent); err != nil {
		t.Fatalf("Error inserting token : %s", err)
	}

	workr, err := worker.RegisterWorker(api.mustDB(), "test-worker", "test-key", model.ID, &h, nil)
	if err != nil {
		t.Fatalf("Error Registering worker : %s", err)
	}

	worker.SetStatus(api.mustDB(), workr.ID, sdk.StatusBuilding)

	//Prepare request
	uri := router.GetRoute("POST", api.workerWaitingHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequestFromWorker(t, workr, "POST", uri, nil)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 204, w.Code)

	workers, err = worker.LoadWorkers(api.mustDB())
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range workers {
		assert.Equal(t, sdk.StatusWaiting, w.Status)
	}

}
