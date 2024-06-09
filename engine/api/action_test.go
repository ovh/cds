package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func Test_getActionExportHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, jwt := assets.InsertAdminUser(t, db)

	grp := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	err := action.Insert(db, &sdk.Action{
		GroupID: &grp.ID,
		Type:    sdk.DefaultAction,
		Name:    "myAction",
	})
	assert.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"permGroupName":  grp.Name,
		"permActionName": "myAction",
	}
	uri := api.Router.GetRoute("GET", api.getActionExportHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())
}

func Test_postActionImportHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, jwt := assets.InsertAdminUser(t, db)

	uri := api.Router.GetRoute("POST", api.importActionHandler, nil)
	test.NotEmpty(t, uri)

	a := exportentities.Action{
		Name:        "myAction",
		Description: "MyDescription",
		Requirements: []exportentities.Requirement{
			{
				Binary: "bash",
			},
		},
		Parameters: map[string]exportentities.ParameterValue{
			"param1": {
				Description:  "this is my param",
				DefaultValue: "default value",
			},
		},
		Steps: []exportentities.Step{
			{
				Script: "echo {{.cds.pip.param1}}",
			},
		},
	}
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, nil)

	body, _ := yaml.Marshal(a)
	req.Body = io.NopCloser(bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	if rec.Code > 201 {
		t.Errorf("http code status %d", rec.Code)
	}

	//Check result
	t.Logf(">>%s", rec.Body.String())
}

func Test_postActionAuditRollbackHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, jwt := assets.InsertAdminUser(t, db)

	grp := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	a := sdk.Action{
		GroupID: &grp.ID,
		Type:    sdk.DefaultAction,
		Name:    "myAction",
		Parameters: []sdk.Parameter{
			{
				Name: "my-string",
				Type: sdk.StringParameter,
			},
			{
				Name: "my-bool",
				Type: sdk.BooleanParameter,
			},
		},
	}
	assert.NoError(t, action.Insert(db, &a))

	before, err := json.Marshal(sdk.Action{
		Type: sdk.DefaultAction,
		Name: "myAction",
		Parameters: []sdk.Parameter{
			{
				Name: "my-string",
				Type: sdk.StringParameter,
			},
		},
		Group: &sdk.Group{Name: grp.Name},
	})
	assert.NoError(t, err)

	after, err := json.Marshal(a)
	assert.NoError(t, err)

	aa := sdk.AuditAction{
		AuditCommon: sdk.AuditCommon{
			EventType: "ActionAdd",
			Created:   time.Now(),
		},
		ActionID:   a.ID,
		DataType:   "json",
		DataBefore: string(before),
		DataAfter:  string(after),
	}
	assert.NoError(t, action.InsertAudit(db, &aa))

	// prepare action rollback request
	uri := api.Router.GetRoute("POST", api.postActionAuditRollbackHandler, map[string]string{
		"permGroupName":  grp.Name,
		"permActionName": a.Name,
		"auditID":        fmt.Sprintf("%d", aa.ID),
	})
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, nil)

	// send action rollback request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)
	var aRollback sdk.Action
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &aRollback))

	assert.Equal(t, 1, len(aRollback.Parameters))
}

func Test_getActions(t *testing.T) {
	api, db, router := newTestAPI(t)

	_, jwtAdmin := assets.InsertAdminUser(t, db)

	g1 := &sdk.Group{Name: sdk.RandomString(10)}
	g2 := assets.InsertGroup(t, db)
	_, jwtGroupMember := assets.InsertLambdaUser(t, db, g1)

	a1 := sdk.Action{
		Name:    "A" + sdk.RandomString(10),
		GroupID: &g1.ID,
		Type:    sdk.DefaultAction,
	}
	assert.NoError(t, action.Insert(db, &a1))

	a2 := sdk.Action{
		Name:    "B" + sdk.RandomString(10),
		GroupID: &g1.ID,
		Type:    sdk.DefaultAction,
	}
	assert.NoError(t, action.Insert(db, &a2))

	a3 := sdk.Action{
		Name:    "C" + sdk.RandomString(10),
		GroupID: &g2.ID,
		Type:    sdk.DefaultAction,
	}
	assert.NoError(t, action.Insert(db, &a3))

	// getActionsHandler by admin
	uri := router.GetRoute(http.MethodGet, api.getActionsHandler, nil)
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtAdmin, http.MethodGet, uri, nil)
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	results := []sdk.Action{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &results))
	require.True(t, len(results) >= 3)

	// getActionsHandler by group member
	uri = router.GetRoute(http.MethodGet, api.getActionsHandler, nil)
	test.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtGroupMember, http.MethodGet, uri, nil)
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	results = []sdk.Action{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &results))
	require.True(t, len(results) >= 2)
	for _, r := range results {
		if r.GroupID != nil {
			assert.True(t, r.Group.Name == g1.Name || r.Group.Name == sdk.SharedInfraGroupName,
				"the group name is %s but should be %s or %s", r.Group.Name, g1.Name, sdk.SharedInfraGroupName)
		}
	}

	// getActionsForGroupHandler
	uri = router.GetRoute(http.MethodGet, api.getActionsForGroupHandler, map[string]string{
		"permGroupName": g2.Name,
	})
	test.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtAdmin, http.MethodGet, uri, nil)
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	results = []sdk.Action{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &results))
	require.True(t, len(results) >= 1)
	for _, r := range results {
		if r.GroupID != nil {
			assert.True(t, r.Group.Name == g2.Name || r.Group.Name == sdk.SharedInfraGroupName,
				"the group name is %s but should be %s or %s", r.Group.Name, g2.Name, sdk.SharedInfraGroupName)
		}
	}
}
