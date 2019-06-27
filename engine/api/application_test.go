package api

import (
	"testing"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/stretchr/testify/assert"
)

func Test_postApplicationMetadataHandler_AsProvider(t *testing.T) {
	api, tsURL, tsClose := newTestServer(t)
	defer tsClose()

	u, _ := assets.InsertAdminUser(api.mustDB())
	_, jws, err := builtin.NewConsumer(api.mustDB(), sdk.RandomString(10), sdk.RandomString(10), u.ID, u.GetGroupIDs(), Scope(sdk.AccessTokenScopeProject))

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, api.mustDB(), api.Cache, pkey, pkey, u)
	test.NoError(t, group.InsertUserInGroup(api.mustDB(), proj.ProjectGroups[0].Group.ID, u.OldUserStruct.ID, true))

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
