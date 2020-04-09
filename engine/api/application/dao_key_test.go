package application_test

import (
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
	db, cache, end := test.SetupPG(t)
	defer end()

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

	ks, err := application.LoadAllKeys(db, app.ID)
	require.NoError(t, err)

	assert.Equal(t, sdk.PasswordPlaceholder, ks[0].Private)

	ks, err = application.LoadAllKeysWithPrivateContent(db, app.ID)
	require.NoError(t, err)

	assert.Equal(t, kssh.Private, ks[0].Private)

	require.NoError(t, application.DeleteKey(db, app.ID, k.Name))
}
