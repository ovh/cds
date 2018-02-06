package api

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
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
	msg, errProcessed := sdk.ProcessError(err, "")
	sdkErr := sdk.Error{Message: msg}
	enc.Encode(sdkErr)
	w.StatusCode = errProcessed.Status
	return w, sdkErr
}

func Test_postImportAsCodeHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(db)

	repositoryService := services.NewRepository(func() *gorp.DbMap {
		return db
	}, api.Cache)
	mockService := &sdk.Service{Name: "Test_postImportAsCodeHandler", Type: services.TypeRepositories}
	repositoryService.Delete(mockService)
	test.NoError(t, repositoryService.Insert(mockService))

	//This is a mock for the repositories service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)

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
			return w, nil
		},
	)

	ope := `{"url":"https://github.com/fsamin/go-repo.git","strategy":{"connection_type":"https","ssh_key":"","user":"","password":"","branch":"","default_branch":"master","pgp_key":""},"setup":{"checkout":{"branch":"master"}}}`

	uri := api.Router.GetRoute("POST", api.postImportAsCodeHandler, nil)
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
	api, db, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(db)

	repositoryService := services.NewRepository(func() *gorp.DbMap {
		return db
	}, api.Cache)
	mockService := &sdk.Service{Name: "Test_getImportAsCodeHandler", Type: services.TypeRepositories}
	repositoryService.Delete(mockService)
	test.NoError(t, repositoryService.Insert(mockService))

	UUID := sdk.UUID()

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
			ope.LoadFiles.Pattern = workflowAsCodePattern
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
