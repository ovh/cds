package action

import (
	"testing"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"

	"github.com/stretchr/testify/assert"
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

func TestRunScriptAction(t *testing.T) {
	wk, ctx := setupTest(t)
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
