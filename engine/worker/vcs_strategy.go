package main

import (
	"fmt"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/vcs"
	"github.com/ovh/cds/sdk/vcs/git"
)

func extractVCSInformations(params []sdk.Parameter) (string, *git.AuthOpts, error) {
	var gitUrl string
	var auth *git.AuthOpts

	// Get connection type
	connetionType := sdk.ParameterFind(params, "git.connection.type")
	if connetionType == nil || (connetionType.Value != "ssh" && connetionType.Value != "https") {
		return gitUrl, nil, fmt.Errorf("git connection type is not set. nothing to perform")
	}

	switch connetionType.Value {
	case "ssh":
		keyName := sdk.ParameterFind(params, "git.ssh.key")
		if keyName == nil || keyName.Value == "" {
			return gitUrl, nil, fmt.Errorf("git ssh key is not set. nothing to perform")
		}

		privateKey := sdk.ParameterFind(params, "cds.key."+keyName.Value+".priv")
		if privateKey == nil || privateKey.Value == "" {
			return gitUrl, nil, fmt.Errorf("ssh key not found. Nothing to perform")
		}
		if err := vcs.SetupSSHKey(nil, keysDirectory, privateKey); err != nil {
			return gitUrl, nil, fmt.Errorf("unable to setup ssh key. %s", err)
		}
		key, errK := vcs.GetSSHKey(params, keysDirectory, privateKey)
		if errK != nil && errK != sdk.ErrKeyNotFound {
			return gitUrl, nil, fmt.Errorf("unable to setup ssh key. %s", errK)
		}
		if key != nil {
			if auth == nil {
				auth = new(git.AuthOpts)
			}
			auth.PrivateKey = *key
		}

		url := sdk.ParameterFind(params, "git.url")
		if url == nil || url.Value == "" {
			return gitUrl, nil, fmt.Errorf("SSH Url (git.url) not found. Nothing to perform")
		}
		gitUrl = url.Value
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
			return gitUrl, nil, fmt.Errorf("SSH Url (git.http_url) not found. Nothing to perform")
		}
		gitUrl = url.Value
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
	return gitUrl, auth, nil
}
