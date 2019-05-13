package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/sdk"
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
			shouldHaveError: true,
		},
	}

	for _, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			script, err := prepareScriptContent(&tst.parameters)
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
