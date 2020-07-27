package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
)

func Test_cachePushPullHandler(t *testing.T) {
	// Create test directory for current test
	fs := afero.NewOsFs()
	basedir := "test-" + test.GetTestName(t) + "-" + sdk.RandomString(10) + "-" + fmt.Sprintf("%d", time.Now().Unix())
	t.Logf("Creating worker basedir at %s", basedir)
	require.NoError(t, fs.MkdirAll(basedir, os.FileMode(0755)))

	// Setup test worker data for push and create run and tmp directories
	ctxPush := context.Background()
	wkPush := &CurrentWorker{
		basedir: afero.NewBasePathFs(fs, basedir),
	}
	pushJobInfo := sdk.WorkflowNodeJobRunData{}
	pushJobInfo.NodeJobRun.Job.Job.Action.Name = sdk.RandomString(10)

	wdPushFile, wdPushAbs, err := wkPush.setupWorkingDirectory(ctxPush, pushJobInfo)
	require.NoError(t, err)
	ctxPush = workerruntime.SetWorkingDirectory(ctxPush, wdPushFile)
	t.Logf("Setup push workspace at %s", wdPushFile.Name())

	tdPushFile, _, err := wkPush.setupTmpDirectory(ctxPush, pushJobInfo)
	require.NoError(t, err)
	ctxPush = workerruntime.SetTmpDirectory(ctxPush, tdPushFile)
	t.Logf("Setup push tmp directory at %s", tdPushFile.Name())

	wkPush.currentJob.context = ctxPush
	wkPush.currentJob.wJob = &sdk.WorkflowNodeJobRun{
		Parameters: []sdk.Parameter{{
			Name:  "cds.project",
			Value: "myProject",
		}},
	}

	// Setup test worker data for pull and create run directory
	ctxPull := context.Background()
	wkPull := &CurrentWorker{
		basedir: afero.NewBasePathFs(fs, basedir),
	}
	pullJobInfo := sdk.WorkflowNodeJobRunData{}
	pullJobInfo.NodeJobRun.Job.Job.Action.Name = sdk.RandomString(10)

	wkPull.currentJob.wJob = &sdk.WorkflowNodeJobRun{
		Parameters: []sdk.Parameter{{
			Name:  "cds.project",
			Value: "myProject",
		}},
	}

	// Create one file in push workspace and a another in worker basedir
	relativeFilePath := path.Join(wdPushFile.Name(), "relative.txt")
	t.Logf("Creating relative file at %s", relativeFilePath)
	require.NoError(t, afero.WriteFile(wkPush.basedir, relativeFilePath, []byte("relative"), os.FileMode(0755)))

	absoluteFilePath, err := filepath.Abs(afero.FullBaseFsPath(wkPush.basedir.(*afero.BasePathFs), "/absolute.txt"))
	require.NoError(t, err)
	t.Logf("Creating absolute file at %s", absoluteFilePath)
	require.NoError(t, ioutil.WriteFile(absoluteFilePath, []byte("absolute"), os.FileMode(0755)))

	// Prepare mock client for cds workers
	ctrl := gomock.NewController(t)
	m := mock_cdsclient.NewMockWorkerInterface(ctrl)
	wkPush.client = m
	wkPull.client = m

	var generatedTar bytes.Buffer
	var retryPush int
	m.EXPECT().WorkflowCachePush("myProject", "shared.infra", "myTag", gomock.Any(), gomock.Any()).DoAndReturn(
		func(projectKey, integrationName, ref string, tarContent io.Reader, size int) error {
			retryPush++
			if retryPush == 1 {
				partialRead := make([]byte, 10)
				l, err := tarContent.Read(partialRead)
				require.NoError(t, err)
				require.Equal(t, 10, l, "we should have read only 10 bytes of the tar")
				return fmt.Errorf("a fake error occured with http request")
			}
			_, err := io.Copy(&generatedTar, tarContent)
			require.NoError(t, err)
			return nil
		},
	).Times(2)
	m.EXPECT().WorkflowCachePull("myProject", "shared.infra", "myTag").DoAndReturn(
		func(projectKey, integrationName, ref string) (io.Reader, error) {
			return bytes.NewBuffer(generatedTar.Bytes()), nil
		},
	).Times(1)

	// Send cash push request for two files, one relative to workspace and another absolute to test basedir.
	buf, err := json.Marshal(sdk.Cache{
		Tag:              "myTag",
		WorkingDirectory: wdPushAbs,
		Files: []string{
			"relative.txt",
			absoluteFilePath,
		},
	})
	require.NoError(t, err)

	reqPush, err := http.NewRequest(http.MethodPost, "", bytes.NewBuffer(buf))
	require.NoError(t, err)
	w := httptest.NewRecorder()
	cachePushHandler(context.Background(), wkPush)(w, reqPush)
	assert.Equal(t, http.StatusOK, w.Code)
	if w.Code != 200 {
		cdsError := sdk.DecodeError(w.Body.Bytes())
		t.Logf("cache push return an error: %s", cdsError.Error())
		t.FailNow()
	}

	// Send pull request
	reqPull, err := http.NewRequest(http.MethodPost, "/cache/myTag/pull", bytes.NewBuffer(buf))
	require.NoError(t, err)
	q := reqPull.URL.Query()
	pullPath, err := filepath.Abs(afero.FullBaseFsPath(wkPush.basedir.(*afero.BasePathFs), "/pull"))
	require.NoError(t, err)
	q.Set("path", pullPath)
	reqPull.URL.RawQuery = q.Encode()

	// Create test router
	router := mux.NewRouter()
	router.HandleFunc("/cache/{ref}/pull", cachePullHandler(ctxPull, wkPull))

	w = httptest.NewRecorder()
	router.ServeHTTP(w, reqPull)
	assert.Equal(t, http.StatusOK, w.Code)
	if w.Code != 200 {
		cdsError := sdk.DecodeError(w.Body.Bytes())
		t.Logf("cache pull return an error: %s", cdsError.Error())
		t.FailNow()
	}

	expectedRelativePath := path.Join(pullPath, "relative.txt")
	expectedAbsolutePath := path.Join(pullPath, absoluteFilePath)

	_, err = os.Stat(expectedRelativePath)
	require.NoError(t, err, "relative pulled file should exists")

	_, err = os.Stat(expectedAbsolutePath)
	require.NoError(t, err, "absolute pulled file should exists")

	btsRelative, err := ioutil.ReadFile(expectedRelativePath)
	require.NoError(t, err)
	assert.Equal(t, "relative", string(btsRelative))

	btsAbsolute, err := ioutil.ReadFile(expectedAbsolutePath)
	require.NoError(t, err)
	assert.Equal(t, "absolute", string(btsAbsolute))
}
