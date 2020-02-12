package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/olivere/elastic.v6"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

type testTimelineEvent struct {
	ProjectKey   string `json:"project_key"`
	WorkflowName string `json:"workflow_name"`
}

func (e testTimelineEvent) String() string { return e.ProjectKey + "/" + e.WorkflowName }

func Test_getTimelineHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()

	// Create two projects with workflows
	proj1 := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	_ = assets.InsertTestWorkflow(t, db, api.Cache, proj1, "workflow1")
	_ = assets.InsertTestWorkflow(t, db, api.Cache, proj1, "workflow2")
	proj2 := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	_ = assets.InsertTestWorkflow(t, db, api.Cache, proj2, "workflow1")
	project1Group := &proj1.ProjectGroups[0].Group

	// Create four users (a maitainer, one in project 1, one in project 1 with muted workflow and one without projects)
	_, jwtMaintainer := assets.InsertMaintainerUser(t, api.mustDB())
	_, jwtLambdaInGroup := assets.InsertLambdaUser(t, api.mustDB(), project1Group)
	lambdaIngroupWithMuted, jwtLambdaInGroupWithMuted := assets.InsertLambdaUser(t, api.mustDB(), project1Group)
	_, jwtLambdaNotInGroup := assets.InsertLambdaUser(t, api.mustDB())
	require.NoError(t, user.InsertTimelineFilter(db, sdk.TimelineFilter{
		Projects: []sdk.ProjectFilter{
			{
				Key:           proj1.Key,
				WorkflowNames: []string{"workflow2"},
			},
		},
	}, lambdaIngroupWithMuted.ID))

	// This is a mock for the elastic service
	mockElasticService, _ := assets.InsertService(t, db, "Test_getTimelineHandler", services.TypeElasticsearch)
	defer func() {
		_ = services.Delete(db, mockElasticService) // nolint
	}()
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			t.Logf("[MOCK] %s %v", r.Method, r.URL)

			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)
			requestBody := new(bytes.Buffer)
			if _, err := requestBody.ReadFrom(r.Body); err != nil {
				return writeError(w, err)
			}

			switch r.URL.String() {
			case "/events":
				// simulate elastic query for given filters
				var filters sdk.EventFilter
				if err := json.Unmarshal(requestBody.Bytes(), &filters); err != nil {
					return writeError(w, err)
				}

				allEvents := []testTimelineEvent{
					{ProjectKey: proj1.Key, WorkflowName: "workflow1"},
					{ProjectKey: proj1.Key, WorkflowName: "workflow2"},
					{ProjectKey: proj2.Key, WorkflowName: "workflow1"},
				}

				var res []testTimelineEvent
				for _, f := range filters.Filter.Projects {
					for _, w := range f.WorkflowNames {
						for _, e := range allEvents {
							if e.ProjectKey == f.Key && e.WorkflowName == w {
								res = append(res, e)
							}
						}
					}
				}

				var hits []elastic.SearchHit
				for i := range res {
					buf, _ := json.Marshal(res[i])
					raw := json.RawMessage(buf)
					hits = append(hits, elastic.SearchHit{Source: &raw})
				}

				if err := enc.Encode(hits); err != nil {
					return writeError(w, err)
				}
			default:
				return writeError(w, fmt.Errorf("route %s must not be called", r.URL.String()))
			}
			return w, nil
		},
	)

	cases := []struct {
		Name    string
		JWT     string
		Expects []testTimelineEvent
	}{
		{
			Name: "Maintainer should get all events",
			JWT:  jwtMaintainer,
			Expects: []testTimelineEvent{
				{ProjectKey: proj1.Key, WorkflowName: "workflow1"},
				{ProjectKey: proj1.Key, WorkflowName: "workflow2"},
				{ProjectKey: proj2.Key, WorkflowName: "workflow1"},
			},
		},
		{
			Name: "Lambda user in a group",
			JWT:  jwtLambdaInGroup,
			Expects: []testTimelineEvent{
				{ProjectKey: proj1.Key, WorkflowName: "workflow1"},
				{ProjectKey: proj1.Key, WorkflowName: "workflow2"},
			},
		},
		{
			Name: "Lambda user in a group with muted workflow",
			JWT:  jwtLambdaInGroupWithMuted,
			Expects: []testTimelineEvent{
				{ProjectKey: proj1.Key, WorkflowName: "workflow1"},
			},
		},
		{
			Name:    "Lambda user not in a group",
			JWT:     jwtLambdaNotInGroup,
			Expects: []testTimelineEvent{},
		},
	}

	for _, c := range cases {
		uri := api.Router.GetRoute(http.MethodGet, api.getTimelineHandler, nil)
		require.NotEmpty(t, uri)
		req := assets.NewJWTAuthentifiedRequest(t, c.JWT, http.MethodGet, uri, nil)
		rec := httptest.NewRecorder()
		api.Router.Mux.ServeHTTP(rec, req)
		require.Equal(t, 200, rec.Code)

		var es []testTimelineEvent
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &es))

		require.Len(t, es, len(c.Expects))
		for _, e := range c.Expects {
			var found bool
			for i := range es {
				if e.String() == es[i].String() {
					found = true
				}
			}
			assert.True(t, found, "event for %s should be returned", e.String())
		}
	}
}
