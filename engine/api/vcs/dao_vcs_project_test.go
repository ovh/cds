package vcs

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
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
		Type:        "github",
		Auth:        map[string]string{"username": "the-username", "token": "the-token"},
		Description: "the-username",
		ProjectID:   proj1.ID,
	}

	err := Insert(context.TODO(), db, vcsProject)
	require.NoError(t, err)
	require.NotEmpty(t, vcsProject.ID)

	vcsProject.ProjectID = proj2.ID
	vcsProject.Description = "the-2-username"
	err = Insert(context.TODO(), db, vcsProject)
	require.NoError(t, err)

	all, err := LoadAllVCSByProject(context.Background(), db, proj1.Key)
	require.NoError(t, err)
	require.Equal(t, 1, len(all))
	require.Equal(t, "the-username", all[0].Description)
	require.Equal(t, "", all[0].Auth["username"]) // not decrypted

	all[0].Description = "the-username-updated"
	all[0].Auth = map[string]string{"username": "the-username-updated", "token": "the-token-updated"}

	err = Update(context.TODO(), db, &all[0])
	require.NoError(t, err)

	vcs, err := LoadVCSByProject(context.Background(), db, proj1.ID, "the-name", gorpmapping.GetOptions.WithDecryption)
	require.NoError(t, err)
	require.Equal(t, "the-username-updated", vcs.Description)
	require.Equal(t, "the-username-updated", vcs.Auth["username"]) // decrypted
	require.Equal(t, "the-token-updated", vcs.Auth["token"])       // decrypted

	err = Delete(db, proj1.ID, "the-name")
	require.NoError(t, err)

	all, err = LoadAllVCSByProject(context.Background(), db, proj1.Key)
	require.NoError(t, err)
	require.Equal(t, 0, len(all))

	//err = Delete(db, proj2.ID, "the-name")
	//require.NoError(t, err)
}
