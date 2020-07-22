package environment_test

import (
	"context"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_DAOAllKeysAllEnvs(t *testing.T) {
	db, cache := test.SetupPG(t)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	env1 := sdk.Environment{Name: "test1", ProjectID: proj.ID}
	env2 := sdk.Environment{Name: "test2", ProjectID: proj.ID}

	require.NoError(t, environment.InsertEnvironment(db, &env1))
	require.NoError(t, environment.InsertEnvironment(db, &env2))

	ssh1, err := keys.GenerateSSHKey("ssh1")
	require.NoError(t, err)
	ssh2, err := keys.GenerateSSHKey("ssh2")
	require.NoError(t, err)
	envssh1 := sdk.EnvironmentKey{EnvironmentID: env1.ID, Type: sdk.KeyTypeSSH, Name: "ssh1", Public: ssh1.Public, Private: ssh1.Private}
	envssh2 := sdk.EnvironmentKey{EnvironmentID: env2.ID, Type: sdk.KeyTypeSSH, Name: "ssh2", Public: ssh2.Public, Private: ssh2.Private}
	require.NoError(t, environment.InsertKey(db, &envssh1))
	require.NoError(t, environment.InsertKey(db, &envssh2))

	keys, err := environment.LoadAllKeysForEnvsWithDecryption(context.TODO(), db, []int64{env1.ID, env2.ID})
	require.NoError(t, err)

	require.Len(t, keys, 2)
	require.NotNil(t, keys[env1.ID])
	require.NotNil(t, keys[env2.ID])
	require.Len(t, keys[env1.ID], 1)
	require.Len(t, keys[env2.ID], 1)
	require.Contains(t, keys[env1.ID][0].Private, "PRIVATE")
	require.Contains(t, keys[env1.ID][0].Name, "ssh1")
	require.Contains(t, keys[env2.ID][0].Private, "PRIVATE")
	require.Contains(t, keys[env2.ID][0].Name, "ssh2")
}
