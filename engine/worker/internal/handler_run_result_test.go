package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_addRunResultStaticFileHandler(t *testing.T) {
	// Create test directory for current test
	fs := afero.NewOsFs()
	basedir := "test-" + test.GetTestName(t) + "-" + sdk.RandomString(10) + "-" + fmt.Sprintf("%d", time.Now().Unix())
	t.Logf("Creating worker basedir at %s", basedir)
	require.NoError(t, fs.MkdirAll(basedir, os.FileMode(0755)))

	// Setup test worker
	wk := &CurrentWorker{basedir: afero.NewBasePathFs(fs, basedir)}
	wk.currentJob.wJob = &sdk.WorkflowNodeJobRun{ID: 1}

	// Prepare mock client for cds workers
	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })
	m := mock_cdsclient.NewMockWorkerInterface(ctrl)
	wk.client = m

	m.EXPECT().QueueWorkflowRunResultCheck(gomock.Any(), int64(1), gomock.Any()).DoAndReturn(
		func(ctx context.Context, jobID int64, check sdk.WorkflowRunResultCheck) (int, error) {
			assert.Equal(t, sdk.WorkflowRunResultTypeStaticFile, check.ResultType)
			return 200, nil
		},
	).Times(1)
	m.EXPECT().QueueWorkflowRunResultsAdd(gomock.Any(), int64(1), gomock.Any()).DoAndReturn(
		func(ctx context.Context, jobID int64, result sdk.WorkflowRunResult) (int, error) {
			assert.Equal(t, sdk.WorkflowRunResultTypeStaticFile, result.Type)
			var reqArgs sdk.WorkflowRunResultStaticFile
			require.NoError(t, sdk.JSONUnmarshal(result.DataRaw, &reqArgs))
			assert.Equal(t, "http://locat.local/static/foo.html", reqArgs.RemoteURL)
			return 200, nil
		},
	).Times(1)

	v := sdk.WorkflowRunResultStaticFile{
		WorkflowRunResultArtifactCommon: sdk.WorkflowRunResultArtifactCommon{
			Name: "foo",
		},
		RemoteURL: "http://locat.local/static/foo.html",
	}
	buf, err := json.Marshal(v)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "", bytes.NewBuffer(buf))
	require.NoError(t, err)
	w := httptest.NewRecorder()
	addRunResultStaticFileHandler(context.Background(), wk)(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	if w.Code != 200 {
		cdsError := sdk.DecodeError(w.Body.Bytes())
		t.Logf("add run result return an error: %v", cdsError.Error())
		t.FailNow()
	}
}

func Test_addRunResultArtifactManagerHandler(t *testing.T) {
	// Create test directory for current test
	fs := afero.NewOsFs()
	basedir := "test-" + test.GetTestName(t) + "-" + sdk.RandomString(10) + "-" + fmt.Sprintf("%d", time.Now().Unix())
	t.Logf("Creating worker basedir at %s", basedir)
	require.NoError(t, fs.MkdirAll(basedir, os.FileMode(0755)))

	// Setup test worker
	wk := &CurrentWorker{basedir: afero.NewBasePathFs(fs, basedir)}
	wk.currentJob.wJob = &sdk.WorkflowNodeJobRun{ID: 1}

	// Prepare mock client for cds workers
	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })
	m := mock_cdsclient.NewMockWorkerInterface(ctrl)
	wk.client = m

	m.EXPECT().QueueWorkflowRunResultCheck(gomock.Any(), int64(1), gomock.Any()).DoAndReturn(
		func(ctx context.Context, jobID int64, check sdk.WorkflowRunResultCheck) (int, error) {
			assert.Equal(t, sdk.WorkflowRunResultTypeArtifactManager, check.ResultType)
			return 200, nil
		},
	).Times(1)
	m.EXPECT().QueueWorkflowRunResultsAdd(gomock.Any(), int64(1), gomock.Any()).DoAndReturn(
		func(ctx context.Context, jobID int64, result sdk.WorkflowRunResult) (int, error) {
			assert.Equal(t, sdk.WorkflowRunResultTypeArtifactManager, result.Type)
			var reqArgs sdk.WorkflowRunResultArtifactManager
			require.NoError(t, sdk.JSONUnmarshal(result.DataRaw, &reqArgs))
			assert.Equal(t, "foo", reqArgs.Name)
			assert.Equal(t, "my-repo", reqArgs.RepoName)
			return 200, nil
		},
	).Times(1)

	v := sdk.WorkflowRunResultArtifactManager{
		WorkflowRunResultArtifactCommon: sdk.WorkflowRunResultArtifactCommon{
			Name: "foo",
		},
		RepoName: "my-repo",
	}
	buf, err := json.Marshal(v)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "", bytes.NewBuffer(buf))
	require.NoError(t, err)
	w := httptest.NewRecorder()
	addRunResultArtifactManagerHandler(context.Background(), wk)(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	if w.Code != 200 {
		cdsError := sdk.DecodeError(w.Body.Bytes())
		t.Logf("add run result return an error: %v", cdsError.Error())
		t.FailNow()
	}
}
