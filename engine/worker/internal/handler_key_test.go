package internal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_keyInstall(t *testing.T) {
	// Init a real worker, not the mocking one
	var w = new(CurrentWorker)
	fs := afero.NewOsFs()
	basedir := "test-" + test.GetTestName(t) + "-" + sdk.RandomString(10) + "-" + fmt.Sprintf("%d", time.Now().Unix())
	require.NoError(t, fs.MkdirAll(basedir, os.FileMode(0755)))

	if err := w.Init("test-worker", "test-hatchery", "http://lolcat.host", "xxx-my-token", "", true, afero.NewBasePathFs(fs, basedir)); err != nil {
		t.Fatalf("worker init failed: %v", err)
	}
	require.NoError(t, w.BaseDir().Mkdir("keys", os.FileMode(0700)))
	keyDir, err := w.BaseDir().Open("keys")
	require.NoError(t, err)
	w.currentJob.context = workerruntime.SetKeysDirectory(context.TODO(), keyDir)

	require.NoError(t, w.BaseDir().Mkdir("workingdir", os.FileMode(0700)))
	workingdir, err := w.BaseDir().Open("workingdir")
	require.NoError(t, err)
	w.currentJob.context = workerruntime.SetWorkingDirectory(context.TODO(), workingdir)
	// End worker init

	path, _ := w.BaseDir().(*afero.BasePathFs).RealPath(workingdir.Name())
	absPath, _ := filepath.Abs(path)

	resp, err := keyInstall(w, filepath.Join(absPath, "myKey"), &sdk.Variable{
		Name:  "cds.key.proj-ssh-key.priv",
		Value: string(test.TestKey),
		Type:  string(sdk.KeyTypeSSH),
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, sdk.KeyTypeSSH, resp.Type)
	assert.NotEmpty(t, resp.Content)

	expectedAbsolutePath, _ := filepath.Abs(filepath.Join(path, "myKey"))
	assert.Equal(t, expectedAbsolutePath, resp.PKey)
}
