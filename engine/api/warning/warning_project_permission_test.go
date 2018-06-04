package warning

import (
	"fmt"
	"testing"

	"github.com/fatih/structs"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func TestMissingProjectPermissionEnvWarning(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	g := sdk.Group{
		Name: sdk.RandomString(10),
	}
	assert.NoError(t, group.InsertGroup(db, &g))

	// Project KEY to test Event
	gp := sdk.GroupPermission{
		Permission: 7,
		Group:      g,
	}

	// Create Environment
	env := sdk.Environment{
		Name:      sdk.RandomString(10),
		ProjectID: proj.ID,
	}
	test.NoError(t, environment.InsertEnvironment(db, &env))
	test.NoError(t, group.InsertGroupInEnvironment(db, env.ID, g.ID, 7))

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
	warnToTest := missingProjectPermissionEnvWarning{}
	test.NoError(t, warnToTest.compute(db, e))

	// Check warning exist
	warnsAfter, errAfter := GetByProject(db, proj.Key)
	test.NoError(t, errAfter)
	assert.Equal(t, 1, len(warnsAfter))

	(&warnsAfter[0]).ComputeMessage("en")
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
	test.NoError(t, warnToTest.compute(db, eAdd))

	// Check that warning disapears
	warnsAdd, errAfterDelete := GetByProject(db, proj.Key)
	test.NoError(t, errAfterDelete)
	assert.Equal(t, 0, len(warnsAdd))
}

func TestMissingProjectPermissionWorkflowWarning(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	pip := sdk.Pipeline{
		Name:      sdk.RandomString(10),
		ProjectID: proj.ID,
		Type:      "build",
	}
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip, u))

	g := sdk.Group{
		Name: sdk.RandomString(10),
	}
	assert.NoError(t, group.InsertGroup(db, &g))

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
		Root: &sdk.WorkflowNode{
			Pipeline: pip,
		},
	}

	projUpdate, err := project.Load(db, cache, proj.Key, u, project.LoadOptions.WithPipelines)
	assert.NoError(t, err)
	test.NoError(t, workflow.Insert(db, cache, &w, projUpdate, u))
	test.NoError(t, workflow.AddGroup(db, &w, gp))

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
	test.NoError(t, warnToTest.compute(db, e))

	// Check warning exist
	warnsAfter, errAfter := GetByProject(db, proj.Key)
	test.NoError(t, errAfter)
	assert.Equal(t, 1, len(warnsAfter))

	(&warnsAfter[0]).ComputeMessage("en")
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
	test.NoError(t, warnToTest.compute(db, eAdd))

	// Check that warning disapears
	warnsAdd, errAfterDelete := GetByProject(db, proj.Key)
	test.NoError(t, errAfterDelete)
	assert.Equal(t, 0, len(warnsAdd))
}
