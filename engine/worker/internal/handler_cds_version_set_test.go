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
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
)

func Test_setVersionHandler(t *testing.T) {
	// Create test directory for current test
	fs := afero.NewOsFs()
	basedir := "test-" + test.GetTestName(t) + "-" + sdk.RandomString(10) + "-" + fmt.Sprintf("%d", time.Now().Unix())
	t.Logf("Creating worker basedir at %s", basedir)
	require.NoError(t, fs.MkdirAll(basedir, os.FileMode(0755)))

	// Setup test worker
	wk := &CurrentWorker{
		basedir: afero.NewBasePathFs(fs, basedir),
	}
	wk.currentJob.wJob = &sdk.WorkflowNodeJobRun{
		ID: 1,
	}

	// Prepare mock client for cds workers
	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })
	m := mock_cdsclient.NewMockWorkerInterface(ctrl)
	wk.client = m

	m.EXPECT().QueueJobSetVersion(gomock.Any(), int64(1), gomock.Any()).DoAndReturn(
		func(ctx context.Context, jobID int64, version sdk.WorkflowRunVersion) error {
			assert.Equal(t, "1.2.3", version.Value)
			return nil
		},
	).Times(1)

	buf, err := json.Marshal(workerruntime.CDSVersionSet{
		Value: "1.2.3",
	})
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "", bytes.NewBuffer(buf))
	require.NoError(t, err)
	w := httptest.NewRecorder()
	setVersionHandler(context.Background(), wk)(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	if w.Code != 200 {
		cdsError := sdk.DecodeError(w.Body.Bytes())
		t.Logf("set version return an error: %s", cdsError.Error())
		t.FailNow()
	}
}
