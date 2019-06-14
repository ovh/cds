package api

import (
	"context"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
)

func TestPostRegisterWorkerHandler(t *testing.T) {
	api, _, _, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()

	ctx := context.TODO()

	g, err := group.LoadByName(ctx, api.mustDB(), "shared.infra")
	if err != nil {
		t.Fatalf("Error getting group : %s", err)
	}

	model, _ := workermodel.LoadByNameAndGroupID(api.mustDB(), "Test1", g.ID)
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

		if err := workermodel.Insert(api.mustDB(), model); err != nil {
			t.Fatalf("Error inserting model : %s", err)
		}
	}

	hSrv, hPrivKey, hConsumer, _ := assets.InsertHatchery(t, api.mustDB(), *g)
	session, jwt, err := hatchery.NewWorkerToken(hSrv.Name, hPrivKey, time.Now().Add(time.Hour), hatchery.SpawnArguments{
		HatcheryName: hSrv.Name,
		Model:        *model,
		WorkerName:   hSrv.Name + "-worker",
	})
	test.NoError(t, err)
	assert.NotNil(t, hConsumer)
	assert.NotNil(t, session)

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
	t.Logf(">>%s", rec.Body.String())
	workerJWT := rec.Header().Get("X-CDS-JWT")
	t.Logf(">>%s", workerJWT)

	uri = api.Router.GetRoute("POST", api.postRefreshWorkerHandler, nil)
	test.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, workerJWT, "POST", uri, nil)

	//Do the request
	rec = httptest.NewRecorder()
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
