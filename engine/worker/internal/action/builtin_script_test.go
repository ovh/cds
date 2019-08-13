package action

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

func init() {
	log.Initialize(&log.Conf{Level: "debug"})
}

func Test_prepareScriptContent(t *testing.T) {
	type testcase struct {
		name            string
		parameters      []sdk.Parameter
		shouldHaveError bool
		scriptContent   string
		scriptShell     string
		scriptOpts      []string
	}

	var tests = []testcase{
		{
			parameters: []sdk.Parameter{
				{
					Name: "script",
					Value: `#!/bin/bash -ex
echo "lol"`,
				},
			},
			scriptShell:   "/bin/bash",
			scriptContent: "echo \"lol\"",
			scriptOpts:    []string{"-ex"},
		},
		{
			parameters: []sdk.Parameter{
				{
					Name: "script",
					Value: `#!/bin/bash
echo "lol"`,
				},
			},
			scriptShell:   "/bin/bash",
			scriptContent: "echo \"lol\"",
			scriptOpts:    []string{"-e"},
		},
		{
			parameters: []sdk.Parameter{
				{
					Name:  "script",
					Value: `echo "lol"`,
				},
			},
			scriptShell:   "/bin/sh",
			scriptContent: "echo \"lol\"",
			scriptOpts:    []string{"-e"},
		},
		{
			parameters: []sdk.Parameter{
				{
					Name: "script",
				},
			},
			shouldHaveError: true,
		},
	}

	for _, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			script, err := prepareScriptContent(tst.parameters)
			if tst.shouldHaveError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tst.scriptShell, script.shell)
				assert.Equal(t, tst.scriptContent, string(script.content))
				assert.EqualValues(t, tst.scriptOpts, script.opts)
			}
		})
	}

}

type TestWorker struct {
	t *testing.T
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

func (_ TestWorker) Workspace() afero.Fs {
	return afero.NewOsFs()
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
func (w TestWorker) SendLog(level workerruntime.Level, format string) {
	w.t.Log("SendLog> [" + string(level) + "] " + format)

}
func (_ TestWorker) Unregister() error {
	return nil
}

var _ workerruntime.Runtime = new(TestWorker)

func TestRunScriptAction(t *testing.T) {
	wk := TestWorker{t}
	wdFS := afero.NewOsFs()
	wdName := sdk.RandomString(10)
	require.NoError(t, wdFS.MkdirAll(wdName, os.FileMode(0755)))
	defer wdFS.RemoveAll(wdName) // nolint

	wdFileInfo, err := wdFS.Stat(wdName)
	require.NoError(t, err)

	ctx := workerruntime.SetWorkingDirectory(context.TODO(), wdFileInfo)
	res, err := RunScriptAction(ctx, wk,
		sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "script",
					Value: "sleep 1\necho this is a test from $HOME\nsleep 1",
				},
			},
		},
		nil,
		[]sdk.Variable{
			{
				Name:  "password",
				Value: "password",
			},
		})
	assert.NoError(t, err)
	assert.Equal(t, sdk.StatusSuccess, res.Status)
}
