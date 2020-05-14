package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_getWorkerModelSecretHandler(t *testing.T) {
	api, _, router, end := newTestAPI(t)
	defer end()

	_, jwt := assets.InsertAdminUser(t, api.mustDB())
	g := assets.InsertTestGroup(t, api.mustDB(), sdk.RandomString(10))
	model := sdk.Model{
		Name:       "Test1",
		GroupID:    g.ID,
		Type:       sdk.Docker,
		Restricted: true,
		ModelDocker: sdk.ModelDocker{
			Image:    "buildpack-deps:jessie",
			Shell:    "sh -c",
			Cmd:      "worker",
			Private:  true,
			Username: "test",
			Password: "pwtest",
		},
	}

	// Create new model with registry password
	uri := router.GetRoute(http.MethodPost, api.postWorkerModelHandler, nil)
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodPost, uri, model)
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// Get secrets for the model
	uri = router.GetRoute(http.MethodGet, api.getWorkerModelSecretHandler, map[string]string{
		"permGroupName": g.Name,
		"permModelName": model.Name,
	})
	test.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodGet, uri, nil)
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var secrets sdk.WorkerModelSecrets
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &secrets))
	require.Len(t, secrets, 1)
	assert.Equal(t, "pwtest", secrets[0].Value)
}
