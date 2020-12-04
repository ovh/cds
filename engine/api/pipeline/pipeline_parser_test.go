package pipeline_test

import (
	"context"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/ascode"
	"github.com/ovh/cds/engine/api/pipeline"
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
	pipName := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
	pip1 := sdk.Pipeline{
		Name:           pipName,
		FromRepository: "foo",
		ProjectID:      proj.ID,
		ProjectKey:     proj.Key,
	}
	require.NoError(t, pipeline.InsertPipeline(db, &pip1))

	var epip = new(exportentities.PipelineV1)

	body := []byte(`
version: v1.0
name: ` + pipName + `
`)
	errenv := yaml.Unmarshal(body, epip)
	require.NoError(t, errenv)

	_, _, globalError := pipeline.ParseAndImport(context.TODO(), db, cache, *proj, *epip, u, pipeline.ImportOptions{Force: false})
	require.Error(t, globalError)

	_, _, globalError2 := pipeline.ParseAndImport(context.TODO(), db, cache, *proj, *epip, u, pipeline.ImportOptions{Force: true, FromRepository: "bar"})
	require.Error(t, globalError2)

	_, _, globalError3 := pipeline.ParseAndImport(context.TODO(), db, cache, *proj, *epip, u, pipeline.ImportOptions{Force: true})
	require.NoError(t, globalError3)
}
func TestParseAndImportCleanAsCode(t *testing.T) {
	db, cache := test.SetupPG(t)
	u, _ := assets.InsertAdminUser(t, db)

	key := sdk.RandomString(10)
	pipName := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
	pip1 := sdk.Pipeline{
		Name:           pipName,
		FromRepository: "myfoorepoenv",
		ProjectID:      proj.ID,
		ProjectKey:     proj.Key,
	}
	require.NoError(t, pipeline.InsertPipeline(db, &pip1))

	var epip = new(exportentities.PipelineV1)

	body := []byte(`
version: v1.0
name: ` + pipName + `
`)
	errenv := yaml.Unmarshal(body, epip)
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
			Pipelines: map[int64]string{
				pip1.ID: pip1.Name,
			},
		},
	}

	assert.NoError(t, ascode.UpsertEvent(db, &asCodeEvent))
	events, err := ascode.LoadEventsByWorkflowID(context.TODO(), db, wf.ID)
	assert.Equal(t, 1, len(events))

	// try to import with force, without a repo, it's ok
	_, _, globalError3 := pipeline.ParseAndImport(context.TODO(), db, cache, *proj, *epip, u, pipeline.ImportOptions{Force: true})
	require.NoError(t, globalError3)

	events, err = ascode.LoadEventsByWorkflowID(context.TODO(), db, wf.ID)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(events))
}
