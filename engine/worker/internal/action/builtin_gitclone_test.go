package action

import (
	"path/filepath"
	"testing"

	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func TestRunGitCloneInSSHWithoutVCSStrategyShouldRaiseError(t *testing.T) {
	wk, ctx := SetupTest(t)

	res, err := RunGitClone(ctx, wk,
		sdk.Action{}, nil)
	assert.Error(t, err)
	assert.NotEqual(t, sdk.StatusSuccess, res.Status)
}

func TestRunGitCloneInSSHWithoutKeyShouldRaiseError(t *testing.T) {
	wk, ctx := SetupTest(t)

	res, err := RunGitClone(ctx, wk,
		sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "url",
					Value: "git@github.com:ovh/cds.git",
				},
				{
					Name:  "privateKey",
					Value: "proj-ssh-key",
				},
			},
		}, nil)
	assert.Error(t, err)
	assert.NotEqual(t, sdk.StatusSuccess, res.Status)
}

func TestRunGitCloneInSSHWithPrivateKey(t *testing.T) {
	wk, ctx := SetupTest(t)

	res, err := RunGitClone(ctx, wk,
		sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "url",
					Value: "git@github.com:fsamin/dummy-empty-repo.git",
				}, {
					Name:  "privateKey",
					Value: "proj-ssh-key",
				},
			},
		},
		[]sdk.Variable{
			{
				Name:  "cds.key.proj-ssh-key.priv",
				Value: string(test.TestKey),
			},
		})
	assert.NoError(t, err)
	assert.Equal(t, sdk.StatusSuccess, res.Status)

	assert.DirExists(t, filepath.Join(wk.workingDirectory.File.Name(), "dummy-empty-repo"))
	assert.DirExists(t, filepath.Join(wk.workingDirectory.File.Name(), "dummy-empty-repo", ".git"))
}

func TestRunGitCloneInSSHWithTheWrongPrivateKeyShouldFail(t *testing.T) {
	wk, ctx := SetupTest(t)

	res, err := RunGitClone(ctx, wk,
		sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "url",
					Value: "git@github.com:fsamin/dummy-empty-repo.git",
				}, {
					Name:  "privateKey",
					Value: "proj-ssh-key",
				},
			},
		},
		[]sdk.Variable{
			{
				Name:  "cds.key.proj-ssh-key.priv",
				Value: "this not a private key",
			},
		})
	assert.Error(t, err)
	assert.NotEqual(t, sdk.StatusSuccess, res.Status)
}

func TestRunGitCloneInSSHWithPrivateKeyWithTargetDirectory(t *testing.T) {
	wk, ctx := SetupTest(t)

	res, err := RunGitClone(ctx, wk,
		sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "url",
					Value: "git@github.com:fsamin/dummy-empty-repo.git",
				}, {
					Name:  "privateKey",
					Value: "proj-ssh-key",
				}, {
					Name:  "directory",
					Value: "there-there",
				},
			},
		},
		[]sdk.Variable{
			{
				Name:  "cds.key.proj-ssh-key.priv",
				Value: string(test.TestKey),
			},
		})
	assert.NoError(t, err)
	assert.Equal(t, sdk.StatusSuccess, res.Status)

	assert.DirExists(t, filepath.Join(wk.workingDirectory.File.Name(), "there-there"))
	assert.DirExists(t, filepath.Join(wk.workingDirectory.File.Name(), "there-there", ".git"))
	assert.Empty(t, res.NewVariables)
}

func TestRunGitCloneInSSHWithPrivateKeyAndExtractInfo(t *testing.T) {
	wk, ctx := SetupTest(t)
	wk.Params = append(wk.Params, []sdk.Parameter{
		{
			Name:  "git.url",
			Value: "git@github.com:fsamin/dummy-empty-repo.git",
		},
		{
			Name:  "cds.version",
			Value: "1",
		},
	}...)
	res, err := RunGitClone(ctx, wk,
		sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "url",
					Value: "git@github.com:fsamin/dummy-empty-repo.git",
				}, {
					Name:  "privateKey",
					Value: "proj-ssh-key",
				}, {
					Name:  "directory",
					Value: ".",
				},
			},
		},
		[]sdk.Variable{
			{
				Name:  "cds.key.proj-ssh-key.priv",
				Value: string(test.TestKey),
			},
		})
	assert.NoError(t, err)
	assert.Equal(t, sdk.StatusSuccess, res.Status)

	assert.DirExists(t, filepath.Join(wk.workingDirectory.File.Name(), ".git"))
	assert.NotEmpty(t, res.NewVariables)
	t.Logf("new variables: %+v", res.NewVariables)
}

func TestRunGitCloneInSSHWithApplicationVCSStrategy(t *testing.T) {
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
	res, err := RunGitClone(ctx, wk,
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

func TestRunGitCloneInHTTPSWithoutAuth(t *testing.T) {

	wk, ctx := SetupTest(t)

	res, err := RunGitClone(ctx, wk,
		sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "url",
					Value: "https://github.com/fsamin/dummy-empty-repo.git",
				},
				{
					Name:  "privateKey",
					Value: "",
				},
				{
					Name:  "user",
					Value: "",
				},
				{
					Name:  "password",
					Value: "",
				},
				{
					Name:  "branch",
					Value: "",
				},
				{
					Name:  "commit",
					Value: "",
				},
				{
					Name:  "directory",
					Value: "",
				},
				{
					Name:  "depth",
					Value: "",
				},
				{
					Name:  "submodules",
					Value: "",
				},
			},
		}, nil)
	assert.NoError(t, err)
	assert.Equal(t, sdk.StatusSuccess, res.Status)
	assert.DirExists(t, filepath.Join(wk.workingDirectory.File.Name(), "dummy-empty-repo"))
	assert.DirExists(t, filepath.Join(wk.workingDirectory.File.Name(), "dummy-empty-repo", ".git"))
}

func TestRunGitCloneWithSecret(t *testing.T) {
	wk, ctx := SetupTest(t)
	res, err := RunGitClone(ctx, wk,
		sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "url",
					Value: "git@github.com:fsamin/dummy-empty-repo.git",
				},
				{
					Name:  "privateKey",
					Value: "proj-ssh-key",
				},
				{
					Name:  "directory",
					Value: ".",
				},
			},
		}, []sdk.Variable{
			{
				Name:  "cds.key.proj-ssh-key.priv",
				Value: string(test.TestKey),
				Type:  string(sdk.KeyTypeSSH),
			},
		})
	assert.NoError(t, err)
	assert.Equal(t, sdk.StatusSuccess, res.Status)

	assert.DirExists(t, filepath.Join(wk.workingDirectory.File.Name(), ".git"))
}
