package warning

import (
	"fmt"
	"github.com/fatih/structs"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUnusedProjectKeyWarningEventProjectKeyAdd(t *testing.T) {
	// INIT
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

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
	test.NoError(t, warnToTest.compute(db, e))

	// Check warning exist
	warnsAfter, errAfter := GetByProject(db, proj.Key)
	test.NoError(t, errAfter)
	assert.Equal(t, 1, len(warnsAfter))

	(&warnsAfter[0]).ComputeMessage("en")
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
	test.NoError(t, warnToTest.compute(db, eDelete))

	// Check that warning disapears
	warnsAfterDelete, errAfterDelete := GetByProject(db, proj.Key)
	test.NoError(t, errAfterDelete)
	assert.Equal(t, 0, len(warnsAfterDelete))
}

func TestMissingProjectKeyPipelineParameterWarning(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

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
		Type:      "build",
	}
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip, u))

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
	test.NoError(t, warnToTest.compute(db, e))

	// Check warning exist
	warnsAfter, errAfter := GetByProject(db, proj.Key)
	test.NoError(t, errAfter)
	assert.Equal(t, 1, len(warnsAfter))

	(&warnsAfter[0]).ComputeMessage("en")
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
	test.NoError(t, warnToTest.compute(db, eAdd))

	// Check that warning disapears
	warnsAdd, errAfterDelete := GetByProject(db, proj.Key)
	test.NoError(t, errAfterDelete)
	assert.Equal(t, 0, len(warnsAdd))
}

func TestMissingProjectKeyPipelineJobWarning(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

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
		Type:      "build",
	}
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip, u))
	s := sdk.Stage{Name: "MyStage", PipelineID: pip.ID}
	test.NoError(t, pipeline.InsertStage(db, &s))
	j := sdk.Job{PipelineStageID: s.ID, Action: sdk.Action{
		Name: "MyJob",
		Actions: []sdk.Action{
			{
				Name: "GitClone",
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
	test.NoError(t, warnToTest.compute(db, e))

	// Check warning exist
	warnsAfter, errAfter := GetByProject(db, proj.Key)
	test.NoError(t, errAfter)
	assert.Equal(t, 1, len(warnsAfter))

	(&warnsAfter[0]).ComputeMessage("en")
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
	test.NoError(t, warnToTest.compute(db, eAdd))

	// Check that warning disapears
	warnsAdd, errAfterDelete := GetByProject(db, proj.Key)
	test.NoError(t, errAfterDelete)
	assert.Equal(t, 0, len(warnsAdd))
}

func TestMissingProjectKeyApplicationWarning(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

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
	assert.NoError(t, application.Insert(db, cache, proj, &app, u))

	// Setup ssh key
	app.RepositoryStrategy = sdk.RepositoryStrategy{
		SSHKey: pKey.Name,
	}
	assert.NoError(t, application.Update(db, cache, &app, u))

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
	test.NoError(t, warnToTest.compute(db, e))

	// Check warning exist
	warnsAfter, errAfter := GetByProject(db, proj.Key)
	test.NoError(t, errAfter)
	assert.Equal(t, 1, len(warnsAfter))

	(&warnsAfter[0]).ComputeMessage("en")
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
	test.NoError(t, warnToTest.compute(db, eAdd))

	// Check that warning disapears
	warnsAdd, errAfterDelete := GetByProject(db, proj.Key)
	test.NoError(t, errAfterDelete)
	assert.Equal(t, 0, len(warnsAdd))

}
