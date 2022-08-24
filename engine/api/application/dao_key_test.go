package application_test

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_DAOKey(t *testing.T) {
	db, cache := test.SetupPG(t)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
	app := sdk.Application{
		Name: "my-app",
	}

	require.NoError(t, application.Insert(db, *proj, &app))

	k := &sdk.ApplicationKey{
		Name:          "mykey-ssh",
		Type:          sdk.KeyTypeSSH,
		ApplicationID: app.ID,
	}
	kssh, err := keys.GenerateSSHKey(k.Name)
	require.NoError(t, err)

	k.Public = kssh.Public
	k.Private = kssh.Private
	k.KeyID = kssh.KeyID
	require.NoError(t, application.InsertKey(db, k))
	assert.Equal(t, sdk.PasswordPlaceholder, k.Private)

	ks, err := application.LoadAllKeys(context.TODO(), db, app.ID)
	require.NoError(t, err)

	assert.Equal(t, sdk.PasswordPlaceholder, ks[0].Private)

	ks, err = application.LoadAllKeysWithPrivateContent(context.TODO(), db, app.ID)
	require.NoError(t, err)

	assert.Equal(t, kssh.Private, ks[0].Private)

	require.NoError(t, application.DeleteKey(db, app.ID, k.Name))
}

func Test_DAOAllKeysAllApps(t *testing.T) {
	t.SkipNow() // skipping this test because the DAO is only used in a migration func
	db, cache := test.SetupPG(t)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
	app1 := sdk.Application{
		Name: "my-app1",
	}
	app2 := sdk.Application{
		Name: "my-app2",
	}
	require.NoError(t, application.Insert(db, *proj, &app1))
	require.NoError(t, application.Insert(db, *proj, &app2))

	ssh1, err := keys.GenerateSSHKey("ssh1")
	require.NoError(t, err)
	ssh2, err := keys.GenerateSSHKey("ssh2")
	require.NoError(t, err)
	appssh1 := sdk.ApplicationKey{ApplicationID: app1.ID, Type: sdk.KeyTypeSSH, Name: "ssh1", Public: ssh1.Public, Private: ssh1.Private}
	appssh2 := sdk.ApplicationKey{ApplicationID: app2.ID, Type: sdk.KeyTypeSSH, Name: "ssh2", Public: ssh2.Public, Private: ssh2.Private}
	require.NoError(t, application.InsertKey(db, &appssh1))
	require.NoError(t, application.InsertKey(db, &appssh2))

	keys, err := application.LoadAllKeysForAppsWithDecryption(context.TODO(), db, []int64{app1.ID, app2.ID})
	require.NoError(t, err)

	require.Len(t, keys, 2)
	require.NotNil(t, keys[app1.ID])
	require.NotNil(t, keys[app2.ID])
	require.Len(t, keys[app1.ID], 1)
	require.Len(t, keys[app2.ID], 1)
	require.Contains(t, keys[app1.ID][0].Private, "PRIVATE")
	require.Contains(t, keys[app1.ID][0].Name, "ssh1")
	require.Contains(t, keys[app2.ID][0].Private, "PRIVATE")
	require.Contains(t, keys[app2.ID][0].Name, "ssh2")
}
