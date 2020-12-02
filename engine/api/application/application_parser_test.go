package application_test

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestParseAndImport(t *testing.T) {
	db, cache := test.SetupPG(t)
	u, _ := assets.InsertAdminUser(t, db)

	key := sdk.RandomString(10)
	appName := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
	app1 := sdk.Application{
		Name:           appName,
		FromRepository: "foo",
	}
	require.NoError(t, application.Insert(db, *proj, &app1))

	var eapp = new(exportentities.Application)

	body := []byte(`
version: v1.0
name: ` + appName + `
`)
	errapp := yaml.Unmarshal(body, eapp)
	require.NoError(t, errapp)

	_, _, _, globalError := application.ParseAndImport(context.TODO(), db, cache, *proj, eapp, application.ImportOptions{Force: false}, nil, u)
	require.Error(t, globalError)

	_, _, _, globalError2 := application.ParseAndImport(context.TODO(), db, cache, *proj, eapp, application.ImportOptions{Force: true, FromRepository: "bar"}, nil, u)
	require.Error(t, globalError2)

	_, _, _, globalError3 := application.ParseAndImport(context.TODO(), db, cache, *proj, eapp, application.ImportOptions{Force: true}, nil, u)
	require.NoError(t, globalError3)
}
