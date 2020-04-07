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
	u, jwtRaw := assets.InsertLambdaUser(t, db, &g)

	uri := api.Router.GetRoute(http.MethodGet, api.getGroupHandler, map[string]string{
		"permGroupName": g.Name,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodGet, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var result sdk.Group
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	assert.Equal(t, g.Name, result.Name)
	assert.Equal(t, 1, len(result.Members))
	assert.Equal(t, u.ID, result.Members[0].ID)
	assert.True(t, result.Members[0].Admin)
}

func Test_getGroupsHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	g1 := &sdk.Group{Name: sdk.RandomString(10)}
	g2 := &sdk.Group{Name: sdk.RandomString(10)}
	_, jwtRaw := assets.InsertLambdaUser(t, db, g1, g2)

	uri := api.Router.GetRoute(http.MethodGet, api.getGroupsHandler, nil)
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodGet, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

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

	u, jwtRaw := assets.InsertLambdaUser(t, db)

	data := sdk.Group{
		Name: sdk.RandomString(10),
	}

	uri := api.Router.GetRoute(http.MethodPost, api.postGroupHandler, nil)
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodPost, uri, data)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var created sdk.Group
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.Equal(t, data.Name, created.Name)
	assert.Equal(t, 1, len(created.Members))
	assert.Equal(t, u.ID, created.Members[0].ID)
	assert.True(t, created.Members[0].Admin)
}

func Test_putGroupHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	g1Name, g2Name := sdk.RandomString(10), sdk.RandomString(10)
	g1, g2 := &sdk.Group{Name: g1Name}, &sdk.Group{Name: g2Name}
	_, jwtRaw := assets.InsertLambdaUser(t, db, g1, g2)

	g1.Name = g2Name
	uri := api.Router.GetRoute(http.MethodPut, api.putGroupHandler, map[string]string{
		"permGroupName": g1Name,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodPut, uri, g1)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	g1.Name = sdk.RandomString(10)
	uri = api.Router.GetRoute(http.MethodPut, api.putGroupHandler, map[string]string{
		"permGroupName": g1Name,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodPut, uri, g1)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	result, err := group.LoadByName(context.TODO(), db, g1.Name)
	require.NoError(t, err)
	assert.Equal(t, g1.ID, result.ID)
}

func Test_deleteGroupHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	g := sdk.Group{Name: sdk.RandomString(10)}
	_, jwtRaw := assets.InsertLambdaUser(t, db, &g)

	uri := api.Router.GetRoute(http.MethodDelete, api.deleteGroupHandler, map[string]string{
		"permGroupName": g.Name,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodDelete, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	_, err := group.LoadByName(context.TODO(), db, g.Name)
	assert.Error(t, err)
}

func Test_postGroupUserHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	g := &sdk.Group{Name: sdk.RandomString(10)}
	_, jwtRaw1 := assets.InsertLambdaUser(t, db, g)
	u2, jwtRaw2 := assets.InsertLambdaUser(t, db)
	u3, _ := assets.InsertLambdaUser(t, db)

	require.NoError(t, group.LoadOptions.WithMembers(context.TODO(), db, g))
	assert.Equal(t, 1, len(g.Members))

	// A group admin should be able to add a user
	uri := api.Router.GetRoute(http.MethodPost, api.postGroupUserHandler, map[string]string{
		"permGroupName": g.Name,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw1, http.MethodPost, uri, sdk.GroupMember{
		ID:    u2.ID,
		Admin: false,
	})
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var result sdk.Group
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	assert.Equal(t, 2, len(result.Members))

	// A group member should not be able to add a user
	uri = api.Router.GetRoute(http.MethodPost, api.postGroupUserHandler, map[string]string{
		"permGroupName": g.Name,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtRaw2, http.MethodPost, uri, sdk.GroupMember{
		ID:    u3.ID,
		Admin: true,
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
}

func Test_putGroupUserHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	g := &sdk.Group{Name: sdk.RandomString(10)}
	_, jwtRaw1 := assets.InsertLambdaUser(t, db, g)
	u2, jwtRaw2 := assets.InsertLambdaUser(t, db, g)

	require.NoError(t, group.LoadOptions.WithMembers(context.TODO(), db, g))
	assert.Equal(t, 2, len(g.Members))

	// A group member should not be able to set himself as an admin
	uri := api.Router.GetRoute(http.MethodPut, api.putGroupUserHandler, map[string]string{
		"permGroupName": g.Name,
		"username":      u2.Username,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw2, http.MethodPut, uri, sdk.GroupMember{
		Admin: true,
	})
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)

	// A group admin should be able to set another member as admin
	uri = api.Router.GetRoute(http.MethodPut, api.putGroupUserHandler, map[string]string{
		"permGroupName": g.Name,
		"username":      u2.Username,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtRaw1, http.MethodPut, uri, sdk.GroupMember{
		Admin: true,
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var result sdk.Group
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	assert.Equal(t, 2, len(result.Members))
}

func Test_deleteGroupUserHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	g := &sdk.Group{Name: sdk.RandomString(10)}
	u1, jwtRaw1 := assets.InsertLambdaUser(t, db, g)
	u2, jwtRaw2 := assets.InsertLambdaUser(t, db, g)
	u3, _ := assets.InsertLambdaUser(t, db, g)

	require.NoError(t, group.LoadOptions.WithMembers(context.TODO(), db, g))
	assert.Equal(t, 3, len(g.Members))

	// A group member should not be able to remove someone from the group
	uri := api.Router.GetRoute(http.MethodDelete, api.deleteGroupUserHandler, map[string]string{
		"permGroupName": g.Name,
		"username":      u3.Username,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw2, http.MethodDelete, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)

	// A group admin should be able to remove another member
	uri = api.Router.GetRoute(http.MethodDelete, api.deleteGroupUserHandler, map[string]string{
		"permGroupName": g.Name,
		"username":      u3.Username,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtRaw1, http.MethodDelete, uri, nil)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var result sdk.Group
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	assert.Equal(t, 2, len(result.Members))

	// A group admin should not be able to remove last admin
	uri = api.Router.GetRoute(http.MethodDelete, api.deleteGroupUserHandler, map[string]string{
		"permGroupName": g.Name,
		"username":      u1.Username,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtRaw1, http.MethodDelete, uri, nil)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	assets.SetUserGroupAdmin(t, db, g.ID, u2.ID)

	// A group admin should be able to remove himself if another admin exist
	uri = api.Router.GetRoute(http.MethodDelete, api.deleteGroupUserHandler, map[string]string{
		"permGroupName": g.Name,
		"username":      u1.Username,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtRaw1, http.MethodDelete, uri, nil)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "null", string(rec.Body.Bytes()))
}
