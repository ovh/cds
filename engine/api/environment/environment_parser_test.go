package environment_test

import (
	"context"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/ascode"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/stretchr/testify/assert"
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

	_, _, _, globalError := environment.ParseAndImport(context.TODO(), db, *proj, *eenv, environment.ImportOptions{Force: false}, nil, u, nil)
	require.Error(t, globalError)

	_, _, _, globalError2 := environment.ParseAndImport(context.TODO(), db, *proj, *eenv, environment.ImportOptions{Force: true, FromRepository: "bar"}, nil, u, nil)
	require.Error(t, globalError2)

	_, _, _, globalError3 := environment.ParseAndImport(context.TODO(), db, *proj, *eenv, environment.ImportOptions{Force: true}, nil, u, nil)
	require.NoError(t, globalError3)
}
func TestParseAndImportCleanAsCode(t *testing.T) {
	db, cache := test.SetupPG(t)
	u, _ := assets.InsertAdminUser(t, db)

	key := sdk.RandomString(10)
	envName := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
	env1 := sdk.Environment{
		Name:           envName,
		FromRepository: "myfoorepoenv",
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

	require.NoError(t, action.CreateBuiltinActions(db))
	wf := assets.InsertTestWorkflow(t, db, cache, proj, "workflow1")

	// Add some events to resync
	asCodeEvent := sdk.AsCodeEvent{
		WorkflowID: wf.ID,
		Username:   u.GetUsername(),
		CreateDate: time.Now(),
		FromRepo:   "myfoorepoenv",
		Data: sdk.AsCodeEventData{
			Environments: map[int64]string{
				env1.ID: env1.Name,
			},
		},
	}

	assert.NoError(t, ascode.UpsertEvent(db, &asCodeEvent))
	events, err := ascode.LoadEventsByWorkflowID(context.TODO(), db, wf.ID)
	assert.Equal(t, 1, len(events))

	// try to import with force, without a repo, it's ok
	_, _, _, globalError3 := environment.ParseAndImport(context.TODO(), db, *proj, *eenv, environment.ImportOptions{Force: true}, nil, u, nil)
	require.NoError(t, globalError3)

	events, err = ascode.LoadEventsByWorkflowID(context.TODO(), db, wf.ID)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(events))
}
