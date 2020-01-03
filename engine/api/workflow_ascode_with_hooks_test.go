package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-gorp/gorp"
	"github.com/golang/mock/gomock"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_WorkflowAsCodeWithNoHook_ShouldGiveAnAutomaticRepoWebHook(t *testing.T) {

	api, db, _, end := newTestAPI(t)
	defer end()

	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ gorp.SqlExecutor, _ []sdk.Service) services.Client {
		return servicesClients
	}

	// Create a project with a repository manager
	prjKey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, prjKey, prjKey)
	u, pass := assets.InsertLambdaUser(t, db, &proj.ProjectGroups[0].Group)

	require.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name: "github",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))

	// Perform a "import as-code operation" to create a new workflow
	ope := `{"repo_fullname":"myrepo",  "vcs_server": "github", "url":"https://github.com/fsamin/go-repo.git","strategy":{"connection_type":"https","ssh_key":"","user":"","password":"","branch":"","default_branch":"master","pgp_key":""},"setup":{"checkout":{"branch":"master"}}}`
	uri := api.Router.GetRoute("POST", api.postImportAsCodeHandler, map[string]string{
		"permProjectKey": proj.Key,
	})

	UUID := sdk.UUID()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/operations", gomock.Any(), gomock.Any()).
		Return(nil, 201, nil)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/myrepo/branches", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				bs := []sdk.VCSBranch{}
				b := sdk.VCSBranch{
					DisplayID: "master",
				}
				bs = append(bs, b)
				out = bs
				return nil, 200, nil
			},
		)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/operations/"+UUID, gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}) (http.Header, int, error) {
				fmt.Printf(">>> %T, %v", out, out)
				ope := new(sdk.Operation)
				ope.URL = "https://github.com/fsamin/go-repo.git"
				ope.UUID = UUID
				ope.Status = sdk.OperationStatusDone
				ope.RepositoryInfo = new(sdk.OperationRepositoryInfo)
				ope.RepositoryInfo.DefaultBranch = "master"
				ope.LoadFiles.Pattern = workflow.WorkflowAsCodePattern
				ope.LoadFiles.Results = map[string][]byte{
					"w-go-repo.yml": []byte(`
name: w-go-repo
version: v1.0
pipeline: build
application: go-repo
pipeline_hooks:
- type: RepositoryWebHook
`,
					),
				}
				*(out.(*sdk.Operation)) = *ope
				return nil, 200, nil
			},
		)

	req, err := http.NewRequest("POST", uri, strings.NewReader(ope))
	require.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)

	uri = api.Router.GetRoute("POST", api.postPerformImportAsCodeHandler, map[string]string{
		"permProjectKey": prjKey,
		"uuid":           UUID,
	})
	req, err = http.NewRequest("POST", uri, nil)
	require.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	t.Logf(w.Body.String())

	wk, err := workflow.Load(context.Background(), db, api.Cache, proj, "w-go-repo", workflow.LoadOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, wk)

}
