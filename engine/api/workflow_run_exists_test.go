package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_getWorkflowRunExistsHandler_RunExists(t *testing.T) {
	api, db, router := newTestAPI(t)

	// Create a CDN service consumer with a JWT
	cdnService, _, jwtCDN := assets.InitCDNService(t, db)
	t.Cleanup(func() { _ = services.Delete(db, cdnService) })

	// Create project, workflow, and a workflow run
	u, _ := assets.InsertAdminUser(t, db)
	consumer, _ := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadUserConsumerOptions.WithAuthentifiedUser)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	w := assets.InsertTestWorkflow(t, db, api.Cache, proj, sdk.RandomString(10))

	wrCreate, err := workflow.CreateRun(api.mustDB(), w, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	require.NoError(t, err)

	// Call the exists endpoint with the CDN JWT — should return 200
	vars := map[string]string{
		"id": fmt.Sprintf("%d", wrCreate.ID),
	}
	uri := router.GetRoute("GET", api.getWorkflowRunExistsHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtCDN, "GET", uri, nil)

	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func Test_getWorkflowRunExistsHandler_RunNotFound(t *testing.T) {
	api, db, router := newTestAPI(t)

	// Create a CDN service consumer with a JWT
	cdnService, _, jwtCDN := assets.InitCDNService(t, db)
	t.Cleanup(func() { _ = services.Delete(db, cdnService) })

	// Call the exists endpoint with a non-existent run ID — should return 404
	vars := map[string]string{
		"id": "999999999",
	}
	uri := router.GetRoute("GET", api.getWorkflowRunExistsHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtCDN, "GET", uri, nil)

	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func Test_getWorkflowRunExistsHandler_ForbiddenForNonCDN(t *testing.T) {
	api, db, router := newTestAPI(t)

	// Create a workflow run so we have a valid ID
	u, pass := assets.InsertAdminUser(t, db)
	consumer, _ := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadUserConsumerOptions.WithAuthentifiedUser)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	w := assets.InsertTestWorkflow(t, db, api.Cache, proj, sdk.RandomString(10))

	wrCreate, err := workflow.CreateRun(api.mustDB(), w, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	require.NoError(t, err)

	// Call with a regular admin user (not a service) — should return 403
	vars := map[string]string{
		"id": fmt.Sprintf("%d", wrCreate.ID),
	}
	uri := router.GetRoute("GET", api.getWorkflowRunExistsHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func Test_getWorkflowRunExistsHandler_DeletedRunNotFound(t *testing.T) {
	api, db, router := newTestAPI(t)

	// Create a CDN service consumer with a JWT
	cdnService, _, jwtCDN := assets.InitCDNService(t, db)
	t.Cleanup(func() { _ = services.Delete(db, cdnService) })

	// Create project, workflow, and a workflow run
	u, _ := assets.InsertAdminUser(t, db)
	consumer, _ := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadUserConsumerOptions.WithAuthentifiedUser)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	w := assets.InsertTestWorkflow(t, db, api.Cache, proj, sdk.RandomString(10))

	wrCreate, err := workflow.CreateRun(api.mustDB(), w, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	require.NoError(t, err)

	// Mark the run as to_delete
	require.NoError(t, workflow.MarkWorkflowRunsAsDelete(db, []int64{wrCreate.ID}))

	// Call the exists endpoint — run is marked to_delete, should return 404
	vars := map[string]string{
		"id": fmt.Sprintf("%d", wrCreate.ID),
	}
	uri := router.GetRoute("GET", api.getWorkflowRunExistsHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtCDN, "GET", uri, nil)

	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
