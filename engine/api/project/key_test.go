package project_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func TestEncryptWithBuiltinKey(t *testing.T) {
	db, cache := test.SetupPG(t)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	content := "This is my content"
	encryptedContent, err := project.EncryptWithBuiltinKey(context.TODO(), db, proj.ID, "test", content)
	test.NoError(t, err)
	t.Logf("%s => %s", content, encryptedContent)

	decryptedContent, err := project.DecryptWithBuiltinKey(context.TODO(), db, proj.ID, encryptedContent)
	test.NoError(t, err)
	t.Logf("%s => %s", encryptedContent, decryptedContent)

	assert.Equal(t, content, decryptedContent)

	encryptedContent2, err := project.EncryptWithBuiltinKey(context.TODO(), db, proj.ID, "test", content)
	test.NoError(t, err)
	t.Logf("%s => %s", content, encryptedContent2)
	assert.Equal(t, encryptedContent, encryptedContent2)
}

func Test_DAOKeysAllProjects(t *testing.T) {
	db, cache := test.SetupPG(t)

	key1 := sdk.RandomString(10)
	proj1 := assets.InsertTestProject(t, db, cache, key1, key1)

	key2 := sdk.RandomString(10)
	proj2 := assets.InsertTestProject(t, db, cache, key2, key2)

	ssh1, err := keys.GenerateSSHKey("ssh1")
	require.NoError(t, err)
	ssh2, err := keys.GenerateSSHKey("ssh2")
	require.NoError(t, err)
	k1 := &sdk.ProjectKey{Type: sdk.KeyTypeSSH, Name: "ssh1", Public: ssh1.Public, Private: ssh1.Private, ProjectID: proj1.ID}
	k2 := &sdk.ProjectKey{Type: sdk.KeyTypeSSH, Name: "ssh2", Public: ssh2.Public, Private: ssh2.Private, ProjectID: proj2.ID}

	require.NoError(t, project.InsertKey(db, k1))
	require.NoError(t, project.InsertKey(db, k2))

	keys, err := project.LoadAllKeysForProjectsWithDecryption(context.TODO(), db, []int64{proj1.ID, proj2.ID})
	require.NoError(t, err)

	require.Len(t, keys, 2)
	require.NotNil(t, keys[proj1.ID])
	require.NotNil(t, keys[proj2.ID])
	require.Len(t, keys[proj1.ID], 1)
	require.Len(t, keys[proj2.ID], 1)

	require.Contains(t, keys[proj1.ID][0].Private, "PRIVATE")
	require.Equal(t, keys[proj1.ID][0].Name, "ssh1")
	require.Contains(t, keys[proj2.ID][0].Private, "PRIVATE")
	require.Equal(t, keys[proj2.ID][0].Name, "ssh2")
}
