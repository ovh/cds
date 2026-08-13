package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/yaml"
	"github.com/stretchr/testify/require"
)

// The endpoint has no project nor run to take a workflow name from, so the caller supplies it.
// Without one the preview of a template using [[.name]] does not match what a run would produce.
func Test_postGenerateWorkflowFromTemplateHandler_Name(t *testing.T) {
	api, db, _ := newTestAPI(t)

	user1, pass := assets.InsertLambdaUser(t, db)

	// parameters has no omitempty, the json schema rejects a null one
	var tmpl sdk.V2WorkflowTemplate
	require.NoError(t, yaml.Unmarshal([]byte(`name: myTemplate
parameters: []
spec: |-
  jobs:
    hello:
      runs-on: mymodel
      steps:
      - run: echo "workflow=[[.name]]"
`), &tmpl))

	generate := func(t *testing.T, req sdk.V2WorkflowTemplateGenerateRequest) sdk.V2WorkflowTemplateGenerateResponse {
		uri := api.Router.GetRouteV2("POST", api.postGenerateWorkflowFromTemplateHandler, nil)
		test.NotEmpty(t, uri)
		httpReq := assets.NewAuthentifiedRequest(t, user1, pass, "POST", uri, req)
		httpReq.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		api.Router.Mux.ServeHTTP(w, httpReq)
		require.Equal(t, 200, w.Code)

		var resp sdk.V2WorkflowTemplateGenerateResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		// The handler answers 200 even on a lint or resolution failure, so report what it said
		require.Empty(t, resp.Error, "generate returned an error: %s", resp.Error)
		return resp
	}

	resp := generate(t, sdk.V2WorkflowTemplateGenerateRequest{Template: tmpl, Name: "myworkflow"})
	require.Contains(t, resp.Workflow, `echo "workflow=myworkflow"`)

	// Omitting the name keeps the previous behaviour rather than inventing one
	respNoName := generate(t, sdk.V2WorkflowTemplateGenerateRequest{Template: tmpl})
	require.Contains(t, respNoName.Workflow, `echo "workflow="`)
	require.NotContains(t, respNoName.Workflow, "myworkflow")
}
