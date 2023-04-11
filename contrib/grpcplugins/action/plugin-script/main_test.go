package main

import (
	"context"
	"fmt"
	"github.com/ovh/cds/engine/test"
	"github.com/spf13/afero"
	"os"
	"testing"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			scriptShell: "/bin/sh",
			scriptOpts:  []string{"-e"},
		},
	}

	for _, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			script, err := prepareScriptContent(tst.parameters[0].Value, "/work/dir")
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

func Test_writeScriptContent_windows(t *testing.T) {
	sdk.GOOS = "windows"
	defer func() {
		sdk.GOOS = ""
	}()

	fs := afero.NewOsFs()
	basedir := "test-" + test.GetTestName(t) + "-" + sdk.RandomString(10) + "-" + fmt.Sprintf("%d", time.Now().Unix())
	t.Logf("creating basedir %s", basedir)
	require.NoError(t, fs.MkdirAll(basedir, os.FileMode(0755)))

	baseDir := afero.NewBasePathFs(fs, basedir)

	err := baseDir.Mkdir("working_directory", os.FileMode(0755))
	require.NoError(t, err)
	_, err = baseDir.Open("working_directory")
	require.NoError(t, err)

	script, err := prepareScriptContent("sleep 1\necho this is a test from %HOME%\nsleep 1", "working_directory")
	require.NoError(t, err)
	require.NotNil(t, script)

	deferFunc, err := writeScriptContent(context.Background(), script, baseDir)
	if deferFunc != nil {
		defer deferFunc()
	}
	require.NoError(t, err)

	require.Equal(t, script.shell, "PowerShell")
	t.Logf("script: %+v", script)
}
