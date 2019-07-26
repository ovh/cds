package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_getGroupHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	g := sdk.Group{Name: sdk.RandomString(10)}
	u, jwtRaw := assets.InsertLambdaUser(db, &g)

	uri := api.Router.GetRoute(http.MethodGet, api.getGroupHandler, map[string]string{
		"permGroupName": g.Name,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodGet, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	var result sdk.Group
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	assert.Equal(t, g.Name, result.Name)
	assert.Equal(t, 1, len(result.Members))
	assert.Equal(t, u.ID, result.Members[0].ID)
	assert.True(t, result.Members[0].GroupAdmin)
}

func Test_getGroupsHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	g1 := &sdk.Group{Name: sdk.RandomString(10)}
	g2 := &sdk.Group{Name: sdk.RandomString(10)}
	_, jwtRaw := assets.InsertLambdaUser(db, g1, g2)

	uri := api.Router.GetRoute(http.MethodGet, api.getGroupsHandler, nil)
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodGet, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	var results []sdk.Group
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &results))
	require.True(t, len(results) == 2)
	var g1Result, g2Result *sdk.Group
	for i := range results {
		if results[i].ID == g1.ID {
			g1Result = &results[i]
		}
		if results[i].ID == g2.ID {
			g2Result = &results[i]
		}
	}
	assert.NotNil(t, g1Result)
	assert.NotNil(t, g2Result)
}

func Test_postGroupHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	u, jwtRaw := assets.InsertLambdaUser(db)

	data := sdk.Group{
		Name: sdk.RandomString(10),
	}

	uri := api.Router.GetRoute(http.MethodPost, api.postGroupHandler, nil)
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodPost, uri, data)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 201, rec.Code)

	var created sdk.Group
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.Equal(t, data.Name, created.Name)
	assert.Equal(t, 1, len(created.Members))
	assert.Equal(t, u.ID, created.Members[0].ID)
	assert.True(t, created.Members[0].GroupAdmin)
}

func Test_putGroupHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	g1Name, g2Name := sdk.RandomString(10), sdk.RandomString(10)
	g1, g2 := &sdk.Group{Name: g1Name}, &sdk.Group{Name: g2Name}
	_, jwtRaw := assets.InsertLambdaUser(db, g1, g2)

	g1.Name = g2Name
	uri := api.Router.GetRoute(http.MethodPut, api.putGroupHandler, map[string]string{
		"permGroupName": g1Name,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodPut, uri, g1)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 400, rec.Code)

	g1.Name = sdk.RandomString(10)
	uri = api.Router.GetRoute(http.MethodPut, api.putGroupHandler, map[string]string{
		"permGroupName": g1Name,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodPut, uri, g1)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	result, err := group.LoadByName(context.TODO(), db, g1.Name)
	require.NoError(t, err)
	assert.Equal(t, g1.ID, result.ID)
}

func Test_deleteGroupHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	g := sdk.Group{Name: sdk.RandomString(10)}
	_, jwtRaw := assets.InsertLambdaUser(db, &g)

	uri := api.Router.GetRoute(http.MethodDelete, api.deleteGroupHandler, map[string]string{
		"permGroupName": g.Name,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodDelete, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	_, err := group.LoadByName(context.TODO(), db, g.Name)
	assert.Error(t, err)
}
