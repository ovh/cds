package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/cds/sdk/log"
)

func RegisterWorker(t *testing.T, api *API, groupID int64, existingWorkerModelName string, jobID int64, registerOnly bool) (*sdk.Worker, string) {
	model, err := workermodel.LoadByNameAndGroupID(context.TODO(), api.mustDB(), existingWorkerModelName, groupID)
	if err != nil {
		t.Fatalf("RegisterWorker> Error getting worker model : %s", err)
	}

	g, err := group.LoadByID(context.TODO(), api.mustDB(), groupID)
	if err != nil {
		t.Fatalf("RegisterWorker> Error getting group : %s", err)
	}
	hSrv, hPrivKey, hConsumer, _ := assets.InsertHatchery(t, api.mustDB(), *g)

	hPubKey, err := jws.ExportPublicKey(hPrivKey)
	if err != nil {
		t.Fatalf("RegisterWorker> Error exporting public key : %s", err)
	}
	log.Debug("hatchery public key is %s", string(hPubKey))

	jwt, err := hatchery.NewWorkerToken(hSrv.Name, hPrivKey, time.Now().Add(time.Hour), hatchery.SpawnArguments{
		HatcheryName: hSrv.Name,
		Model:        model,
		WorkerName:   hSrv.Name + "-worker",
		JobID:        jobID,
		RegisterOnly: registerOnly,
	})
	test.NoError(t, err)
	assert.NotNil(t, hConsumer)

	uri := api.Router.GetRoute("POST", api.postRegisterWorkerHandler, nil)
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, sdk.WorkerRegistrationForm{
		Arch:    runtime.GOARCH,
		OS:      runtime.GOOS,
		Version: sdk.VERSION,
	})

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	var w sdk.Worker
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &w))
	workerJWT := rec.Header().Get("X-CDS-JWT")

	t.Logf("Worker JWT: %s", workerJWT)

	return &w, workerJWT
}

func LoadSharedInfraGroup(t *testing.T, api *API) *sdk.Group {
	g, err := group.LoadByName(context.TODO(), api.mustDB(), "shared.infra")
	if err != nil {
		t.Fatalf("Error getting group : %s", err)
	}
	return g
}

func LoadOrCreateWorkerModel(t *testing.T, api *API, groupID int64, workermodelName string) *sdk.Model {
	model, _ := workermodel.LoadByNameAndGroupID(context.TODO(), api.mustDB(), workermodelName, groupID)
	if model == nil {
		model = &sdk.Model{
			Name:    workermodelName,
			GroupID: groupID,
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

		if err := workermodel.Insert(context.TODO(), api.mustDB(), model); err != nil {
			t.Fatalf("Error inserting model : %s", err)
		}
	}

	return model
}

func TestPostRegisterWorkerHandler(t *testing.T) {
	api, _, _, end := newTestAPI(t)
	defer end()

	g := LoadSharedInfraGroup(t, api)

	model := LoadOrCreateWorkerModel(t, api, g.ID, "Test1")

	_, workerJWT := RegisterWorker(t, api, g.ID, model.Name, 0, true)

	uri := api.Router.GetRoute("POST", api.postRefreshWorkerHandler, nil)
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, workerJWT, "POST", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 204, rec.Code)

	uri = api.Router.GetRoute("POST", api.postUnregisterWorkerHandler, nil)
	test.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, workerJWT, "POST", uri, nil)

	//Do the request
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 204, rec.Code)
}

// TestPostInvalidRegister tests to register a worker for a job, without a JobID
func TestPostInvalidRegister(t *testing.T) {
	api, _, _, end := newTestAPI(t)
	defer end()

	g := LoadSharedInfraGroup(t, api)

	model := LoadOrCreateWorkerModel(t, api, g.ID, "Test2")

	hSrv, hPrivKey, hConsumer, _ := assets.InsertHatchery(t, api.mustDB(), *g)

	hPubKey, err := jws.ExportPublicKey(hPrivKey)
	if err != nil {
		t.Fatalf("RegisterWorker> Error exporting public key : %s", err)
	}
	log.Debug("hatchery public key is %s", string(hPubKey))

	jwt, err := hatchery.NewWorkerToken(hSrv.Name, hPrivKey, time.Now().Add(time.Hour), hatchery.SpawnArguments{
		HatcheryName: hSrv.Name,
		Model:        model,
		WorkerName:   hSrv.Name + "-worker",
		JobID:        0,
		RegisterOnly: false,
	})
	test.NoError(t, err)
	assert.NotNil(t, hConsumer)

	uri := api.Router.GetRoute("POST", api.postRegisterWorkerHandler, nil)
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, sdk.WorkerRegistrationForm{
		Arch:    runtime.GOARCH,
		OS:      runtime.GOOS,
		Version: sdk.VERSION,
	})

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 401, rec.Code)
}
