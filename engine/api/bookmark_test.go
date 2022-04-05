package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_postUserFavoriteHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	wkf := assets.InsertTestWorkflow(t, db, api.Cache, proj, sdk.RandomString(10))

	_, jwt := assets.InsertLambdaUser(t, db, &proj.ProjectGroups[0].Group)

	favProject := sdk.FavoriteParams{
		Type:       "project",
		ProjectKey: proj.Key,
	}

	favWorkflow := sdk.FavoriteParams{
		Type:         "workflow",
		ProjectKey:   proj.Key,
		WorkflowName: wkf.Name,
	}

	uri := api.Router.GetRoute(http.MethodPost, api.postUserFavoriteHandler, nil)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodPost, uri, favProject)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	uri = api.Router.GetRoute(http.MethodPost, api.postUserFavoriteHandler, nil)
	req = assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodPost, uri, favWorkflow)
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	uri = api.Router.GetRoute(http.MethodGet, api.getBookmarksHandler, nil)
	req = assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodGet, uri, nil)
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var res []sdk.Bookmark
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
	require.Len(t, res, 2)

	var foundFavProject, foundFavWorkflow bool
	for i := range res {
		switch res[i].Type {
		case "project":
			if res[i].Key == favProject.ProjectKey {
				foundFavProject = true
			}
		case "workflow":
			if res[i].Key == favWorkflow.ProjectKey && res[i].WorkflowName == favWorkflow.WorkflowName {
				foundFavWorkflow = true
			}
		}
	}
	assert.True(t, foundFavProject, "project favorite should be found")
	assert.True(t, foundFavWorkflow, "workflow favorite should be found")

	uri = api.Router.GetRoute(http.MethodGet, api.getProjectHandler, map[string]string{
		"permProjectKey": proj.Key,
	})
	req = assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodGet, uri, nil)
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var pRes sdk.Project
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &pRes))
	assert.True(t, pRes.Favorite, "project favorite flag should be set")

	uri = api.Router.GetRoute(http.MethodGet, api.getWorkflowHandler, map[string]string{
		"key":                      proj.Key,
		"permWorkflowNameAdvanced": wkf.Name,
	})
	req = assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodGet, uri, nil)
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var wRes sdk.Workflow
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &wRes))
	assert.True(t, wRes.Favorite, "workflow favorite flag should be set")
}
