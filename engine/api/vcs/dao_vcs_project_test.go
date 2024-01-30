package vcs_test

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func TestCrud(t *testing.T) {
	db, cache := test.SetupPG(t)

	key1 := sdk.RandomString(10)
	key2 := sdk.RandomString(10)
	proj1 := assets.InsertTestProject(t, db, cache, key1, key1)
	proj2 := assets.InsertTestProject(t, db, cache, key2, key2)

	vcsProject := &sdk.VCSProject{
		Name:        "the-name",
		Type:        sdk.VCSTypeGithub,
		Auth:        sdk.VCSAuthProject{Username: "the-username", Token: "the-token"},
		Description: "the-username",
		ProjectID:   proj1.ID,
	}

	err := vcs.Insert(context.TODO(), db, vcsProject)
	require.NoError(t, err)
	require.NotEmpty(t, vcsProject.ID)

	vcsProject.ProjectID = proj2.ID
	vcsProject.Description = "the-2-username"
	err = vcs.Insert(context.TODO(), db, vcsProject)
	require.NoError(t, err)

	all, err := vcs.LoadAllVCSByProject(context.Background(), db, proj1.Key)
	require.NoError(t, err)
	require.Equal(t, 1, len(all))
	require.Equal(t, "the-username", all[0].Description)
	require.Equal(t, "", all[0].Auth.Username) // not decrypted

	all[0].Description = "the-username-updated"
	all[0].Auth = sdk.VCSAuthProject{Username: "the-username-updated", Token: "the-token-updated"}

	err = vcs.Update(context.TODO(), db, &all[0])
	require.NoError(t, err)

	vcsProject2, err := vcs.LoadVCSByProject(context.Background(), db, proj1.Key, "the-name", gorpmapping.GetOptions.WithDecryption)
	require.NoError(t, err)
	require.Equal(t, "the-username-updated", vcsProject2.Description)
	require.Equal(t, "the-username-updated", vcsProject2.Auth.Username) // decrypted
	require.Equal(t, "the-token-updated", vcsProject2.Auth.Token)       // decrypted

	err = vcs.Delete(db, proj1.ID, "the-name")
	require.NoError(t, err)

	all, err = vcs.LoadAllVCSByProject(context.Background(), db, proj1.Key)
	require.NoError(t, err)
	require.Equal(t, 0, len(all))

	err = vcs.Delete(db, proj2.ID, "the-name")
	require.NoError(t, err)
}
