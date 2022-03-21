package vcs

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func TestCrud(t *testing.T) {
	db, cache := test.SetupPG(t)

	_, err := db.Exec("DELETE FROM vcs_server")
	require.NoError(t, err)

	key1 := sdk.RandomString(10)
	key2 := sdk.RandomString(10)
	proj1 := assets.InsertTestProject(t, db, cache, key1, key1)
	proj2 := assets.InsertTestProject(t, db, cache, key2, key2)

	vcsProject := &sdk.VCSProject{
		Name:      "foo",
		Type:      "github",
		Value:     []byte("my-secret"),
		Username:  "the-username",
		ProjectID: proj1.ID,
	}

	err = Insert(context.TODO(), db, vcsProject)
	require.NoError(t, err)
	require.True(t, vcsProject.ID > 0)

	vcsProject.ProjectID = proj2.ID
	vcsProject.Username = "the-2-username"
	err = Insert(context.TODO(), db, vcsProject)
	require.NoError(t, err)

	all, err := LoadAllVCSByProject(context.Background(), db, proj1.ID)
	require.NoError(t, err)
	require.Equal(t, 1, len(all))
	require.Equal(t, "the-username", all[0].Username)
	require.Equal(t, "", string(all[0].Value)) // not decrypted

	all[0].Username = "the-username-updated"
	all[0].Value = []byte("my-secret-updated")

	err = Update(context.TODO(), db, &all[0])
	require.NoError(t, err)

	vcs, err := LoadVCSByProjectWithDecryption(context.Background(), db, proj1.ID, "foo")
	require.NoError(t, err)
	require.Equal(t, "the-username-updated", vcs.Username)
	require.Equal(t, "my-secret-updated", string(vcs.Value)) // decrypted

	err = Delete(db, proj1.ID, "foo")
	require.NoError(t, err)

	all, err = LoadAllVCSByProject(context.Background(), db, proj1.ID)
	require.NoError(t, err)
	require.Equal(t, 0, len(all))

	err = Delete(db, proj2.ID, "foo")
	require.NoError(t, err)
}
