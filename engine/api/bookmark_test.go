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

func Test_postBookmarkHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	wkf := assets.InsertTestWorkflow(t, db, api.Cache, proj, sdk.RandomString(10))

	_, jwt := assets.InsertLambdaUser(t, db, &proj.ProjectGroups[0].Group)

	favProject := sdk.Bookmark{
		Type: sdk.ProjectBookmarkType,
		ID:   proj.Key,
	}

	favWorkflow := sdk.Bookmark{
		Type: sdk.WorkflowLegacyBookmarkType,
		ID:   proj.Key + "/" + wkf.Name,
	}

	uri := api.Router.GetRouteV2(http.MethodPost, api.postBookmarkHandler, nil)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodPost, uri, favProject)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	uri = api.Router.GetRouteV2(http.MethodPost, api.postBookmarkHandler, nil)
	req = assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodPost, uri, favWorkflow)
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	uri = api.Router.GetRouteV2(http.MethodGet, api.getBookmarksHandler, nil)
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
			if res[i].ID == favProject.ID {
				foundFavProject = true
			}
		case "workflow-legacy":
			if res[i].ID == favWorkflow.ID {
				foundFavWorkflow = true
			}
		}
	}
	assert.True(t, foundFavProject, "project favorite should be found")
	assert.True(t, foundFavWorkflow, "workflow favorite should be found")
}
