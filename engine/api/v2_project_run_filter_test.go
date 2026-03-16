package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GetProjectRunFilters(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertLambdaUser(t, db)

	assets.InsertRBAcProject(t, db, sdk.ProjectRoleRead, proj.Key, *user1)

	// Create filters
	filter1 := &sdk.ProjectRunFilter{
		ProjectKey: proj.Key,
		Name:       "Filter A",
		Value:      "status:Success",
		Sort:       "started:desc",
		Order:      0,
	}
	filter2 := &sdk.ProjectRunFilter{
		ProjectKey: proj.Key,
		Name:       "Filter B",
		Value:      "branch:main",
		Sort:       "last_modified:asc",
		Order:      1,
	}

	require.NoError(t, project.InsertRunFilter(context.TODO(), db, filter1))
	require.NoError(t, project.InsertRunFilter(context.TODO(), db, filter2))

	// GET filters
	vars := map[string]string{
		"projectKey": proj.Key,
	}
	uri := api.Router.GetRouteV2("GET", api.getProjectRunFiltersHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, user1, pass, "GET", uri, nil)

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	var filters []sdk.ProjectRunFilter
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &filters))
	require.Len(t, filters, 2)
	assert.Equal(t, "Filter A", filters[0].Name)
	assert.Equal(t, "Filter B", filters[1].Name)
}

func Test_GetProjectRunFilters_RBAC(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertLambdaUser(t, db)

	// User without projectRead permission
	vars := map[string]string{
		"projectKey": proj.Key,
	}
	uri := api.Router.GetRouteV2("GET", api.getProjectRunFiltersHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, user1, pass, "GET", uri, nil)

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 403, w.Code)
}

func Test_PostProjectRunFilter(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertLambdaUser(t, db)

	assets.InsertRBAcProject(t, db, sdk.ProjectRoleManage, proj.Key, *user1)

	filter := sdk.ProjectRunFilter{
		Name:  "My Filter",
		Value: "status:Success branch:main",
		Sort:  "started:desc",
	}

	vars := map[string]string{
		"projectKey": proj.Key,
	}
	uri := api.Router.GetRouteV2("POST", api.postProjectRunFilterHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, user1, pass, "POST", uri, filter)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 201, w.Code)

	var created sdk.ProjectRunFilter
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Equal(t, "My Filter", created.Name)
	assert.Equal(t, "status:Success branch:main", created.Value)
	assert.Equal(t, "started:desc", created.Sort)
	assert.Equal(t, int64(0), created.Order)
	assert.NotEmpty(t, created.ID)
	assert.NotEmpty(t, created.LastModified)
}

func Test_PostProjectRunFilter_Validation(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertLambdaUser(t, db)

	assets.InsertRBAcProject(t, db, sdk.ProjectRoleManage, proj.Key, *user1)

	vars := map[string]string{
		"projectKey": proj.Key,
	}

	testCases := []struct {
		name           string
		filter         sdk.ProjectRunFilter
		expectedStatus int
	}{
		{
			name: "Empty name",
			filter: sdk.ProjectRunFilter{
				Name:  "",
				Value: "status:Success",
				Sort:  "started:desc",
			},
			expectedStatus: 400,
		},
		{
			name: "Name too long",
			filter: sdk.ProjectRunFilter{
				Name:  sdk.RandomString(101),
				Value: "status:Success",
				Sort:  "started:desc",
			},
			expectedStatus: 400,
		},
		{
			name: "Empty value",
			filter: sdk.ProjectRunFilter{
				Name:  "My Filter",
				Value: "",
				Sort:  "started:desc",
			},
			expectedStatus: 400,
		},
		{
			name: "Invalid sort",
			filter: sdk.ProjectRunFilter{
				Name:  "My Filter",
				Value: "status:Success",
				Sort:  "invalid:sort",
			},
			expectedStatus: 400,
		},
		{
			name: "Negative order",
			filter: sdk.ProjectRunFilter{
				Name:  "My Filter",
				Value: "status:Success",
				Sort:  "started:desc",
				Order: -1,
			},
			expectedStatus: 400,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			uri := api.Router.GetRouteV2("POST", api.postProjectRunFilterHandler, vars)
			test.NotEmpty(t, uri)
			req := assets.NewAuthentifiedRequest(t, user1, pass, "POST", uri, tc.filter)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			api.Router.Mux.ServeHTTP(w, req)
			require.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

func Test_PostProjectRunFilter_Conflict(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertLambdaUser(t, db)

	assets.InsertRBAcProject(t, db, sdk.ProjectRoleManage, proj.Key, *user1)

	// Create first filter
	filter1 := &sdk.ProjectRunFilter{
		ProjectKey: proj.Key,
		Name:       "My_Filter",
		Value:      "status:Success",
		Sort:       "started:desc",
		Order:      0,
	}
	require.NoError(t, project.InsertRunFilter(context.TODO(), db, filter1))

	// Try to create duplicate
	filter2 := sdk.ProjectRunFilter{
		Name:  "My_Filter",
		Value: "status:Fail",
		Sort:  "started:asc",
	}

	vars := map[string]string{
		"projectKey": proj.Key,
	}
	uri := api.Router.GetRouteV2("POST", api.postProjectRunFilterHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, user1, pass, "POST", uri, filter2)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 409, w.Code)
}

func Test_PostProjectRunFilter_RBAC(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertLambdaUser(t, db)

	// User without projectManage permission
	assets.InsertRBAcProject(t, db, sdk.ProjectRoleRead, proj.Key, *user1)

	filter := sdk.ProjectRunFilter{
		Name:  "My Filter",
		Value: "status:Success",
		Sort:  "started:desc",
	}

	vars := map[string]string{
		"projectKey": proj.Key,
	}
	uri := api.Router.GetRouteV2("POST", api.postProjectRunFilterHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, user1, pass, "POST", uri, filter)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 403, w.Code)
}

func Test_PutProjectRunFilter(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertLambdaUser(t, db)

	assets.InsertRBAcProject(t, db, sdk.ProjectRoleManage, proj.Key, *user1)

	// Create filter
	filter := &sdk.ProjectRunFilter{
		ProjectKey: proj.Key,
		Name:       "My_Filter",
		Value:      "status:Success",
		Sort:       "started:desc",
		Order:      0,
	}
	require.NoError(t, project.InsertRunFilter(context.TODO(), db, filter))

	// Update order
	update := sdk.ProjectRunFilter{
		Order: 5,
	}

	vars := map[string]string{
		"projectKey": proj.Key,
		"filterName": "My_Filter",
	}
	uri := api.Router.GetRouteV2("PUT", api.putProjectRunFilterHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, user1, pass, "PUT", uri, update)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	var updated sdk.ProjectRunFilter
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &updated))
	assert.Equal(t, int64(5), updated.Order)
}

func Test_PutProjectRunFilter_RBAC(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertLambdaUser(t, db)

	// User without projectManage permission
	assets.InsertRBAcProject(t, db, sdk.ProjectRoleRead, proj.Key, *user1)

	filter := &sdk.ProjectRunFilter{
		ProjectKey: proj.Key,
		Name:       "My Filter",
		Value:      "status:Success",
		Sort:       "started:desc",
		Order:      0,
	}
	require.NoError(t, project.InsertRunFilter(context.TODO(), db, filter))

	update := sdk.ProjectRunFilter{
		Order: 5,
	}

	vars := map[string]string{
		"projectKey": proj.Key,
		"filterName": "My Filter",
	}
	uri := api.Router.GetRouteV2("PUT", api.putProjectRunFilterHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, user1, pass, "PUT", uri, update)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 403, w.Code)
}

func Test_DeleteProjectRunFilter(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertLambdaUser(t, db)

	assets.InsertRBAcProject(t, db, sdk.ProjectRoleManage, proj.Key, *user1)

	// Create filters with order 0, 1, 2
	filter0 := &sdk.ProjectRunFilter{ProjectKey: proj.Key, Name: "Filter_0", Value: "status:Success", Sort: "started:desc", Order: 0}
	filter1 := &sdk.ProjectRunFilter{ProjectKey: proj.Key, Name: "Filter_1", Value: "status:Fail", Sort: "started:desc", Order: 1}
	filter2 := &sdk.ProjectRunFilter{ProjectKey: proj.Key, Name: "Filter_2", Value: "branch:main", Sort: "started:desc", Order: 2}

	require.NoError(t, project.InsertRunFilter(context.TODO(), db, filter0))
	require.NoError(t, project.InsertRunFilter(context.TODO(), db, filter1))
	require.NoError(t, project.InsertRunFilter(context.TODO(), db, filter2))

	// Delete filter with order 1
	vars := map[string]string{
		"projectKey": proj.Key,
		"filterName": "Filter_1",
	}
	uri := api.Router.GetRouteV2("DELETE", api.deleteProjectRunFilterHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, user1, pass, "DELETE", uri, nil)

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 204, w.Code)

	// Check that orders were recomputed
	filters, err := project.LoadRunFiltersByProjectKey(context.TODO(), db, proj.Key)
	require.NoError(t, err)
	require.Len(t, filters, 2)
	assert.Equal(t, "Filter_0", filters[0].Name)
	assert.Equal(t, int64(0), filters[0].Order)
	assert.Equal(t, "Filter_2", filters[1].Name)
	assert.Equal(t, int64(1), filters[1].Order) // Should be recomputed from 2 to 1
}

func Test_DeleteProjectRunFilter_RBAC(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertLambdaUser(t, db)

	// User without projectManage permission
	assets.InsertRBAcProject(t, db, sdk.ProjectRoleRead, proj.Key, *user1)

	filter := &sdk.ProjectRunFilter{
		ProjectKey: proj.Key,
		Name:       "My Filter",
		Value:      "status:Success",
		Sort:       "started:desc",
		Order:      0,
	}
	require.NoError(t, project.InsertRunFilter(context.TODO(), db, filter))

	vars := map[string]string{
		"projectKey": proj.Key,
		"filterName": "My Filter",
	}
	uri := api.Router.GetRouteV2("DELETE", api.deleteProjectRunFilterHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, user1, pass, "DELETE", uri, nil)

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 403, w.Code)
}

func Test_ProjectRunFilter_UTF8Support(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertLambdaUser(t, db)

	assets.InsertRBAcProject(t, db, sdk.ProjectRoleManage, proj.Key, *user1)

	// Test with emojis and special characters
	filter := sdk.ProjectRunFilter{
		Name:  "🚀 Filter with émojis 日本語",
		Value: "status:Success branch:féature/test",
		Sort:  "started:desc",
	}

	vars := map[string]string{
		"projectKey": proj.Key,
	}
	uri := api.Router.GetRouteV2("POST", api.postProjectRunFilterHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, user1, pass, "POST", uri, filter)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 201, w.Code)

	var created sdk.ProjectRunFilter
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Equal(t, "🚀 Filter with émojis 日本語", created.Name)
	assert.Equal(t, "status:Success branch:féature/test", created.Value)
}
