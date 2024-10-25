package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/ascode"
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

	// try to import without force, it must give an error
	_, _, _, globalError := application.ParseAndImport(context.TODO(), db, cache, *proj, eapp, application.ImportOptions{Force: false}, nil, u, nil)
	require.Error(t, globalError)

	// try to import with force, but with another repository, it must give an error
	_, _, _, globalError2 := application.ParseAndImport(context.TODO(), db, cache, *proj, eapp, application.ImportOptions{Force: true, FromRepository: "bar"}, nil, u, nil)
	require.Error(t, globalError2)

	// try to import with force, without a repo, it's ok
	_, _, _, globalError3 := application.ParseAndImport(context.TODO(), db, cache, *proj, eapp, application.ImportOptions{Force: true}, nil, u, nil)
	require.NoError(t, globalError3)

}

func TestParseAndImportAsCodeEvent(t *testing.T) {
	db, cache := test.SetupPG(t)
	u, _ := assets.InsertAdminUser(t, db)

	key := sdk.RandomString(10)
	appName := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
	app1 := sdk.Application{
		Name:           appName,
		FromRepository: "myfoorepo",
	}
	require.NoError(t, application.Insert(db, *proj, &app1))

	var eapp = new(exportentities.Application)

	body := []byte(`
version: v1.0
name: ` + appName + `
`)
	errapp := yaml.Unmarshal(body, eapp)
	require.NoError(t, errapp)

	require.NoError(t, action.CreateBuiltinActions(db))
	wf := assets.InsertTestWorkflow(t, db, cache, proj, "workflow1")

	// Add some events to resync
	asCodeEvent := sdk.AsCodeEvent{
		WorkflowID: wf.ID,
		Username:   u.GetUsername(),
		CreateDate: time.Now(),
		FromRepo:   "myfoorepo",
		Data: sdk.AsCodeEventData{
			Applications: map[int64]string{
				app1.ID: app1.Name,
			},
		},
	}

	assert.NoError(t, ascode.UpsertEvent(db, &asCodeEvent))
	events, err := ascode.LoadEventsByWorkflowID(context.TODO(), db, wf.ID)
	assert.Equal(t, 1, len(events))

	// try to import with force, without a repo, it's ok
	_, _, _, globalError3 := application.ParseAndImport(context.TODO(), db, cache, *proj, eapp, application.ImportOptions{Force: true}, nil, u, nil)
	require.NoError(t, globalError3)

	events, err = ascode.LoadEventsByWorkflowID(context.TODO(), db, wf.ID)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(events))
}
