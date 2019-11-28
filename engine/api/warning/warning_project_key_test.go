package warning

import (
	"context"
	"fmt"
	"testing"

	"github.com/fatih/structs"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func TestUnusedProjectKeyWarningEventProjectKeyAdd(t *testing.T) {
	// INIT
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	_ = event.Initialize(context.Background(), db, cache)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	// Projecr KEY to test Event
	pKey := sdk.ProjectKey{
		ProjectID: proj.ID,
		Key: sdk.Key{
			Name: sdk.RandomString(3),
		},
	}

	// Create Add key event
	ePayload := sdk.EventProjectKeyAdd{
		Key: pKey,
	}
	e := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayload),
		Payload:    structs.Map(ePayload),
	}

	// Compute event
	warnToTest := unusedProjectKeyWarning{}
	test.NoError(t, warnToTest.compute(context.Background(), db, e))

	// Check warning exist
	warnsAfter, errAfter := GetByProject(db, proj.Key)
	test.NoError(t, errAfter)
	assert.Equal(t, 1, len(warnsAfter))

	(&warnsAfter[0]).ComputeMessage(context.TODO(), "en")
	t.Logf("%s", warnsAfter[0].Message)

	// Create Key deletion event
	ePlayloadDelete := sdk.EventProjectKeyDelete{
		Key: pKey,
	}
	eDelete := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePlayloadDelete),
		Payload:    structs.Map(ePlayloadDelete),
	}

	// Compute event
	test.NoError(t, warnToTest.compute(context.Background(), db, eDelete))

	// Check that warning disapears
	warnsAfterDelete, errAfterDelete := GetByProject(db, proj.Key)
	test.NoError(t, errAfterDelete)
	assert.Equal(t, 0, len(warnsAfterDelete))
}

func TestMissingProjectKeyPipelineParameterWarning(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	// Project KEY to test Event
	pKey := sdk.ProjectKey{
		ProjectID: proj.ID,
		Key: sdk.Key{
			Name: sdk.RandomString(3),
		},
	}

	// Create pipeline
	pip := sdk.Pipeline{
		Name:      sdk.RandomString(10),
		ProjectID: proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip))

	// Add a parameter that use project key
	pa := sdk.Parameter{
		Name:  "foo",
		Type:  sdk.StringParameter,
		Value: fmt.Sprintf("foo bar %s bar foo", pKey.Name),
	}
	assert.NoError(t, pipeline.InsertParameterInPipeline(db, pip.ID, &pa))

	// Create delete key event
	ePayload := sdk.EventProjectKeyDelete{
		Key: pKey,
	}
	e := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayload),
		Payload:    structs.Map(ePayload),
	}

	// Compute event
	warnToTest := missingProjectKeyPipelineParameterWarning{}
	test.NoError(t, warnToTest.compute(context.Background(), db, e))

	// Check warning exist
	warnsAfter, errAfter := GetByProject(db, proj.Key)
	test.NoError(t, errAfter)
	assert.Equal(t, 1, len(warnsAfter))

	(&warnsAfter[0]).ComputeMessage(context.TODO(), "en")
	t.Logf("%s", warnsAfter[0].Message)

	// Create Add key event
	ePayloadAdd := sdk.EventProjectKeyAdd{
		Key: pKey,
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
}

func TestMissingProjectKeyPipelineJobWarning(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	// Project KEY to test Event
	pKey := sdk.ProjectKey{
		ProjectID: proj.ID,
		Key: sdk.Key{
			Name: sdk.RandomString(3),
		},
	}

	// get git clone action
	gitClone := assets.GetBuiltinOrPluginActionByName(t, db, sdk.GitCloneAction)

	// Create pipeline
	pip := sdk.Pipeline{
		Name:      sdk.RandomString(10),
		ProjectID: proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip))
	s := sdk.Stage{Name: "MyStage", PipelineID: pip.ID}
	test.NoError(t, pipeline.InsertStage(db, &s))
	j := sdk.Job{PipelineStageID: s.ID, Action: sdk.Action{
		Name: "MyJob",
		Actions: []sdk.Action{
			{
				ID: gitClone.ID,
				Parameters: []sdk.Parameter{
					{
						Name:  "privateKey",
						Type:  sdk.StringParameter,
						Value: fmt.Sprintf("blabla %s blabla", pKey.Name),
					},
				},
			},
		},
	}}
	test.NoError(t, pipeline.InsertJob(db, &j, s.ID, &pip))

	// Create delete key event
	ePayload := sdk.EventProjectKeyDelete{
		Key: pKey,
	}
	e := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayload),
		Payload:    structs.Map(ePayload),
	}

	// Compute event
	warnToTest := missingProjectKeyPipelineJobWarning{}
	test.NoError(t, warnToTest.compute(context.Background(), db, e))

	// Check warning exist
	warnsAfter, errAfter := GetByProject(db, proj.Key)
	test.NoError(t, errAfter)
	assert.Equal(t, 1, len(warnsAfter))

	(&warnsAfter[0]).ComputeMessage(context.TODO(), "en")
	t.Logf("%s", warnsAfter[0].Message)

	// Create Add key event
	ePayloadAdd := sdk.EventProjectKeyAdd{
		Key: pKey,
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
}

func TestMissingProjectKeyApplicationWarning(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	// Project KEY to test Event
	pKey := sdk.ProjectKey{
		ProjectID: proj.ID,
		Key: sdk.Key{
			Name: sdk.RandomString(3),
		},
	}

	// create application
	app := sdk.Application{
		Name:      sdk.RandomString(10),
		ProjectID: proj.ID,
	}
	assert.NoError(t, application.Insert(db, cache, proj, &app))

	// Setup ssh key
	app.RepositoryStrategy = sdk.RepositoryStrategy{
		SSHKey: pKey.Name,
	}
	assert.NoError(t, application.Update(db, cache, &app))

	// Create delete key event
	ePayload := sdk.EventProjectKeyDelete{
		Key: pKey,
	}
	e := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayload),
		Payload:    structs.Map(ePayload),
	}

	// Compute event
	warnToTest := missingProjectKeyApplicationWarning{}
	test.NoError(t, warnToTest.compute(context.Background(), db, e))

	// Check warning exist
	warnsAfter, errAfter := GetByProject(db, proj.Key)
	test.NoError(t, errAfter)
	assert.Equal(t, 1, len(warnsAfter))

	(&warnsAfter[0]).ComputeMessage(context.TODO(), "en")
	t.Logf("%s", warnsAfter[0].Message)

	// Create Add key event
	ePayloadAdd := sdk.EventProjectKeyAdd{
		Key: pKey,
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

}
