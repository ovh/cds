package action

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"testing"
	"time"

	"github.com/rockbears/log"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/vcs"
)

type TestWorker struct {
	t                *testing.T
	workspace        afero.Fs
	workingDirectory *afero.BasePathFile
	keyDirectory     *afero.BasePathFile
	client           cdsclient.WorkerInterface
	Params           []sdk.Parameter
	logBuffer        bytes.Buffer
}

func (w *TestWorker) ClientV2() cdsclient.V2WorkerInterface {
	//TODO implement me
	panic("implement me")
}

func (w *TestWorker) PluginGet(pluginName string) (*sdk.GRPCPlugin, error) {
	//TODO implement me
	panic("implement me")
}

func (w *TestWorker) PluginGetBinary(name, os, arch string, wr io.Writer) error {
	//TODO implement me
	panic("implement me")
}

func (w *TestWorker) GetJobIdentifiers() (int64, int64, int64) {
	return 0, 0, 0
}

func (w *TestWorker) WorkerCacheSignature(tag string) (string, error) {
	return "mysig", nil
}

func (w *TestWorker) RunResultSignature(artifactName string, perm uint32, t sdk.WorkflowRunResultType) (string, error) {
	return "mysignature", nil
}

func (w *TestWorker) FeatureEnabled(featureName sdk.FeatureName) bool {
	return false
}

func (w *TestWorker) CDNHttpURL() string {
	return "http://cds-cdn.local"
}

func (w *TestWorker) WorkingDirectory() *afero.BasePathFile {
	return w.workingDirectory
}

func (w *TestWorker) KeyDirectory() *afero.BasePathFile {
	return w.keyDirectory
}

func (w *TestWorker) Blur(i interface{}) error {
	w.t.Log("Blur")
	return nil
}

func (w *TestWorker) GetIntegrationPlugin(t string) *sdk.GRPCPlugin {
	return nil
}
func (w *TestWorker) GetActionPlugin(t string) *sdk.GRPCPlugin {
	return nil
}
func (w *TestWorker) SetActionPlugin(p *sdk.GRPCPlugin) {

}
func (w *TestWorker) Parameters() []sdk.Parameter {
	return w.Params
}

func (w *TestWorker) Client() cdsclient.WorkerInterface {
	return w.client
}

func (_ *TestWorker) Environ() []string {
	return os.Environ()
}

func (_ *TestWorker) HTTPPort() int32 {
	return 0
}

func (_ *TestWorker) Name() string {
	return "test"
}

func (wk TestWorker) BaseDir() afero.Fs {
	return wk.workspace
}

func (_ *TestWorker) Register(ctx context.Context) error {
	return nil
}
func (_ *TestWorker) Take(ctx context.Context, job sdk.WorkflowNodeJobRun) error {
	return nil
}
func (_ *TestWorker) ProcessJob(job sdk.WorkflowNodeJobRunData) sdk.Result {
	return sdk.Result{}
}
func (w *TestWorker) SendLog(ctx context.Context, level workerruntime.Level, format string) {
	s := "SendLog> [" + string(level) + "] " + format
	w.t.Log(s)
	_, err := w.logBuffer.WriteString(s)
	require.NoError(w.t, err)
}
func (_ *TestWorker) Unregister(ctx context.Context) error {
	return nil
}

func (w *TestWorker) InstallKey(key sdk.Variable) (*workerruntime.KeyResponse, error) {
	installedKeyPath := path.Join(w.keyDirectory.Name(), key.Name)
	err := vcs.CleanAllSSHKeys(w.BaseDir(), w.keyDirectory.Name())
	require.NoError(w.t, err)

	err = vcs.SetupSSHKey(w.BaseDir(), w.keyDirectory.Name(), key)
	require.NoError(w.t, err)

	if x, ok := w.BaseDir().(*afero.BasePathFs); ok {
		installedKeyPath, _ = x.RealPath(installedKeyPath)
	}

	return &workerruntime.KeyResponse{
		Content: []byte(key.Value),
		Type:    sdk.KeyType(key.Type),
		PKey:    installedKeyPath,
	}, nil
}

func (w *TestWorker) InstallKeyTo(key sdk.Variable, destinationPath string) (*workerruntime.KeyResponse, error) {
	var installedKeyPath string

	w.t.Logf("InstallKey> destination : %s", destinationPath)
	err := vcs.WriteKey(afero.NewOsFs(), destinationPath, key.Value)
	require.NoError(w.t, err)
	if x, ok := w.BaseDir().(*afero.BasePathFs); ok {
		installedKeyPath, _ = x.RealPath(destinationPath)
	}

	return &workerruntime.KeyResponse{
		Content: []byte(key.Value),
		Type:    sdk.KeyType(key.Type),
		PKey:    installedKeyPath,
	}, nil
}

var _ workerruntime.Runtime = new(TestWorker)

func SetupTest(t *testing.T) (*TestWorker, context.Context) {
	fs := afero.NewOsFs()
	basedir := "test-" + test.GetTestName(t) + "-" + sdk.RandomString(10) + "-" + fmt.Sprintf("%d", time.Now().Unix())
	log.Debug(context.TODO(), "creating basedir %s", basedir)
	require.NoError(t, fs.MkdirAll(basedir, os.FileMode(0755)))

	wk := TestWorker{
		t:         t,
		workspace: afero.NewBasePathFs(fs, basedir),
	}

	err := wk.BaseDir().Mkdir("working_directory", os.FileMode(0755))
	require.NoError(t, err)
	fi, err := wk.BaseDir().Open("working_directory")
	require.NoError(t, err)

	wk.workingDirectory = fi.(*afero.BasePathFile)

	err = wk.BaseDir().Mkdir("key_directory", os.FileMode(0755))
	require.NoError(t, err)
	fi, err = wk.BaseDir().Open("key_directory")
	require.NoError(t, err)

	wk.keyDirectory = fi.(*afero.BasePathFile)

	wk.client = cdsclient.NewWorker("http://cds-api.local", "test-client", cdsclient.NewHTTPClient(time.Second*360, false))

	ctx := workerruntime.SetWorkingDirectory(context.TODO(), wk.workingDirectory)
	ctx = workerruntime.SetKeysDirectory(ctx, wk.keyDirectory)
	ctx = workerruntime.SetJobID(ctx, 666)

	return &wk, ctx
}
