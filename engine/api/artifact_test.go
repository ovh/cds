package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func TestAPI_getStorageDriverDefault(t *testing.T) {
	api, _, _, end := newTestAPI(t)
	defer end()

	cfg := objectstore.Config{
		Kind: objectstore.Filesystem,
		Options: objectstore.ConfigOptions{
			Filesystem: objectstore.ConfigOptionsFilesystem{
				Basedir: path.Join(os.TempDir(), "store"),
			},
		},
	}

	storage, errO := objectstore.Init(context.Background(), cfg)
	test.NoError(t, errO)
	api.SharedStorage = storage

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, api.mustDB(), api.Cache, key, key)

	storageDriver, err := api.getStorageDriver(proj.Key, sdk.DefaultStorageIntegrationName)
	test.NoError(t, err)

	test.NotNil(t, storageDriver)
	test.Equal(t, storageDriver.GetProjectIntegration().Name, sdk.DefaultStorageIntegrationName)
}

func TestAPI_getArtifactsStoreHandler(t *testing.T) {
	api, _, _, end := newTestAPI(t)
	defer end()

	cfg := objectstore.Config{
		Kind: objectstore.Filesystem,
		Options: objectstore.ConfigOptions{
			Filesystem: objectstore.ConfigOptionsFilesystem{
				Basedir: path.Join(os.TempDir(), "store"),
			},
		},
	}

	storage, errO := objectstore.Init(context.Background(), cfg)
	test.NoError(t, errO)
	api.SharedStorage = storage

	u, pass := assets.InsertAdminUser(t, api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, api.mustDB(), api.Cache, key, key)

	//Prepare request
	vars := map[string]string{
		"permProjectKey":  proj.Key,
		"integrationName": sdk.DefaultStorageIntegrationName,
	}
	uri := api.Router.GetRoute("GET", api.getArtifactsStoreHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	res := sdk.ArtifactsStore{}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, sdk.DefaultStorageIntegrationName, res.Name)
}
