package api

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/token"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
)

func Test_workerCheckingHandler(t *testing.T) {
	api, _, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()

	//1. Load all workers
	workers, err := worker.LoadWorkers(api.mustDB(), "")
	if err != nil {
		t.Fatal(err)
	}
	//2. Delete all workers
	for _, w := range workers {
		if err := worker.DeleteWorker(api.mustDB(), w.ID); err != nil {
			t.Fatal(err)
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
			ModelDocker: sdk.ModelDocker{
				Image: "buildpack-deps:jessie",
			},
			RegisteredCapabilities: sdk.RequirementList{
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
	h := sdk.Service{
		Name:    "test-hatchery-Test_workerCheckingHandler",
		GroupID: &g.ID,
	}
	if err := services.Insert(api.mustDB(), &h); err != nil {
		t.Fatalf("Error inserting hatchery : %s", err)
	}

	if err := token.InsertToken(api.mustDB(), g.ID, "test-key", sdk.Persistent, "", ""); err != nil {
		t.Fatalf("Error inserting token : %s", err)
	}

	workr, err := worker.RegisterWorker(api.mustDB(), api.Cache, "test-worker", "test-key", model.ID, &h, nil, "linux", "amd64")
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

	workers, err = worker.LoadWorkers(api.mustDB(), "")
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range workers {
		assert.Equal(t, sdk.StatusChecking, w.Status)
	}

}

func Test_workerWaitingHandler(t *testing.T) {
	api, _, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()

	//1. Load all workers
	workers, errlw := worker.LoadWorkers(api.mustDB(), "")
	if errlw != nil {
		t.Fatal(errlw)
	}
	//2. Delete all workers
	for _, w := range workers {
		if err := worker.DeleteWorker(api.mustDB(), w.ID); err != nil {
			t.Fatal(err)
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
			ModelDocker: sdk.ModelDocker{
				Image: "buildpack-deps:jessie",
			},
			RegisteredCapabilities: sdk.RequirementList{
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
	h := sdk.Service{
		Name:    "test-hatchery-Test_workerWaitingHandler",
		GroupID: &g.ID,
	}
	if err := services.Insert(api.mustDB(), &h); err != nil {
		t.Fatalf("Error inserting hatchery : %s", err)
	}

	if err := token.InsertToken(api.mustDB(), g.ID, "test-key", sdk.Persistent, "", ""); err != nil {
		t.Fatalf("Error inserting token : %s", err)
	}

	workr, err := worker.RegisterWorker(api.mustDB(), "test-worker", "test-key", model.ID, &h, nil, "linux", "amd64")
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

	workers, err = worker.LoadWorkers(api.mustDB(), "")
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range workers {
		assert.Equal(t, sdk.StatusWaiting, w.Status)
	}

}
