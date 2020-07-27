package action_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/engine/worker/internal"
	"github.com/ovh/cds/engine/worker/internal/action"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func TestRunInstallKeyAction_Relative(t *testing.T) {
	// Init a real worker, not the mocking one
	var w = new(internal.CurrentWorker)
	fs := afero.NewOsFs()
	basedir := "test-" + test.GetTestName(t) + "-" + sdk.RandomString(10) + "-" + fmt.Sprintf("%d", time.Now().Unix())
	require.NoError(t, fs.MkdirAll(basedir, os.FileMode(0755)))

	if err := w.Init("test-worker", "test-hatchery", "http://lolcat.host", "xxx-my-token", "", true, afero.NewBasePathFs(fs, basedir)); err != nil {
		t.Fatalf("worker init failed: %v", err)
	}
	require.NoError(t, w.BaseDir().Mkdir("keys", os.FileMode(0700)))
	keyDir, err := w.BaseDir().Open("keys")
	require.NoError(t, err)
	w.SetContext(context.Background())
	ctx := workerruntime.SetKeysDirectory(w.GetContext(), keyDir)

	require.NoError(t, w.BaseDir().Mkdir("workingdir", os.FileMode(0700)))
	workingdir, err := w.BaseDir().Open("workingdir")
	require.NoError(t, err)
	w.SetContext(workerruntime.SetWorkingDirectory(ctx, workingdir))

	w.SetGelfLogger(nil, logrus.New())
	// End worker init

	keyInstallAction := sdk.Action{
		Parameters: []sdk.Parameter{
			{
				Name:  "key",
				Type:  sdk.KeyParameter,
				Value: "proj-mykey",
			}, {
				Name:  "file",
				Type:  sdk.KeyParameter,
				Value: "my-key",
			},
		},
	}
	secrets := []sdk.Variable{
		{
			ID:    1,
			Name:  "cds.key.proj-mykey.priv",
			Value: "test",
			Type:  string(sdk.KeyTypeSSH),
		},
	}

	res, err := action.RunInstallKey(w.GetContext(), w, keyInstallAction, secrets)
	assert.NoError(t, err)
	assert.Equal(t, sdk.StatusSuccess, res.Status)

	wkd, err := workerruntime.WorkingDirectory(w.GetContext())
	require.NoError(t, err)

	absBasedir, _ := filepath.Abs(basedir)
	assert.FileExists(t, filepath.Join(absBasedir, wkd.Name(), "my-key"))

}

func TestRunInstallKeyAction_Absolute(t *testing.T) {
	// Init a real worker, not the mocking one
	var w = new(internal.CurrentWorker)
	fs := afero.NewOsFs()
	basedir := "test-" + test.GetTestName(t) + "-" + sdk.RandomString(10) + "-" + fmt.Sprintf("%d", time.Now().Unix())
	require.NoError(t, fs.MkdirAll(basedir, os.FileMode(0755)))

	if err := w.Init("test-worker", "test-hatchery", "http://lolcat.host", "xxx-my-token", "", true, afero.NewBasePathFs(fs, basedir)); err != nil {
		t.Fatalf("worker init failed: %v", err)
	}
	require.NoError(t, w.BaseDir().Mkdir("keys", os.FileMode(0700)))
	keyDir, err := w.BaseDir().Open("keys")
	require.NoError(t, err)
	w.SetContext(context.Background())
	ctx := workerruntime.SetKeysDirectory(w.GetContext(), keyDir)

	require.NoError(t, w.BaseDir().Mkdir("workingdir", os.FileMode(0700)))
	workingdir, err := w.BaseDir().Open("workingdir")
	require.NoError(t, err)
	w.SetContext(workerruntime.SetWorkingDirectory(ctx, workingdir))

	w.SetGelfLogger(nil, logrus.New())
	// End worker init

	keyInstallAction := sdk.Action{
		Parameters: []sdk.Parameter{
			{
				Name:  "key",
				Type:  sdk.KeyParameter,
				Value: "proj-mykey",
			}, {
				Name:  "file",
				Type:  sdk.KeyParameter,
				Value: "/tmp/my-key",
			},
		},
	}
	secrets := []sdk.Variable{
		sdk.Variable{
			ID:    1,
			Name:  "cds.key.proj-mykey.priv",
			Value: "test",
			Type:  string(sdk.KeyTypeSSH),
		},
	}
	res, err := action.RunInstallKey(w.GetContext(), w, keyInstallAction, secrets)
	assert.NoError(t, err)
	assert.Equal(t, sdk.StatusSuccess, res.Status)
	assert.FileExists(t, "/tmp/my-key")
	os.RemoveAll("/tmp/my-key")
}
