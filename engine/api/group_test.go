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
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

func Test_getProjectGroupHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	g := sdk.Group{Name: sdk.RandomString(10)}
	_, jwtRaw := assets.InsertLambdaUser(t, db, &g)

	p1 := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	_ = assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	p3 := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		ProjectID: p1.ID,
		GroupID:   g.ID,
		Role:      7,
	}))
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		ProjectID: p3.ID,
		GroupID:   g.ID,
		Role:      7,
	}))

	uri := api.Router.GetRoute(http.MethodGet, api.getProjectGroupHandler, map[string]string{
		"permGroupName": g.Name,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodGet, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var result []sdk.Project
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))

	require.Equal(t, 2, len(result))

	if result[0].Key == p1.Key {
		require.Equal(t, p3.Key, result[1].Key)
	} else if result[1].Key == p1.Key {
		require.Equal(t, p3.Key, result[0].Key)
	} else {
		t.Fail()
	}
}

func Test_getGroupHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

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
	api, db, _ := newTestAPI(t)

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
	api, db, _ := newTestAPI(t)

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
	api, db, _ := newTestAPI(t)

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
	api, db, _ := newTestAPI(t)

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
	api, db, _ := newTestAPI(t)

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
	api, db, _ := newTestAPI(t)

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
	api, db, _ := newTestAPI(t)

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
	assert.Equal(t, "null", rec.Body.String())
}

func Test_postImportGroupHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	g1Name := sdk.RandomString(10)
	u1, jwtRaw := assets.InsertLambdaUser(t, db, &sdk.Group{Name: g1Name})
	u2, _ := assets.InsertLambdaUser(t, db)

	// Get the group
	uri := api.Router.GetRoute(http.MethodGet, api.getGroupHandler, map[string]string{
		"permGroupName": g1Name,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodGet, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var g1 sdk.Group
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &g1))
	require.Len(t, g1.Members, 1)
	require.Equal(t, u1.ID, g1.Members[0].ID)

	// Add a new member
	g1.Members = append(g1.Members, sdk.GroupMember{ID: u2.ID})
	uri = api.Router.GetRoute(http.MethodPost, api.postGroupImportHandler, nil)
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodPost, uri, g1)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &g1))
	require.Len(t, g1.Members, 2)
	require.NotNil(t, g1.Members.GetUserByID(u1.ID))
	require.True(t, g1.Members.GetUserByID(u1.ID).Admin)
	require.NotNil(t, g1.Members.GetUserByID(u2.ID))
	require.False(t, g1.Members.GetUserByID(u2.ID).Admin)

	// Change name
	g1NewName := sdk.RandomString(10)
	g1.Name = g1NewName
	uri = api.Router.GetRoute(http.MethodPost, api.postGroupImportHandler, nil)
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodPost, uri, g1)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &g1))
	require.Equal(t, g1NewName, g1.Name)
	g1Name = g1.Name

	// Cannot change override another group
	g2 := assets.InsertGroup(t, db)
	g1.Name = g2.Name
	uri = api.Router.GetRoute(http.MethodPost, api.postGroupImportHandler, nil)
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodPost, uri, g1)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)

	// Cannot send members list without admin
	g1.Name = g1Name
	for i := range g1.Members {
		g1.Members[i].Admin = false
	}
	uri = api.Router.GetRoute(http.MethodPost, api.postGroupImportHandler, nil)
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodPost, uri, g1)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	var resultError sdk.Error
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resultError))
	require.Equal(t, "invalid given group members, at least one admin required", resultError.From)

	// Remove a member
	g1.Members = []sdk.GroupMember{{ID: u1.ID, Admin: true}}
	uri = api.Router.GetRoute(http.MethodPost, api.postGroupImportHandler, nil)
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodPost, uri, g1)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &g1))
	require.Len(t, g1.Members, 1)
	require.Equal(t, u1.ID, g1.Members[0].ID)
}

func Test_postImportGroupHandler_CheckOrganization(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, jwtAdmin := assets.InsertAdminUser(t, db)

	groupName := sdk.RandomString(10)
	u1, jwtLambda := assets.InsertLambdaUser(t, db, &sdk.Group{Name: groupName})
	require.NoError(t, user.InsertOrganization(context.TODO(), db, &user.Organization{
		AuthentifiedUserID: u1.ID,
		Organization:       "org1",
	}))
	u2, _ := assets.InsertLambdaUser(t, db)
	require.NoError(t, user.InsertOrganization(context.TODO(), db, &user.Organization{
		AuthentifiedUserID: u2.ID,
		Organization:       "org1",
	}))
	u3, _ := assets.InsertLambdaUser(t, db)
	require.NoError(t, user.InsertOrganization(context.TODO(), db, &user.Organization{
		AuthentifiedUserID: u3.ID,
		Organization:       "org2",
	}))

	// Get the group
	uri := api.Router.GetRoute(http.MethodGet, api.getGroupHandler, map[string]string{
		"permGroupName": groupName,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtLambda, http.MethodGet, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var group sdk.Group
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &group))
	require.Len(t, group.Members, 1)
	require.Equal(t, u1.ID, group.Members[0].ID)
	require.True(t, group.Members[0].Admin)

	// Add new member from same organization
	group.Members = append(group.Members, sdk.GroupMember{ID: u2.ID})
	uri = api.Router.GetRoute(http.MethodPost, api.postGroupImportHandler, nil)
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtLambda, http.MethodPost, uri, group)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &group))
	require.Len(t, group.Members, 2)
	require.NotNil(t, group.Members.GetUserByID(u1.ID))
	require.True(t, group.Members.GetUserByID(u1.ID).Admin)
	require.NotNil(t, group.Members.GetUserByID(u2.ID))
	require.False(t, group.Members.GetUserByID(u2.ID).Admin)

	// Try add new member from other organization
	group.Members = append(group.Members, sdk.GroupMember{ID: u3.ID})
	uri = api.Router.GetRoute(http.MethodPost, api.postGroupImportHandler, nil)
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtLambda, http.MethodPost, uri, group)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	var resultError sdk.Error
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resultError))
	require.Equal(t, "group members organization conflict \"org1\" and \"org2\"", resultError.From)

	// Try change group organization
	group.Members = sdk.GroupMembers{{ID: u3.ID, Admin: true}}
	uri = api.Router.GetRoute(http.MethodPost, api.postGroupImportHandler, nil)
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtAdmin, http.MethodPost, uri, group)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resultError))
	require.Equal(t, "can't change group organization", resultError.From)
}
