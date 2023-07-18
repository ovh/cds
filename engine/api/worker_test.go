package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/jws"
)

func RegisterWorker(t *testing.T, api *API, db gorpmapper.SqlExecutorWithTx, groupID int64, existingWorkerModelName string, jobID int64, registerOnly bool) (*sdk.Worker, string) {
	model, err := workermodel.LoadByNameAndGroupID(context.TODO(), api.mustDB(), existingWorkerModelName, groupID)
	require.NoError(t, err)

	g, err := group.LoadByID(context.TODO(), api.mustDB(), groupID)
	require.NoError(t, err)
	hSrv, hPrivKey, hConsumer, _ := assets.InsertHatchery(t, db, *g)

	hPubKey, err := jws.ExportPublicKey(hPrivKey)
	require.NoError(t, err)
	log.Debug(context.TODO(), "hatchery public key is %s", string(hPubKey))

	jwt, err := hatchery.NewWorkerToken(hSrv.Name, hPrivKey, time.Now().Add(time.Hour), hatchery.SpawnArguments{
		HatcheryName: hSrv.Name,
		Model:        sdk.WorkerStarterWorkerModel{ModelV1: model},
		WorkerName:   hSrv.Name + "-worker",
		JobID:        fmt.Sprintf("%d", jobID),
		RegisterOnly: registerOnly,
	})
	require.NoError(t, err)
	require.NotNil(t, hConsumer)

	uri := api.Router.GetRoute("POST", api.postRegisterWorkerHandler, nil)
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, sdk.WorkerRegistrationForm{
		Arch:    runtime.GOARCH,
		OS:      runtime.GOOS,
		Version: sdk.VERSION,
	})

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	//Check result
	var w sdk.Worker
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &w))
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

func LoadOrCreateWorkerModel(t *testing.T, api *API, db gorpmapper.SqlExecutorWithTx, groupID int64, workermodelName string) *sdk.Model {
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

		if err := workermodel.Insert(context.TODO(), db, model); err != nil {
			t.Fatalf("Error inserting model : %s", err)
		}
	}

	return model
}

func TestPostRegisterWorkerHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	g := LoadSharedInfraGroup(t, api)

	model := LoadOrCreateWorkerModel(t, api, db, g.ID, "Test1")

	_, workerJWT := RegisterWorker(t, api, db, g.ID, model.Name, 0, true)

	uri := api.Router.GetRoute("POST", api.postRefreshWorkerHandler, nil)
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, workerJWT, "POST", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 204, rec.Code)

	uri = api.Router.GetRoute("POST", api.postUnregisterWorkerHandler, nil)
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, workerJWT, "POST", uri, nil)

	//Do the request
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 204, rec.Code)
}

// TestPostInvalidRegister tests to register a worker for a job, without a JobID
func TestPostInvalidRegister(t *testing.T) {
	api, db, _ := newTestAPI(t)

	g := LoadSharedInfraGroup(t, api)

	model := LoadOrCreateWorkerModel(t, api, db, g.ID, "Test2")

	hSrv, hPrivKey, hConsumer, _ := assets.InsertHatchery(t, db, *g)

	hPubKey, err := jws.ExportPublicKey(hPrivKey)
	if err != nil {
		t.Fatalf("RegisterWorker> Error exporting public key : %s", err)
	}
	log.Debug(context.TODO(), "hatchery public key is %s", string(hPubKey))

	jwt, err := hatchery.NewWorkerToken(hSrv.Name, hPrivKey, time.Now().Add(time.Hour), hatchery.SpawnArguments{
		HatcheryName: hSrv.Name,
		Model:        sdk.WorkerStarterWorkerModel{ModelV1: model},
		WorkerName:   hSrv.Name + "-worker",
		JobID:        "0",
		RegisterOnly: false,
	})
	require.NoError(t, err)
	require.NotNil(t, hConsumer)

	uri := api.Router.GetRoute("POST", api.postRegisterWorkerHandler, nil)
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, sdk.WorkerRegistrationForm{
		Arch:    runtime.GOARCH,
		OS:      runtime.GOOS,
		Version: sdk.VERSION,
	})

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 401, rec.Code)
}
