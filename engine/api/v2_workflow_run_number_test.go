package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
)

type runNumberFixture struct {
	api      *API
	db       *test.FakeTransaction
	proj     *sdk.Project
	vcs      *sdk.VCSProject
	repo     *sdk.ProjectRepository
	workflow string
	admin    *sdk.AuthentifiedUser
	adminPwd string
}

func newRunNumberFixture(t *testing.T) runNumberFixture {
	api, db, _ := newTestAPI(t)
	admin, pwd := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	e := sdk.Entity{
		Initiator:           &sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator()},
		Name:                sdk.RandomString(10),
		Type:                sdk.EntityTypeWorkflow,
		ProjectKey:          proj.Key,
		Ref:                 "refs/heads/master",
		Commit:              "123456",
		ProjectRepositoryID: repo.ID,
		Head:                true,
		Data:                "name: workflow",
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &e))

	return runNumberFixture{api: api, db: db, proj: proj, vcs: vcsServer, repo: repo, workflow: e.Name, admin: admin, adminPwd: pwd}
}

func (f runNumberFixture) do(t *testing.T, u *sdk.AuthentifiedUser, pwd, method, workflow string, body interface{}) *httptest.ResponseRecorder {
	handler := f.api.getWorkflowRunNumberHandler
	if method == "PUT" {
		handler = f.api.putWorkflowRunNumberHandler
	}
	uri := f.api.Router.GetRouteV2(method, handler, map[string]string{
		"projectKey":           f.proj.Key,
		"vcsIdentifier":        f.vcs.Name,
		"repositoryIdentifier": f.repo.Name,
		"workflow":             workflow,
	})
	require.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pwd, method, uri, body)
	w := httptest.NewRecorder()
	f.api.Router.Mux.ServeHTTP(w, req)
	return w
}

func (f runNumberFixture) show(t *testing.T) int64 {
	w := f.do(t, f.admin, f.adminPwd, "GET", f.workflow, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	var runNumber sdk.V2WorkflowRunNumber
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &runNumber))
	return runNumber.Value
}

func TestWorkflowRunNumber_SetMakesTheNextRunFollowIt(t *testing.T) {
	f := newRunNumberFixture(t)

	require.Equal(t, int64(0), f.show(t))

	w := f.do(t, f.admin, f.adminPwd, "PUT", f.workflow, sdk.V2WorkflowRunNumber{Value: 83})
	require.Equal(t, 200, w.Code, w.Body.String())
	require.Equal(t, int64(83), f.show(t))

	next, err := workflow_v2.WorkflowRunNextNumber(f.db, f.repo.ID, f.workflow)
	require.NoError(t, err)
	require.Equal(t, int64(84), next)
	require.Equal(t, int64(84), f.show(t))
}

// allocateRunNumbers moves the sequence of the fixture workflow up to n, as n triggered runs would.
func (f runNumberFixture) allocateRunNumbers(t *testing.T, n int) {
	for i := 0; i < n; i++ {
		_, err := workflow_v2.WorkflowRunNextNumber(f.db, f.repo.ID, f.workflow)
		require.NoError(t, err)
	}
}

func (f runNumberFixture) insertRun(t *testing.T, workflow string, runNumber int64) {
	wr := sdk.V2WorkflowRun{
		ProjectKey:   f.proj.Key,
		VCSServerID:  f.vcs.ID,
		VCSServer:    f.vcs.Name,
		RepositoryID: f.repo.ID,
		Repository:   f.repo.Name,
		WorkflowName: workflow,
		WorkflowSha:  "123456",
		WorkflowRef:  "refs/heads/master",
		RunAttempt:   1,
		RunNumber:    runNumber,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.V2WorkflowRunStatusSuccess,
		Initiator:    &sdk.V2Initiator{UserID: f.admin.ID, User: f.admin.Initiator()},
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), f.db, &wr))
}

func TestWorkflowRunNumber_RefusesANumberBelowTheHighestRun(t *testing.T) {
	f := newRunNumberFixture(t)
	f.allocateRunNumbers(t, 10)
	f.insertRun(t, f.workflow, 10)

	w := f.do(t, f.admin, f.adminPwd, "PUT", f.workflow, sdk.V2WorkflowRunNumber{Value: 5})
	require.Equal(t, 400, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "lower than the highest existing run number 10")
	require.Equal(t, int64(10), f.show(t))

	w = f.do(t, f.admin, f.adminPwd, "PUT", f.workflow, sdk.V2WorkflowRunNumber{Value: 10})
	require.Equal(t, 200, w.Code, w.Body.String())
	require.Equal(t, int64(10), f.show(t))
}

func TestWorkflowRunNumber_CanGoBackDownToTheHighestRun(t *testing.T) {
	f := newRunNumberFixture(t)
	f.allocateRunNumbers(t, 20)
	f.insertRun(t, f.workflow, 10)

	w := f.do(t, f.admin, f.adminPwd, "PUT", f.workflow, sdk.V2WorkflowRunNumber{Value: 15})
	require.Equal(t, 200, w.Code, w.Body.String())
	require.Equal(t, int64(15), f.show(t))

	next, err := workflow_v2.WorkflowRunNextNumber(f.db, f.repo.ID, f.workflow)
	require.NoError(t, err)
	require.Equal(t, int64(16), next)
}

func TestWorkflowRunNumber_IgnoresTheRunsOfOtherWorkflows(t *testing.T) {
	f := newRunNumberFixture(t)
	f.insertRun(t, sdk.RandomString(10), 50)

	w := f.do(t, f.admin, f.adminPwd, "PUT", f.workflow, sdk.V2WorkflowRunNumber{Value: 5})
	require.Equal(t, 200, w.Code, w.Body.String())
	require.Equal(t, int64(5), f.show(t))
}

func TestWorkflowRunNumber_ManagersCanSet(t *testing.T) {
	f := newRunNumberFixture(t)

	manager, managerPwd := assets.InsertLambdaUser(t, f.db)
	assets.InsertRBAcProject(t, f.db, sdk.ProjectRoleManage, f.proj.Key, *manager)

	w := f.do(t, manager, managerPwd, "PUT", f.workflow, sdk.V2WorkflowRunNumber{Value: 83})
	require.Equal(t, 200, w.Code, w.Body.String())
	require.Equal(t, int64(83), f.show(t))
}

func TestWorkflowRunNumber_RefusesANegativeNumber(t *testing.T) {
	f := newRunNumberFixture(t)

	w := f.do(t, f.admin, f.adminPwd, "PUT", f.workflow, sdk.V2WorkflowRunNumber{Value: -1})
	require.Equal(t, 400, w.Code, w.Body.String())
	require.Equal(t, int64(0), f.show(t))
}

func TestWorkflowRunNumber_UnknownWorkflow(t *testing.T) {
	f := newRunNumberFixture(t)

	w := f.do(t, f.admin, f.adminPwd, "GET", "unknown-workflow", nil)
	require.Equal(t, 404, w.Code, w.Body.String())

	w = f.do(t, f.admin, f.adminPwd, "PUT", "unknown-workflow", sdk.V2WorkflowRunNumber{Value: 1})
	require.Equal(t, 404, w.Code, w.Body.String())
}

func TestWorkflowRunNumber_ReadersCanShowButNotSet(t *testing.T) {
	f := newRunNumberFixture(t)
	db := f.db

	reader, readerPwd := assets.InsertLambdaUser(t, db)
	assets.InsertRBAcProject(t, db, sdk.ProjectRoleRead, f.proj.Key, *reader)

	w := f.do(t, reader, readerPwd, "GET", f.workflow, nil)
	require.Equal(t, 200, w.Code, w.Body.String())

	w = f.do(t, reader, readerPwd, "PUT", f.workflow, sdk.V2WorkflowRunNumber{Value: 83})
	require.Equal(t, 403, w.Code, w.Body.String())
	require.Equal(t, int64(0), f.show(t))
}
