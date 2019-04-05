package api

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/feature"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

type mockHTTPClient struct {
	f func(r *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(r *http.Request) (*http.Response, error) {
	return m.f(r)
}

func mock(f func(r *http.Request) (*http.Response, error)) sdk.HTTPClient {
	return &mockHTTPClient{f}
}

func writeError(w *http.Response, err error) (*http.Response, error) {
	body := new(bytes.Buffer)
	enc := json.NewEncoder(body)
	w.Body = ioutil.NopCloser(body)
	sdkErr := sdk.ExtractHTTPError(err, "")
	enc.Encode(sdkErr)
	w.StatusCode = sdkErr.Status
	return w, sdkErr
}

func Test_postImportAsCodeHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	u, pass := assets.InsertAdminUser(db)

	p := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	assert.NoError(t, repositoriesmanager.InsertForProject(db, p, &sdk.ProjectVCSServer{
		Name: "github",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))

	mockService := &sdk.Service{Name: "Test_postImportAsCodeHandler", Type: services.TypeRepositories}
	_ = services.Delete(api.mustDB(), mockService)
	test.NoError(t, services.Insert(api.mustDB(), mockService))

	mockVCSservice := &sdk.Service{Name: "Test_VCSService", Type: services.TypeVCS}
	test.NoError(t, services.Insert(api.mustDB(), mockVCSservice))

	//This is a mock for the repositories service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)

			switch r.URL.String() {
			case "/vcs/github/repos/myrepo/branches":
				bs := []sdk.VCSBranch{}
				b := sdk.VCSBranch{
					DisplayID: "master",
				}
				bs = append(bs, b)
				if err := enc.Encode(bs); err != nil {
					return writeError(w, err)
				}
			default:
				ope := new(sdk.Operation)
				btes, err := ioutil.ReadAll(r.Body)
				if err != nil {
					return writeError(w, err)
				}
				defer r.Body.Close()
				if err := json.Unmarshal(btes, ope); err != nil {
					return writeError(w, err)
				}
				ope.UUID = sdk.UUID()
				if err := enc.Encode(ope); err != nil {
					return writeError(w, err)
				}

				w.StatusCode = http.StatusCreated
			}

			return w, nil
		},
	)

	ope := `{"repo_fullname":"myrepo",  "vcs_server": "github", "url":"https://github.com/fsamin/go-repo.git","strategy":{"connection_type":"https","ssh_key":"","user":"","password":"","branch":"","default_branch":"master","pgp_key":""},"setup":{"checkout":{"branch":"master"}}}`

	uri := api.Router.GetRoute("POST", api.postImportAsCodeHandler, map[string]string{
		"permProjectKey": p.Key,
	})
	req, err := http.NewRequest("POST", uri, strings.NewReader(ope))
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)
	myOpe := new(sdk.Operation)
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), myOpe))
	assert.NotEmpty(t, myOpe.UUID)
}

func Test_getImportAsCodeHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	u, pass := assets.InsertAdminUser(db)

	mockService := &sdk.Service{Name: "Test_getImportAsCodeHandler", Type: services.TypeRepositories}
	_ = services.Delete(db, mockService)
	test.NoError(t, services.Insert(db, mockService))

	UUID := sdk.UUID()

	feature.SetClient(nil)

	//This is a mock for the repositories service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)

			ope := new(sdk.Operation)
			ope.URL = "https://github.com/fsamin/go-repo.git"
			ope.UUID = UUID
			ope.Status = sdk.OperationStatusDone
			ope.LoadFiles.Pattern = workflow.WorkflowAsCodePattern
			ope.LoadFiles.Results = map[string][]byte{
				"w-go-repo.yml": []byte(`name: w-go-repo
					version: v1.0
					pipeline: build
					application: go-repo
					pipeline_hooks:
					- type: RepositoryWebHook
					`),
			}
			if err := enc.Encode(ope); err != nil {
				return writeError(w, err)
			}

			w.StatusCode = http.StatusOK
			return w, nil
		},
	)

	uri := api.Router.GetRoute("GET", api.getImportAsCodeHandler, map[string]string{
		"uuid": UUID,
	})
	req, err := http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	myOpe := new(sdk.Operation)
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), myOpe))
	assert.NotEmpty(t, myOpe.UUID)
}

func Test_postPerformImportAsCodeHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	u, pass := assets.InsertAdminUser(db)

	assert.NoError(t, workflow.CreateBuiltinWorkflowHookModels(db))

	//Insert Project
	pkey := sdk.RandomString(10)
	_ = assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)

	mockService := &sdk.Service{Name: "Test_postPerformImportAsCodeHandler_Repo", Type: services.TypeRepositories}
	_ = services.Delete(db, mockService)
	test.NoError(t, services.Insert(db, mockService))

	mockService = &sdk.Service{Name: "Test_postPerformImportAsCodeHandler_VCS", Type: services.TypeHooks}
	services.Delete(db, mockService)
	test.NoError(t, services.Insert(db, mockService))

	UUID := sdk.UUID()

	//This is a mock for the repositories service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			t.Logf("RequestURI: %s", r.URL.Path)
			switch r.URL.Path {
			case "/task/bulk":
				hooks := map[string]sdk.WorkflowNodeHook{}
				if err := service.UnmarshalBody(r, &hooks); err != nil {
					return nil, sdk.WithStack(err)
				}

				body := new(bytes.Buffer)
				w := new(http.Response)
				enc := json.NewEncoder(body)
				w.Body = ioutil.NopCloser(body)
				if err := enc.Encode(hooks); err != nil {
					return writeError(w, err)
				}

				w.StatusCode = http.StatusOK
				return w, nil
			default:
				body := new(bytes.Buffer)
				w := new(http.Response)
				enc := json.NewEncoder(body)
				w.Body = ioutil.NopCloser(body)

				ope := new(sdk.Operation)
				ope.URL = "https://github.com/fsamin/go-repo.git"
				ope.UUID = UUID
				ope.VCSServer = "github"
				ope.RepoFullName = "fsamin/go-repo"
				ope.RepositoryInfo = &sdk.OperationRepositoryInfo{
					Name:          "go-repo",
					FetchURL:      ope.URL,
					DefaultBranch: "master",
				}
				ope.Status = sdk.OperationStatusDone
				ope.LoadFiles.Pattern = workflow.WorkflowAsCodePattern
				ope.LoadFiles.Results = map[string][]byte{
					"w-go-repo.yml": []byte(`name: w-go-repo
version: v1.0
pipeline: build
application: go-repo`),
					"go-repo.app.yml": []byte(`name: go-repo
version: v1.0`),
					"go-repo.pip.yml": []byte(`name: build
version: v1.0`),
				}
				if err := enc.Encode(ope); err != nil {
					return writeError(w, err)
				}

				w.StatusCode = http.StatusOK
				return w, nil
			}
		},
	)

	uri := api.Router.GetRoute("POST", api.postPerformImportAsCodeHandler, map[string]string{
		"permProjectKey": pkey,
		"uuid":           UUID,
	})
	req, err := http.NewRequest("POST", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	t.Logf(w.Body.String())
}
