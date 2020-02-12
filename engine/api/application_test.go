package api

import (
	"context"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/exportentities"
)

func Test_postApplicationMetadataHandler_AsProvider(t *testing.T) {
	api, tsURL, tsClose := newTestServer(t)
	defer tsClose()

	u, _ := assets.InsertAdminUser(t, api.mustDB())
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)
	_, jws, err := builtin.NewConsumer(context.TODO(), api.mustDB(), sdk.RandomString(10), sdk.RandomString(10), localConsumer, u.GetGroupIDs(), Scope(sdk.AuthConsumerScopeProject))

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, api.mustDB(), api.Cache, pkey, pkey)
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), api.mustDB(), &group.LinkGroupUser{
		GroupID:            proj.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
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

func Test_getAsCodeApplicationHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	u, pass := assets.InsertAdminUser(t, db)
	pkey := sdk.RandomString(10)
	p := assets.InsertTestProject(t, db, api.Cache, pkey, pkey)

	assert.NoError(t, repositoriesmanager.InsertForProject(db, p, &sdk.ProjectVCSServer{
		Name: "github",
		Data: map[string]string{
			"token":  "foo",
			"secret": "bar",
		},
	}))

	// Add application
	appS := `version: v1.0
name: blabla
vcs_server: github
repo: sguiheux/demo
vcs_ssh_key: proj-blabla
`
	var eapp = new(exportentities.Application)
	assert.NoError(t, yaml.Unmarshal([]byte(appS), eapp))
	app, _, globalError := application.ParseAndImport(context.Background(), db, api.Cache, p, eapp, application.ImportOptions{Force: true}, nil, u)
	assert.NoError(t, globalError)

	app.FromRepository = "myrepository"
	assert.NoError(t, application.Update(db, api.Cache, app))

	uri := api.Router.GetRoute("GET", api.getAsCodeApplicationHandler, map[string]string{
		"permProjectKey": pkey,
	})
	uri = fmt.Sprintf("%s?repo=myrepository", uri)

	req, err := http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var appDB []sdk.Application
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &appDB))
	assert.Equal(t, app.ID, appDB[0].ID)

}
