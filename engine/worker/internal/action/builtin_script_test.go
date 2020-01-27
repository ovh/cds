package action

import (
	"testing"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			wk, _ := SetupTest(t)
			script, err := prepareScriptContent(tst.parameters, wk.BaseDir(), wk.workingDirectory)
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

func TestRunScriptAction(t *testing.T) {
	wk, ctx := SetupTest(t)
	res, err := RunScriptAction(ctx, wk,
		sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "script",
					Value: "sleep 1\necho this is a test from $HOME\nsleep 1",
				},
			},
		},
		[]sdk.Variable{
			{
				Name:  "password",
				Value: "password",
			},
		})
	assert.NoError(t, err)
	assert.Equal(t, sdk.StatusSuccess, res.Status)
}

func Test_writeScriptContent_windows(t *testing.T) {
	sdk.GOOS = "windows"
	defer func() {
		sdk.GOOS = ""
	}()

	wk, ctx := SetupTest(t)

	script, err := prepareScriptContent([]sdk.Parameter{
		{
			Name:  "script",
			Value: "sleep 1\necho this is a test from %HOME%\nsleep 1",
		},
	}, wk.BaseDir(), wk.workingDirectory)
	require.NoError(t, err)
	require.NotNil(t, script)

	deferFunc, err := writeScriptContent(ctx, script, wk.BaseDir(), wk.workingDirectory)
	if deferFunc != nil {
		defer deferFunc()
	}
	require.NoError(t, err)

	t.Logf("script: %+v", script)
}
