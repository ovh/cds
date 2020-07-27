package action

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
)

func TestRunCheckoutApplication(t *testing.T) {
	wk, ctx := SetupTest(t)
	wk.Params = append(wk.Params, []sdk.Parameter{
		{
			Name:  "git.connection.type",
			Value: "ssh",
		},
		{
			Name:  "git.url",
			Value: "git@github.com:fsamin/dummy-empty-repo.git",
		},
		{
			Name:  "git.ssh.key",
			Value: "proj-ssh-key",
		},
		{
			Name:  "cds.key.proj-ssh-key.priv",
			Value: string(test.TestKey),
		},
		{
			Name:  "cds.version",
			Value: "1",
		},
	}...)
	res, err := RunCheckoutApplication(ctx, wk,
		sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "directory",
					Value: ".",
				},
			},
		}, nil)
	assert.NoError(t, err)
	assert.Equal(t, sdk.StatusSuccess, res.Status)

	assert.DirExists(t, filepath.Join(wk.workingDirectory.File.Name(), ".git"))
	assert.NotEmpty(t, res.NewVariables)
	t.Logf("new variables: %+v", res.NewVariables)
}
