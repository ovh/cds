package environment_test

import (
	"testing"

	"github.com/ovh/cds/engine/api/environment"
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
	envName := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
	env1 := sdk.Environment{
		Name:           envName,
		FromRepository: "foo",
		ProjectID:      proj.ID,
		ProjectKey:     proj.Key,
	}
	require.NoError(t, environment.InsertEnvironment(db, &env1))

	var eenv = new(exportentities.Environment)

	body := []byte(`
version: v1.0
name: ` + envName + `
`)
	errenv := yaml.Unmarshal(body, eenv)
	require.NoError(t, errenv)

	_, _, _, globalError := environment.ParseAndImport(db, *proj, *eenv, environment.ImportOptions{Force: false}, nil, u)
	require.Error(t, globalError)

	_, _, _, globalError2 := environment.ParseAndImport(db, *proj, *eenv, environment.ImportOptions{Force: true, FromRepository: "bar"}, nil, u)
	require.Error(t, globalError2)

	_, _, _, globalError3 := environment.ParseAndImport(db, *proj, *eenv, environment.ImportOptions{Force: true}, nil, u)
	require.NoError(t, globalError3)
}
