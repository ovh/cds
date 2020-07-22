package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/slug"
)

func generateTemplate(groupID int64, pipelineName string) *sdk.WorkflowTemplate {
	name := sdk.RandomString(10)
	return &sdk.WorkflowTemplate{
		GroupID: groupID,
		Name:    name,
		Slug:    slug.Convert(name),
		Workflow: base64.StdEncoding.EncodeToString([]byte(
			`name: [[.name]]
version: v2.0
workflow:
  Node-1:
    pipeline: ` + pipelineName,
		)),
		Pipelines: []sdk.PipelineTemplate{{
			Value: base64.StdEncoding.EncodeToString([]byte(
				`version: v1.0
name: ` + pipelineName + `
stages:
- Stage 1
jobs:
- job: Job 1
  stage: Stage 1
  steps:
  - script:
    - echo "Hello World!"`,
			)),
		}},
	}
}

func Test_postTemplateApplyHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	sharedInfraGroup, err := group.LoadByName(context.TODO(), api.mustDB(), "shared.infra")
	require.NoError(t, err)
	projectGroup := &proj.ProjectGroups[0].Group

	_, jwtAdmin := assets.InsertAdminUser(t, db)
	_, jwtLambdaInGroup := assets.InsertLambdaUser(t, db, projectGroup)
	_, jwtLambdaNotInGroup := assets.InsertLambdaUser(t, db)

	cases := []struct {
		Name  string
		Group *sdk.Group
		JWT   string
		Error bool
	}{
		{
			Name:  "Apply a shared.infra template by an admin",
			Group: sharedInfraGroup,
			JWT:   jwtAdmin,
		},
		{
			Name:  "Apply a shared.infra template by a lambda user",
			Group: sharedInfraGroup,
			JWT:   jwtLambdaInGroup,
		},
		{
			Name:  "Apply a lambda group template by an admin",
			Group: projectGroup,
			JWT:   jwtAdmin,
		},
		{
			Name:  "Apply a lambda group template by a lambda user",
			Group: projectGroup,
			JWT:   jwtLambdaInGroup,
		},
		{
			Name:  "Apply a lambda group template by a lambda user not in the group",
			Group: projectGroup,
			JWT:   jwtLambdaNotInGroup,
			Error: true,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			pipelineName := sdk.RandomString(10)
			template := generateTemplate(c.Group.ID, pipelineName)
			assert.NoError(t, workflowtemplate.Insert(db, template))

			uri := api.Router.GetRoute("POST", api.postTemplateApplyHandler, map[string]string{
				"groupName":    c.Group.Name,
				"templateSlug": template.Slug,
			})
			test.NotEmpty(t, uri)
			wtr := sdk.WorkflowTemplateRequest{
				ProjectKey:   proj.Key,
				WorkflowName: sdk.RandomString(10),
			}
			req := assets.NewJWTAuthentifiedRequest(t, c.JWT, "POST", uri+"?import=true", wtr)

			// execute the request
			rec := httptest.NewRecorder()
			api.Router.Mux.ServeHTTP(rec, req)

			// check result
			if c.Error {
				assert.NotEqual(t, 200, rec.Code)
				return
			}
			assert.Equal(t, 200, rec.Code)
			assert.Equal(t, wtr.WorkflowName, rec.Header().Get(sdk.ResponseWorkflowNameHeader))

			v, err := json.Marshal([]string{"Pipeline " + pipelineName + " successfully created", "Workflow " + wtr.WorkflowName + " has been created"})
			assert.NoError(t, err)

			assert.Equal(t, string(v), rec.Body.String())
		})
	}
}

func Test_postTemplateBulkHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, jwt := assets.InsertAdminUser(t, db)
	g, err := group.LoadByName(context.TODO(), api.mustDB(), "shared.infra")
	assert.NoError(t, err)

	name := sdk.RandomString(10)
	pipelineName := sdk.RandomString(10)
	template := &sdk.WorkflowTemplate{
		GroupID: g.ID,
		Name:    name,
		Slug:    slug.Convert(name),
		Workflow: base64.StdEncoding.EncodeToString([]byte(
			`name: [[.name]]
version: v2.0
workflow:
  Node-1:
    pipeline: ` + pipelineName,
		)),
		Pipelines: []sdk.PipelineTemplate{{
			Value: base64.StdEncoding.EncodeToString([]byte(
				`version: v1.0
name: ` + pipelineName + `
stages:
- Stage 1
jobs:
- job: Job 1
  stage: Stage 1
  steps:
  - script:
    - echo "Hello World!"`,
			)),
		}},
	}
	assert.NoError(t, workflowtemplate.Insert(db, template))

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	// prepare the request
	uri := api.Router.GetRoute("POST", api.postTemplateBulkHandler, map[string]string{
		"groupName":    g.Name,
		"templateSlug": template.Slug,
	})
	test.NotEmpty(t, uri)

	wtb := sdk.WorkflowTemplateBulk{
		Operations: []sdk.WorkflowTemplateBulkOperation{{
			Request: sdk.WorkflowTemplateRequest{
				ProjectKey:   proj.Key,
				WorkflowName: sdk.RandomString(10),
			},
		}, {
			Request: sdk.WorkflowTemplateRequest{
				ProjectKey:   proj.Key,
				WorkflowName: sdk.RandomString(10),
			},
		}},
	}
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, wtb)

	// execute the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)

	// check result
	assert.Equal(t, 200, rec.Code)

	var result sdk.WorkflowTemplateBulk
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))

	assert.Equal(t, 2, len(result.Operations))
}

func Test_getTemplateInstancesHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	projectOne := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	projectTwo := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	projectThree := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	sharedInfraGroup, err := group.LoadByName(context.TODO(), api.mustDB(), "shared.infra")
	require.NoError(t, err)
	projectOneGroup := &projectOne.ProjectGroups[0].Group
	projectTwoGroup := &projectTwo.ProjectGroups[0].Group

	_, jwtAdmin := assets.InsertAdminUser(t, db)
	_, jwtLambdaInGroupOneAndTwo := assets.InsertLambdaUser(t, db, projectOneGroup, projectTwoGroup)
	_, jwtLambdaInGroupOne := assets.InsertLambdaUser(t, db, projectOneGroup)

	template := generateTemplate(sharedInfraGroup.ID, sdk.RandomString(10))
	assert.NoError(t, workflowtemplate.Insert(db, template))

	apply := func(t *testing.T, projectKey, workflowName string) {
		uri := api.Router.GetRoute("POST", api.postTemplateApplyHandler, map[string]string{
			"groupName":    sharedInfraGroup.Name,
			"templateSlug": template.Slug,
		})
		test.NotEmpty(t, uri)
		wtr := sdk.WorkflowTemplateRequest{
			ProjectKey:   projectKey,
			WorkflowName: workflowName,
		}
		req := assets.NewJWTAuthentifiedRequest(t, jwtAdmin, "POST", uri+"?import=true", wtr)
		rec := httptest.NewRecorder()
		api.Router.Mux.ServeHTTP(rec, req)
		assert.Equal(t, 200, rec.Code)
		assert.Equal(t, wtr.WorkflowName, rec.Header().Get(sdk.ResponseWorkflowNameHeader))
	}

	workflowProjectOneName := sdk.RandomString(10)
	apply(t, projectOne.Key, workflowProjectOneName)

	workflowProjectTwoName := sdk.RandomString(10)
	apply(t, projectTwo.Key, workflowProjectTwoName)

	workflowProjectThreeName := sdk.RandomString(10)
	apply(t, projectThree.Key, workflowProjectThreeName)

	getInstances := func(t *testing.T, jwtToken string, expectedWorkflows []string) {
		uri := api.Router.GetRoute(http.MethodGet, api.getTemplateInstancesHandler, map[string]string{
			"groupName":    sharedInfraGroup.Name,
			"templateSlug": template.Slug,
		})
		test.NotEmpty(t, uri)
		req := assets.NewJWTAuthentifiedRequest(t, jwtToken, http.MethodGet, uri, nil)
		rec := httptest.NewRecorder()
		api.Router.Mux.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		var is []sdk.WorkflowTemplateInstance
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &is))

		require.Len(t, is, len(expectedWorkflows))
		for i := range expectedWorkflows {
			var found bool
			for j := range is {
				if is[j].WorkflowName == expectedWorkflows[i] {
					found = true
				}
			}
			assert.Truef(t, found, "Workflow %s should be in instances list", expectedWorkflows[i])
		}
	}

	t.Run("Get admin instances", func(t *testing.T) {
		getInstances(t, jwtAdmin, []string{workflowProjectOneName, workflowProjectTwoName, workflowProjectThreeName})
	})
	t.Run("Get instances for user in group one", func(t *testing.T) { getInstances(t, jwtLambdaInGroupOne, []string{workflowProjectOneName}) })
	t.Run("Get instances for user in group one and two", func(t *testing.T) {
		getInstances(t, jwtLambdaInGroupOneAndTwo, []string{workflowProjectOneName, workflowProjectTwoName})
	})

	getUsage := func(t *testing.T, jwtToken string, expectedWorkflows []string) {
		uri := api.Router.GetRoute(http.MethodGet, api.getTemplateUsageHandler, map[string]string{
			"groupName":    sharedInfraGroup.Name,
			"templateSlug": template.Slug,
		})
		test.NotEmpty(t, uri)
		req := assets.NewJWTAuthentifiedRequest(t, jwtToken, http.MethodGet, uri, nil)
		rec := httptest.NewRecorder()
		api.Router.Mux.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		var ws []sdk.Workflow
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &ws))

		require.Len(t, ws, len(expectedWorkflows))
		for i := range expectedWorkflows {
			var found bool
			for j := range ws {
				if ws[j].Name == expectedWorkflows[i] {
					found = true
				}
			}
			assert.Truef(t, found, "Workflow %s should be in instances list", expectedWorkflows[i])
		}
	}

	t.Run("Get admin usages", func(t *testing.T) {
		getUsage(t, jwtAdmin, []string{workflowProjectOneName, workflowProjectTwoName, workflowProjectThreeName})
	})
	t.Run("Get usage for user in group one", func(t *testing.T) { getUsage(t, jwtLambdaInGroupOne, []string{workflowProjectOneName}) })
	t.Run("Get usage for user in group one and two", func(t *testing.T) {
		getUsage(t, jwtLambdaInGroupOneAndTwo, []string{workflowProjectOneName, workflowProjectTwoName})
	})
}
