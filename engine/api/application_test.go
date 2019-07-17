package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

func Test_postApplicationMetadataHandler_AsProvider(t *testing.T) {
	api, tsURL, tsClose := newTestServer(t)
	defer tsClose()

	u, _ := assets.InsertAdminUser(api.mustDB())
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)
	_, jws, err := builtin.NewConsumer(api.mustDB(), sdk.RandomString(10), sdk.RandomString(10), localConsumer, u.GetGroupIDs(), Scope(sdk.AuthConsumerScopeProject))

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, api.mustDB(), api.Cache, pkey, pkey)
	require.NoError(t, group.InsertLinkGroupUser(api.mustDB(), &group.LinkGroupUser{
		GroupID: proj.ProjectGroups[0].Group.ID,
		UserID:  u.OldUserStruct.ID,
		Admin:   true,
	}))

	//Insert Application
	app := &sdk.Application{
		Name: sdk.RandomString(10),
		Metadata: map[string]string{
			"a1": "a1",
		},
	}
	if err := application.Insert(api.mustDB(), api.Cache, proj, app); err != nil {
		t.Fatal(err)
	}

	sdkclient := cdsclient.NewProviderClient(cdsclient.ProviderConfig{
		Host:  tsURL,
		Token: jws,
	})

	test.NoError(t, sdkclient.ApplicationMetadataUpdate(pkey, app.Name, "b1", "b1"))
	app, err = application.LoadByName(api.mustDB(), api.Cache, pkey, app.Name)
	test.NoError(t, err)
	assert.Equal(t, "a1", app.Metadata["a1"])
	assert.Equal(t, "b1", app.Metadata["b1"])

	apps, err := sdkclient.ApplicationsList(pkey, cdsclient.FilterByUser(u.Username), cdsclient.FilterByWritablePermission())
	test.NoError(t, err)
	assert.Equal(t, 1, len(apps))
}
