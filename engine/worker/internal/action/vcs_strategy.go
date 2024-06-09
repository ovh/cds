package action

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/vcs"
	"github.com/ovh/cds/sdk/vcs/git"
)

func vcsStrategy(ctx context.Context, wk workerruntime.Runtime, params []sdk.Parameter, secrets []sdk.Variable) (string, *git.AuthOpts, error) {
	var gitURL string
	auth := new(git.AuthOpts)
	// Get connection type
	connectionType := sdk.ParameterFind(params, "git.connection.type")
	if connectionType == nil || (connectionType.Value != "ssh" && connectionType.Value != "https") {
		return gitURL, nil, fmt.Errorf("git connection type is not set. nothing to perform")
	}

	switch connectionType.Value {
	case "ssh":
		keyName := sdk.ParameterFind(params, "git.ssh.key")
		if keyName == nil || keyName.Value == "" {
			return gitURL, nil, fmt.Errorf("git ssh key is not set. nothing to perform")
		}
		privateKey := sdk.ParameterFind(params, "cds.key."+keyName.Value+".priv")
		if privateKey == nil || privateKey.Value == "" {
			return gitURL, nil, fmt.Errorf("ssh key not found. Nothing to perform")
		}
		privateKeyVar := sdk.Variable{
			Name:  "cds.key." + keyName.Value + ".priv",
			Type:  string(sdk.KeyTypeSSH),
			Value: privateKey.Value,
		}

		installedKey, err := wk.InstallKey(privateKeyVar)
		if err != nil {
			return gitURL, nil, err
		}

		aferoKeyDir, err := workerruntime.KeysDirectory(ctx)
		if err != nil {
			return "", nil, sdk.WithStack(err)
		}

		if err := vcs.SetupSSHKey(wk.BaseDir(), aferoKeyDir.Name(), privateKeyVar); err != nil {
			return gitURL, nil, sdk.WithStack(fmt.Errorf("unable to setup ssh key. %s", err))
		}

		keysDirectory := filepath.Dir(installedKey.PKey)
		key, errK := vcs.GetSSHKey(secrets, keysDirectory, &privateKeyVar)
		if errK != nil && !sdk.ErrorIs(errK, sdk.ErrKeyNotFound) {
			return gitURL, nil, sdk.WithStack(fmt.Errorf("unable to setup ssh key. %s", errK))
		}

		auth.PrivateKey = *key

		url := sdk.ParameterFind(params, "git.url")
		if url == nil || url.Value == "" {
			return gitURL, nil, sdk.WithStack(fmt.Errorf("SSH Url (git.url) not found. Nothing to perform"))
		}
		gitURL = url.Value

	case "https":
		user := sdk.ParameterFind(params, "git.http.user")
		password := sdk.ParameterFind(params, "git.http.password")

		if user != nil || password != nil {
			auth = new(git.AuthOpts)
			if user != nil {
				auth.Username = user.Value
			}
			if password != nil {
				auth.Password = password.Value
			}
		}

		url := sdk.ParameterFind(params, "git.http_url")
		if url == nil || url.Value == "" {
			return gitURL, nil, sdk.WithStack(fmt.Errorf("SSH Url (git.http_url) not found. Nothing to perform"))
		}
		gitURL = url.Value
	}

	pgpKeyName := sdk.ParameterFind(params, "git.pgp.key")
	if pgpKeyName != nil && pgpKeyName.Value != "" {
		auth.SignKey = vcs.PGPKey{
			Name: pgpKeyName.Value,
		}
		pgpPrivate := sdk.ParameterFind(params, "cds.key."+pgpKeyName.Value+".priv")
		if pgpPrivate != nil && pgpPrivate.Value != "" {
			auth.SignKey.Private = pgpPrivate.Value
		}
		pgpPublic := sdk.ParameterFind(params, "cds.key."+pgpKeyName.Value+".pub")
		if pgpPublic != nil && pgpPublic.Value != "" {
			auth.SignKey.Public = pgpPublic.Value
		}
		pgpID := sdk.ParameterFind(params, "cds.key."+pgpKeyName.Value+".id")
		if pgpID != nil && pgpID.Value != "" {
			auth.SignKey.ID = pgpID.Value
		}

	}
	return gitURL, auth, nil
}
