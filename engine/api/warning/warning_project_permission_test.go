package warning

import (
	"context"
	"fmt"
	"testing"

	"github.com/fatih/structs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func TestMissingProjectPermissionWorkflowWarning(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	pip := sdk.Pipeline{
		Name:      sdk.RandomString(10),
		ProjectID: proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip))

	g := sdk.Group{
		Name: sdk.RandomString(10),
	}
	assert.NoError(t, group.Insert(context.TODO(), db, &g))

	// Project KEY to test Event
	gp := sdk.GroupPermission{
		Permission: 7,
		Group:      g,
	}

	// Create Environment
	w := sdk.Workflow{
		Name:       sdk.RandomString(10),
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "node1",
				Ref:  "ref1",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}

	projUpdate, err := project.Load(db, cache, proj.Key, project.LoadOptions.WithPipelines)
	assert.NoError(t, err)
	test.NoError(t, workflow.Insert(context.TODO(), db, cache, &w, projUpdate))
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   gp.Group.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionReadWriteExecute,
	}))
	test.NoError(t, group.AddWorkflowGroup(context.TODO(), db, &w, gp))

	// Create delete key event
	ePayload := sdk.EventProjectPermissionDelete{
		Permission: gp,
	}
	e := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayload),
		Payload:    structs.Map(ePayload),
	}

	// Compute event
	warnToTest := missingProjectPermissionWorkflowWarning{}
	test.NoError(t, warnToTest.compute(context.Background(), db, e))

	// Check warning exist
	warnsAfter, errAfter := GetByProject(db, proj.Key)
	test.NoError(t, errAfter)
	assert.Equal(t, 1, len(warnsAfter))

	(&warnsAfter[0]).ComputeMessage(context.TODO(), "en")
	t.Logf("%s", warnsAfter[0].Message)

	// Create Add key event
	ePayloadAdd := sdk.EventProjectPermissionAdd{
		Permission: gp,
	}
	eAdd := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayloadAdd),
		Payload:    structs.Map(ePayloadAdd),
	}
	test.NoError(t, warnToTest.compute(context.Background(), db, eAdd))

	// Check that warning disapears
	warnsAdd, errAfterDelete := GetByProject(db, proj.Key)
	test.NoError(t, errAfterDelete)
	assert.Equal(t, 0, len(warnsAdd))

	// add warning
	test.NoError(t, warnToTest.compute(context.Background(), db, e))

	// Check warning exist
	warnsAfter2, errAfter2 := GetByProject(db, proj.Key)
	test.NoError(t, errAfter2)
	assert.Equal(t, 1, len(warnsAfter2))

	// Delete group on workflow event
	// Create Add key event
	ePayloadDelWorkflow := sdk.EventWorkflowPermissionDelete{
		Permission: gp,
	}
	eDelWorkflow := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayloadDelWorkflow),
		Payload:    structs.Map(ePayloadDelWorkflow),
	}
	test.NoError(t, warnToTest.compute(context.Background(), db, eDelWorkflow))

	warnsDelEnv, errDelEnv := GetByProject(db, proj.Key)
	test.NoError(t, errDelEnv)
	assert.Equal(t, 0, len(warnsDelEnv))
}
