package action

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/vcs"
)

type TestWorker struct {
	t                *testing.T
	workspace        afero.Fs
	workingDirectory *afero.BasePathFile
	keyDirectory     *afero.BasePathFile
}

func (w TestWorker) Blur(i interface{}) error {
	w.t.Log("Blur")
	return nil
}

func (_ TestWorker) Client() cdsclient.WorkerInterface {
	return nil
}

func (_ TestWorker) Environ() []string {
	return os.Environ()
}

func (_ TestWorker) HTTPPort() int32 {
	return 0
}

func (_ TestWorker) Name() string {
	return "test"
}

func (wk TestWorker) Workspace() afero.Fs {
	return wk.workspace
}

func (_ TestWorker) Register(ctx context.Context) error {
	return nil
}
func (_ TestWorker) Take(ctx context.Context, job sdk.WorkflowNodeJobRun) error {
	return nil
}
func (_ TestWorker) ProcessJob(job sdk.WorkflowNodeJobRunData) (sdk.Result, error) {
	return sdk.Result{}, nil
}
func (w TestWorker) SendLog(ctx context.Context, level workerruntime.Level, format string) {
	w.t.Log("SendLog> [" + string(level) + "] " + format)

}
func (_ TestWorker) Unregister() error {
	return nil
}

func (w TestWorker) InstallKey(key sdk.Variable, destinationPath string) (*workerruntime.KeyResponse, error) {
	installedKeyPath := path.Join(w.keyDirectory.Name(), key.Name)
	err := vcs.CleanAllSSHKeys(w.Workspace(), w.keyDirectory.Name())
	require.NoError(w.t, err)

	err = vcs.SetupSSHKey(w.Workspace(), w.keyDirectory.Name(), key)
	require.NoError(w.t, err)

	if x, ok := w.Workspace().(*afero.BasePathFs); ok {
		installedKeyPath, _ = x.RealPath(installedKeyPath)
	}

	return &workerruntime.KeyResponse{
		Content: []byte(key.Value),
		Type:    key.Type,
		PKey:    installedKeyPath,
	}, nil
}

var _ workerruntime.Runtime = new(TestWorker)

func setupTest(t *testing.T) (TestWorker, context.Context) {
	fs := afero.NewOsFs()
	basedir := "test-" + test.GetTestName(t) + "-" + sdk.RandomString(10) + "-" + fmt.Sprintf("%d", time.Now().Unix())
	log.Debug("creating basedir %s", basedir)
	require.NoError(t, fs.MkdirAll(basedir, os.FileMode(0755)))

	wk := TestWorker{
		t:         t,
		workspace: afero.NewBasePathFs(fs, basedir),
	}

	err := wk.Workspace().Mkdir("working_directory", os.FileMode(0755))
	require.NoError(t, err)
	fi, err := wk.Workspace().Open("working_directory")
	require.NoError(t, err)

	wk.workingDirectory = fi.(*afero.BasePathFile)

	err = wk.Workspace().Mkdir("key_directory", os.FileMode(0755))
	require.NoError(t, err)
	fi, err = wk.Workspace().Open("key_directory")
	require.NoError(t, err)

	wk.keyDirectory = fi.(*afero.BasePathFile)

	ctx := workerruntime.SetWorkingDirectory(context.TODO(), wk.workingDirectory)
	ctx = workerruntime.SetKeysDirectory(ctx, wk.keyDirectory)
	return wk, ctx
}
