package api

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

func Test_postApplicationMetadataHandler_AsProvider(t *testing.T) {
	api, tsURL, tsClose := newTestServer(t)
	defer tsClose()

	api.Config.Providers = append(api.Config.Providers, ProviderConfiguration{
		Name:  "test-provider",
		Token: "my-token",
	})

	u, _ := assets.InsertAdminUser(api.mustDB())

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, api.mustDB(), api.Cache, pkey, pkey, u)
	test.NoError(t, group.InsertUserInGroup(api.mustDB(), proj.ProjectGroups[0].Group.ID, u.ID, true))

	//Insert Application
	app := &sdk.Application{
		Name: sdk.RandomString(10),
		Metadata: map[string]string{
			"a1": "a1",
		},
	}
	if err := application.Insert(api.mustDB(), api.Cache, proj, app, u); err != nil {
		t.Fatal(err)
	}

	sdkclient := cdsclient.NewProviderClient(cdsclient.ProviderConfig{
		Host:  tsURL,
		Name:  "test-provider",
		Token: "my-token",
	})

	test.NoError(t, sdkclient.ApplicationMetadataUpdate(pkey, app.Name, "b1", "b1"))
	app, err := application.LoadByName(api.mustDB(), api.Cache, pkey, app.Name)
	test.NoError(t, err)
	assert.Equal(t, "a1", app.Metadata["a1"])
	assert.Equal(t, "b1", app.Metadata["b1"])

}
