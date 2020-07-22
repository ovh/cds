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
	api, db, router := newTestAPI(t)

	g := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	_, jwtAdmin := assets.InsertAdminUser(t, db)
	_, jwtLambda := assets.InsertLambdaUser(t, db, g)

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
	req := assets.NewJWTAuthentifiedRequest(t, jwtAdmin, http.MethodPost, uri, model)
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// Get secrets for the model
	uri = router.GetRoute(http.MethodGet, api.getWorkerModelSecretHandler, map[string]string{
		"permGroupName": g.Name,
		"permModelName": model.Name,
	})
	test.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtAdmin, http.MethodGet, uri, nil)
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var secrets sdk.WorkerModelSecrets
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &secrets))
	require.Len(t, secrets, 1)
	assert.Equal(t, "pwtest", secrets[0].Value)

	// Get secrets for the model by lambda user should fail even if user in model's group
	uri = router.GetRoute(http.MethodGet, api.getWorkerModelSecretHandler, map[string]string{
		"permGroupName": g.Name,
		"permModelName": model.Name,
	})
	test.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtLambda, http.MethodGet, uri, nil)
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 403, w.Code)
}
