package warning

import (
	"fmt"
	"testing"

	"github.com/fatih/structs"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"regexp"
)

func TestMissingProjectVariablePipelineJob(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	v := sdk.Variable{
		Name:  sdk.RandomString(10),
		Type:  "string",
		Value: "foo",
	}

	pip := sdk.Pipeline{
		Name:      sdk.RandomString(10),
		ProjectID: proj.ID,
		Type:      "build",
	}
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip, u))

	s := sdk.Stage{
		PipelineID: pip.ID,
		Name:       sdk.RandomString(10),
	}
	test.NoError(t, pipeline.InsertStage(db, &s))

	j := sdk.Job{
		PipelineStageID: s.ID,
		Action: sdk.Action{
			Name: sdk.RandomString(10),
			Actions: []sdk.Action{
				{
					Name: "GitClone",
					Parameters: []sdk.Parameter{
						{
							Name:  "git.branch",
							Type:  "string",
							Value: fmt.Sprintf("foo{{.cds.proj.%s}}bar", v.Name),
						},
					},
				},
			},
		},
	}
	test.NoError(t, pipeline.InsertJob(db, &j, s.ID, &pip))

	// Create Delete var
	ePayloadDelete := sdk.EventProjectVariableDelete{
		Variable: v,
	}
	eDelete := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayloadDelete),
		Payload:    structs.Map(ePayloadDelete),
	}

	// Compute event
	warnToTest := missingProjectVariablePipelineJob{}
	test.NoError(t, warnToTest.compute(db, eDelete))

	// Check warning exist
	warnsDelete, errAfter := GetByProject(db, proj.Key)
	test.NoError(t, errAfter)
	assert.Equal(t, 1, len(warnsDelete))

	(&warnsDelete[0]).ComputeMessage("en")
	t.Logf("%s", warnsDelete[0].Message)

	// Create update var event
	ePayloadUpdate := sdk.EventProjectVariableUpdate{
		OldVariable: v,
		NewVariable: v,
	}
	eUpdate := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayloadUpdate),
		Payload:    structs.Map(ePayloadUpdate),
	}
	test.NoError(t, warnToTest.compute(db, eUpdate))
	// Check warning exist
	warnsUpdate, errUpdate := GetByProject(db, proj.Key)
	test.NoError(t, errUpdate)
	assert.Equal(t, 0, len(warnsUpdate))

	// Recreate warning
	test.NoError(t, warnToTest.compute(db, eDelete))

	// Check warning exist
	warnsDelete2, errAfter2 := GetByProject(db, proj.Key)
	test.NoError(t, errAfter2)
	assert.Equal(t, 1, len(warnsDelete2))

	// Create add var event
	ePayloadAdd := sdk.EventProjectVariableAdd{
		Variable: v,
	}
	eAdd := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayloadAdd),
		Payload:    structs.Map(ePayloadAdd),
	}
	test.NoError(t, warnToTest.compute(db, eAdd))
	// Check warning exist
	warnsAdd, errAdd := GetByProject(db, proj.Key)
	test.NoError(t, errAdd)
	assert.Equal(t, 0, len(warnsAdd))
}

func TestMissingProjectVariablePipelineParameter(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	v := sdk.Variable{
		Name:  sdk.RandomString(10),
		Type:  "string",
		Value: "foo",
	}

	pip := sdk.Pipeline{
		Name:      sdk.RandomString(10),
		ProjectID: proj.ID,
		Type:      "build",
	}
	pipParam := sdk.Parameter{
		Name:  sdk.RandomString(10),
		Type:  "string",
		Value: fmt.Sprintf("foo{{.cds.proj.%s}}bar", v.Name),
	}
	pip.Parameter = []sdk.Parameter{
		pipParam,
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip, u))

	// Create Delete var
	ePayloadDelete := sdk.EventProjectVariableDelete{
		Variable: v,
	}
	eDelete := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayloadDelete),
		Payload:    structs.Map(ePayloadDelete),
	}

	// Compute event
	warnToTest := missingProjectVariablePipelineParameter{}
	test.NoError(t, warnToTest.compute(db, eDelete))

	// Check warning exist
	warnsDelete, errAfter := GetByProject(db, proj.Key)
	test.NoError(t, errAfter)
	assert.Equal(t, 1, len(warnsDelete))

	(&warnsDelete[0]).ComputeMessage("en")
	t.Logf("%s", warnsDelete[0].Message)

	// Create update var event
	ePayloadUpdate := sdk.EventProjectVariableUpdate{
		OldVariable: v,
		NewVariable: v,
	}
	eUpdate := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayloadUpdate),
		Payload:    structs.Map(ePayloadUpdate),
	}
	test.NoError(t, warnToTest.compute(db, eUpdate))
	// Check warning exist
	warnsUpdate, errUpdate := GetByProject(db, proj.Key)
	test.NoError(t, errUpdate)
	assert.Equal(t, 0, len(warnsUpdate))

	// Recreate warning
	test.NoError(t, warnToTest.compute(db, eDelete))

	// Check warning exist
	warnsDelete2, errAfter2 := GetByProject(db, proj.Key)
	test.NoError(t, errAfter2)
	assert.Equal(t, 1, len(warnsDelete2))

	// Create add var event
	ePayloadAdd := sdk.EventProjectVariableAdd{
		Variable: v,
	}
	eAdd := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayloadAdd),
		Payload:    structs.Map(ePayloadAdd),
	}
	test.NoError(t, warnToTest.compute(db, eAdd))
	// Check warning exist
	warnsAdd, errAdd := GetByProject(db, proj.Key)
	test.NoError(t, errAdd)
	assert.Equal(t, 0, len(warnsAdd))
}

func TestMissingProjectVariableApplication(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	app := sdk.Application{
		Name:      sdk.RandomString(10),
		ProjectID: proj.ID,
	}

	test.NoError(t, application.Insert(db, cache, proj, &app, u))

	v := sdk.Variable{
		Name:  sdk.RandomString(10),
		Type:  "string",
		Value: "foo",
	}

	appV := sdk.Variable{
		Name:  sdk.RandomString(10),
		Type:  "string",
		Value: "foo{{.cds.proj." + v.Name + "}}bar",
	}
	test.NoError(t, application.InsertVariable(db, cache, &app, appV, u))

	// Create Delete var
	ePayloadDelete := sdk.EventProjectVariableDelete{
		Variable: v,
	}
	eDelete := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayloadDelete),
		Payload:    structs.Map(ePayloadDelete),
	}

	// Compute event
	warnToTest := missingProjectVariableApplication{}
	test.NoError(t, warnToTest.compute(db, eDelete))

	// Check warning exist
	warnsDelete, errAfter := GetByProject(db, proj.Key)
	test.NoError(t, errAfter)
	assert.Equal(t, 1, len(warnsDelete))

	(&warnsDelete[0]).ComputeMessage("en")
	t.Logf("%s", warnsDelete[0].Message)

	// Create update var event
	ePayloadUpdate := sdk.EventProjectVariableUpdate{
		OldVariable: v,
		NewVariable: v,
	}
	eUpdate := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayloadUpdate),
		Payload:    structs.Map(ePayloadUpdate),
	}
	test.NoError(t, warnToTest.compute(db, eUpdate))
	// Check warning exist
	warnsUpdate, errUpdate := GetByProject(db, proj.Key)
	test.NoError(t, errUpdate)
	assert.Equal(t, 0, len(warnsUpdate))

	// Recreate warning
	test.NoError(t, warnToTest.compute(db, eDelete))

	// Check warning exist
	warnsDelete2, errAfter2 := GetByProject(db, proj.Key)
	test.NoError(t, errAfter2)
	assert.Equal(t, 1, len(warnsDelete2))

	// Create add var event
	ePayloadAdd := sdk.EventProjectVariableAdd{
		Variable: v,
	}
	eAdd := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayloadAdd),
		Payload:    structs.Map(ePayloadAdd),
	}
	test.NoError(t, warnToTest.compute(db, eAdd))
	// Check warning exist
	warnsAdd, errAdd := GetByProject(db, proj.Key)
	test.NoError(t, errAdd)
	assert.Equal(t, 0, len(warnsAdd))
}

func TestMissingProjectVariableWorkflow(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	pip := sdk.Pipeline{
		Name:      sdk.RandomString(10),
		ProjectID: proj.ID,
		Type:      "build",
	}
	pipParam := sdk.Parameter{
		Name:  sdk.RandomString(10),
		Type:  "string",
		Value: "",
	}
	pip.Parameter = []sdk.Parameter{
		pipParam,
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip, u))

	v := sdk.Variable{
		Name:  sdk.RandomString(10),
		Type:  "string",
		Value: "foo",
	}

	w := sdk.Workflow{
		Name:       sdk.RandomString(10),
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
		Root: &sdk.WorkflowNode{
			Pipeline: pip,
			Context: &sdk.WorkflowNodeContext{
				DefaultPipelineParameters: []sdk.Parameter{
					{
						Name:  pipParam.Name,
						Type:  "string",
						Value: fmt.Sprintf("foo{{.cds.proj.%s}}", v.Name),
					},
				},
			},
		},
	}

	projUpdate, err := project.Load(db, cache, proj.Key, u, project.LoadOptions.WithPipelines)
	assert.NoError(t, err)
	test.NoError(t, workflow.Insert(db, cache, &w, projUpdate, u))

	// Create Delete var
	ePayloadDelete := sdk.EventProjectVariableDelete{
		Variable: v,
	}
	eDelete := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayloadDelete),
		Payload:    structs.Map(ePayloadDelete),
	}

	// Compute event
	warnToTest := missingProjectVariableWorkflow{}
	test.NoError(t, warnToTest.compute(db, eDelete))

	// Check warning exist
	warnsDelete, errAfter := GetByProject(db, proj.Key)
	test.NoError(t, errAfter)
	assert.Equal(t, 1, len(warnsDelete))

	(&warnsDelete[0]).ComputeMessage("en")
	t.Logf("%s", warnsDelete[0].Message)

	// Create update var event
	ePayloadUpdate := sdk.EventProjectVariableUpdate{
		OldVariable: v,
		NewVariable: v,
	}
	eUpdate := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayloadUpdate),
		Payload:    structs.Map(ePayloadUpdate),
	}
	test.NoError(t, warnToTest.compute(db, eUpdate))
	// Check warning exist
	warnsUpdate, errUpdate := GetByProject(db, proj.Key)
	test.NoError(t, errUpdate)
	assert.Equal(t, 0, len(warnsUpdate))

	// Recreate warning
	test.NoError(t, warnToTest.compute(db, eDelete))

	// Check warning exist
	warnsDelete2, errAfter2 := GetByProject(db, proj.Key)
	test.NoError(t, errAfter2)
	assert.Equal(t, 1, len(warnsDelete2))

	// Create add var event
	ePayloadAdd := sdk.EventProjectVariableAdd{
		Variable: v,
	}
	eAdd := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayloadAdd),
		Payload:    structs.Map(ePayloadAdd),
	}
	test.NoError(t, warnToTest.compute(db, eAdd))
	// Check warning exist
	warnsAdd, errAdd := GetByProject(db, proj.Key)
	test.NoError(t, errAdd)
	assert.Equal(t, 0, len(warnsAdd))
}

func TestMissingProjectVariableEnv(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	v := sdk.Variable{
		Name:  sdk.RandomString(10),
		Type:  "string",
		Value: "foo",
	}

	envVar := sdk.Variable{
		Name:  sdk.RandomString(10),
		Type:  "string",
		Value: fmt.Sprintf("foo{{.cds.proj.%s}}bar", v.Name),
	}

	env := sdk.Environment{
		Name:      sdk.RandomString(10),
		ProjectID: proj.ID,
	}
	assert.NoError(t, environment.InsertEnvironment(db, &env))
	assert.NoError(t, environment.InsertVariable(db, env.ID, &envVar, u))

	// Create Delete var
	ePayloadDelete := sdk.EventProjectVariableDelete{
		Variable: v,
	}
	eDelete := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayloadDelete),
		Payload:    structs.Map(ePayloadDelete),
	}

	// Compute event
	warnToTest := missingProjectVariableEnv{}
	test.NoError(t, warnToTest.compute(db, eDelete))

	// Check warning exist
	warnsDelete, errAfter := GetByProject(db, proj.Key)
	test.NoError(t, errAfter)
	assert.Equal(t, 1, len(warnsDelete))

	(&warnsDelete[0]).ComputeMessage("en")
	t.Logf("%s", warnsDelete[0].Message)

	// Create update var event
	ePayloadUpdate := sdk.EventProjectVariableUpdate{
		OldVariable: v,
		NewVariable: v,
	}
	eUpdate := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayloadUpdate),
		Payload:    structs.Map(ePayloadUpdate),
	}
	test.NoError(t, warnToTest.compute(db, eUpdate))
	// Check warning exist
	warnsUpdate, errUpdate := GetByProject(db, proj.Key)
	test.NoError(t, errUpdate)
	assert.Equal(t, 0, len(warnsUpdate))

	// Recreate warning
	test.NoError(t, warnToTest.compute(db, eDelete))

	// Check warning exist
	warnsDelete2, errAfter2 := GetByProject(db, proj.Key)
	test.NoError(t, errAfter2)
	assert.Equal(t, 1, len(warnsDelete2))

	// Create add var event
	ePayloadAdd := sdk.EventProjectVariableAdd{
		Variable: v,
	}
	eAdd := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayloadAdd),
		Payload:    structs.Map(ePayloadAdd),
	}
	test.NoError(t, warnToTest.compute(db, eAdd))
	// Check warning exist
	warnsAdd, errAdd := GetByProject(db, proj.Key)
	test.NoError(t, errAdd)
	assert.Equal(t, 0, len(warnsAdd))
}

func TestUnusedProjectVariableWarningOnApplicationEvent(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	assert.NoError(t, Init())

	v := sdk.Variable{
		Name:  sdk.RandomString(10),
		Type:  "string",
		Value: "foo",
	}

	// Create delete application variable event
	ePayload := sdk.EventApplicationVariableDelete{
		Variable: sdk.Variable{
			Name:  "foo",
			Type:  "string",
			Value: fmt.Sprintf("Welcome {{.cds.proj.%s}}", v.Name),
		},
	}
	e := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayload),
		Payload:    structs.Map(ePayload),
	}

	// Compute event
	warnToTest := unusedProjectVariableWarning{}
	test.NoError(t, warnToTest.compute(db, e))

	// Check warning exist
	warnsDeleteVar, errDelVar := GetByProject(db, proj.Key)
	test.NoError(t, errDelVar)
	assert.Equal(t, 1, len(warnsDeleteVar))

	(&warnsDeleteVar[0]).ComputeMessage("en")
	t.Logf("%s", warnsDeleteVar[0].Message)
	t.Logf("%+v", warnsDeleteVar[0])

	// Create add variable evenT
	ePayloadAdd := sdk.EventApplicationVariableAdd{
		Variable: sdk.Variable{
			Name:  "foo",
			Type:  "string",
			Value: fmt.Sprintf("Welcome {{.cds.proj.%s}}", v.Name),
		},
	}
	eAdd := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayloadAdd),
		Payload:    structs.Map(ePayloadAdd),
	}
	test.NoError(t, warnToTest.compute(db, eAdd))

	// Check warning
	warnsAddVar, errAddVar := GetByProject(db, proj.Key)
	test.NoError(t, errAddVar)
	assert.Equal(t, 0, len(warnsAddVar))

	// Update variable event
	ePayloadUpdate := sdk.EventApplicationVariableUpdate{
		OldVariable: sdk.Variable{
			Name:  "foo",
			Type:  "string",
			Value: fmt.Sprintf("Welcome {{.cds.proj.%s}}", v.Name),
		},
		NewVariable: sdk.Variable{
			Name:  "foo",
			Type:  "string",
			Value: fmt.Sprintf("Welcome all", v.Name),
		},
	}
	eUpdate := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayloadUpdate),
		Payload:    structs.Map(ePayloadUpdate),
	}
	test.NoError(t, warnToTest.compute(db, eUpdate))

	// Check warning
	warnsUpdateVar, errUpdateVar := GetByProject(db, proj.Key)
	test.NoError(t, errUpdateVar)
	assert.Equal(t, 1, len(warnsUpdateVar))
}

func TestUnusedProjectVariableWarning(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	v := sdk.Variable{
		Name:  sdk.RandomString(10),
		Type:  "string",
		Value: "foo",
	}

	// Create add variable event
	ePayload := sdk.EventProjectVariableAdd{
		Variable: v,
	}
	e := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayload),
		Payload:    structs.Map(ePayload),
	}

	// Compute event
	warnToTest := unusedProjectVariableWarning{}
	test.NoError(t, warnToTest.compute(db, e))

	// Check warning exist
	warnsAfter, errAfter := GetByProject(db, proj.Key)
	test.NoError(t, errAfter)
	assert.Equal(t, 1, len(warnsAfter))

	(&warnsAfter[0]).ComputeMessage("en")
	t.Logf("%s", warnsAfter[0].Message)

	vAfter := sdk.Variable{
		Name:  sdk.RandomString(10),
		Type:  "string",
		Value: "foo",
	}
	// Create Update var event
	ePayloadUpdate := sdk.EventProjectVariableUpdate{
		OldVariable: v,
		NewVariable: vAfter,
	}
	eUpdate := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayloadUpdate),
		Payload:    structs.Map(ePayloadUpdate),
	}
	test.NoError(t, warnToTest.compute(db, eUpdate))

	// Check that warning disapears
	warnsUpdate, errAfterUpdate := GetByProject(db, proj.Key)
	test.NoError(t, errAfterUpdate)
	assert.Equal(t, 1, len(warnsUpdate))
	assert.Equal(t, fmt.Sprintf("cds.proj.%s", vAfter.Name), warnsUpdate[0].Element)

	// Create Delete var
	ePayloadDelete := sdk.EventProjectVariableDelete{
		Variable: vAfter,
	}
	eDelete := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayloadDelete),
		Payload:    structs.Map(ePayloadDelete),
	}
	test.NoError(t, warnToTest.compute(db, eDelete))
	warnsDelete, errAfterDelete := GetByProject(db, proj.Key)
	test.NoError(t, errAfterDelete)
	assert.Equal(t, 0, len(warnsDelete))
}

func TestToto(t *testing.T) {
	re, err := regexp.Compile("cds\\.proj\\.[a-zA-Z\\-_]+")
	assert.NoError(t, err)
	stringToTest := "Hi {{.cds.proj.user-rr}}, welcome to {{.cds.proj.title_fr}}"
	result := re.FindAllString(stringToTest, -1)
	t.Logf("%v", result)

}
