package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func Test_releaseApplicationWorkflowHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	mockVCSSservice, _ := assets.InsertService(t, db, "Test_releaseApplicationWorkflowHandlerVCS", sdk.TypeVCS)
	mockCDNService, _, _ := assets.InitCDNService(t, db)
	defer func() {
		_ = services.Delete(db, mockCDNService)
		_ = services.Delete(db, mockVCSSservice)
	}()

	//This is a mock for the repositories service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			body := new(bytes.Buffer)
			wri := new(http.Response)
			enc := json.NewEncoder(body)
			wri.Body = ioutil.NopCloser(body)

			switch r.URL.String() {
			case "/vcs/github/repos/myproj/myapp":
				repo := sdk.VCSRepo{
					ID:           "1",
					Name:         "bar",
					URL:          "url",
					Fullname:     "foo/bar",
					HTTPCloneURL: "",
					Slug:         "",
					SSHCloneURL:  "",
				}
				if err := enc.Encode(repo); err != nil {
					return writeError(wri, err)
				}
			case "/vcs/github/repos/myproj/myapp/branches/?branch=&default=true":
				b := sdk.VCSBranch{
					DisplayID: "master",
					Default:   true,
				}
				if err := enc.Encode(b); err != nil {
					return writeError(wri, err)
				}
			case "/vcs/github/repos/myproj/myapp/branches/?branch=master&default=false":
				b := sdk.VCSBranch{
					DisplayID: "master",
					Default:   true,
				}
				if err := enc.Encode(b); err != nil {
					return writeError(wri, err)
				}
			case "/vcs/github/repos/myproj/myapp/commits/":
				c := sdk.VCSCommit{
					URL:       "url",
					Message:   "Msg",
					Timestamp: time.Now().Unix(),
					Hash:      "123",
				}
				if err := enc.Encode(c); err != nil {
					return writeError(wri, err)
				}
			case "/vcs/github/repos/myproj/myapp/branches/?branch=my-branch&default=false":
				b := sdk.VCSBranch{
					DisplayID: "my-branch",
					Default:   true,
				}
				if err := enc.Encode(b); err != nil {
					return writeError(wri, err)
				}
				wri.StatusCode = http.StatusCreated
			case "/vcs/github/repos/myproj/myapp/releases":
				r := sdk.VCSRelease{
					ID:        0,
					UploadURL: "upload-url",
				}
				if err := enc.Encode(r); err != nil {
					return writeError(wri, err)
				}
				wri.StatusCode = http.StatusOK
			}
			return wri, nil
		},
	)

	f := func(t *testing.T, db gorpmapper.SqlExecutorWithTx, _ *sdk.Pipeline, app *sdk.Application) {
		app.VCSServer = "github"
		app.RepositoryFullname = "myproj/myapp"
		app.RepositoryStrategy = sdk.RepositoryStrategy{
			ConnectionType: "https",
		}
		require.NoError(t, application.Update(db, app))
	}

	ctx := testRunWorkflow(t, api, router, f)
	testGetWorkflowJobAsWorker(t, api, db, router, &ctx)
	assert.NotNil(t, ctx.job)

	// Register the worker
	testRegisterWorker(t, api, db, router, &ctx)
	// Register the hatchery
	testRegisterHatchery(t, api, db, router, &ctx)

	uri := router.GetRoute("POST", api.releaseApplicationWorkflowHandler, map[string]string{
		"key":              ctx.project.Key,
		"permWorkflowName": ctx.workflow.Name,
		"number":           fmt.Sprintf("%d", ctx.run.Number),
		"nodeRunID":        fmt.Sprintf("%d", ctx.job.WorkflowNodeRunID),
	})
	test.NotEmpty(t, uri)
	rec := httptest.NewRecorder()
	req := assets.NewJWTAuthentifiedRequest(t, ctx.workerToken, "POST", uri, sdk.WorkflowNodeRunRelease{
		TagName:        "my_tag",
		ReleaseTitle:   "my_release",
		ReleaseContent: "my_content",
	})
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 204, rec.Code)
}
