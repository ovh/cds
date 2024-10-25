package internal

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"strings"
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
	"github.com/ovh/cds/sdk/jws"
)

func Test_cachePushPullHandler(t *testing.T) {
	// Create test directory for current test
	fs := afero.NewOsFs()
	basedir := "test-" + test.GetTestName(t) + "-" + sdk.RandomString(10) + "-" + fmt.Sprintf("%d", time.Now().Unix())
	t.Logf("Creating worker basedir at %s", basedir)
	require.NoError(t, fs.MkdirAll(basedir, os.FileMode(0755)))

	workerKey, err := jws.NewRandomSymmetricKey(32)
	require.NoError(t, err)
	signingKey := base64.StdEncoding.EncodeToString(workerKey)

	secretKey := make([]byte, 32)
	_, err = base64.StdEncoding.Decode(secretKey, []byte(signingKey))
	require.NoError(t, err)

	signer, err := jws.NewHMacSigner(secretKey)
	require.NoError(t, err)

	// Setup test worker data for push and create run and tmp directories
	ctxPush := context.Background()
	wkPush := &CurrentWorker{
		basedir: afero.NewBasePathFs(fs, basedir),
	}
	pushJobInfo := sdk.WorkflowNodeJobRunData{}
	pushJobInfo.NodeJobRun.Job.Job.Action.Name = sdk.RandomString(10)

	wdPushFile, wdPushAbs, err := wkPush.setupWorkingDirectory(ctxPush, pushJobInfo.NodeJobRun.Job.Job.Action.Name)
	require.NoError(t, err)
	ctxPush = workerruntime.SetWorkingDirectory(ctxPush, wdPushFile)
	t.Logf("Setup push workspace at %s", wdPushFile.Name())

	tdPushFile, _, err := wkPush.setupTmpDirectory(ctxPush, pushJobInfo.NodeJobRun.Job.Job.Action.Name)
	require.NoError(t, err)
	ctxPush = workerruntime.SetTmpDirectory(ctxPush, tdPushFile)
	t.Logf("Setup push tmp directory at %s", tdPushFile.Name())

	wkPush.signer = signer
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

	wkPull.signer = signer
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
	require.NoError(t, os.WriteFile(absoluteFilePath, []byte("absolute"), os.FileMode(0755)))

	// Prepare mock client for cds workers
	ctrl := gomock.NewController(t)
	m := mock_cdsclient.NewMockWorkerInterface(ctrl)
	wkPush.client = m
	wkPull.client = m
	wkPush.cfg = &workerruntime.WorkerConfig{CDNEndpoint: "https://cdn.local"}
	wkPull.cfg = &workerruntime.WorkerConfig{CDNEndpoint: "https://cdn.local"}

	var bodyBytes []byte
	m.EXPECT().CDNItemUpload(gomock.Any(), "https://cdn.local", gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, cdnAddr string, signature string, fs afero.Fs, path string) (time.Duration, error) {
			t0 := time.Now()
			require.True(t, strings.Contains(path, "tar-"))
			f, err := fs.Open(path)
			require.NoError(t, err)
			bodyBytes, err = io.ReadAll(f)
			require.NoError(t, err)
			require.NotEqual(t, 0, len(bodyBytes))
			return time.Since(t0), nil
		},
	)

	md5Hash := md5.New()
	md5S := hex.EncodeToString(md5Hash.Sum(nil))
	apiRefHash := sdk.RandomString(10)
	m.EXPECT().QueueWorkerCacheLink(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, jobID int64, tag string) (sdk.CDNItemLinks, error) {
			return sdk.CDNItemLinks{
				CDNHttpURL: "https://cdn.local",
				Items: []sdk.CDNItem{
					{
						ID:         "foo",
						MD5:        md5S,
						APIRefHash: apiRefHash,
						Type:       sdk.CDNTypeItemWorkerCache,
					},
				},
			}, nil
		},
	)

	// ctx context.Context, cdnAddr string, hash string, itemType sdk.CDNItemType, md5Sum string, writer io.WriteSeeker) error
	m.EXPECT().CDNItemDownload(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, cdnAddr string, hash string, itemType sdk.CDNItemType, md5Sum string, writer io.WriteSeeker) error {
			require.Equal(t, "https://cdn.local", cdnAddr)
			require.Equal(t, sdk.CDNTypeItemWorkerCache, itemType)
			require.Equal(t, md5S, md5Sum)
			require.Equal(t, apiRefHash, hash)

			_, err := writer.Seek(0, io.SeekStart)
			require.NoError(t, err)

			_, err = io.Copy(writer, bytes.NewBuffer(bodyBytes))
			require.NoError(t, err)
			return nil
		},
	)

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

	btsRelative, err := os.ReadFile(expectedRelativePath)
	require.NoError(t, err)
	assert.Equal(t, "relative", string(btsRelative))

	btsAbsolute, err := os.ReadFile(expectedAbsolutePath)
	require.NoError(t, err)
	assert.Equal(t, "absolute", string(btsAbsolute))
}
