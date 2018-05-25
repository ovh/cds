package warning

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_VarUsedInEnvironment(t *testing.T) {
	db, store := test.SetupPG(t, bootstrap.InitiliazeDB)
	if db == nil {
		t.FailNow()
	}
	u, _ := assets.InsertAdminUser(db)

	p := sdk.Project{
		Key:  sdk.RandomString(10),
		Name: sdk.RandomString(10),
	}
	assert.NoError(t, project.Insert(db, store, &p, u))

	e := sdk.Environment{
		Name:      sdk.RandomString(10),
		ProjectID: p.ID,
	}
	assert.NoError(t, environment.InsertEnvironment(db, &e))

	prjVar := sdk.Variable{
		Name:  "foo",
		Type:  "string",
		Value: "world",
	}
	assert.NoError(t, project.InsertVariable(db, &p, &prjVar, u))

	v := sdk.Variable{
		Name:  sdk.RandomString(3),
		Type:  "string",
		Value: "foo {{.cds.proj.foo}} bar",
	}
	assert.NoError(t, environment.InsertVariable(db, e.ID, &v, u))

	envs, apps, pips := variableIsUsed(db, p.Key, "{{.cds.proj.foo}}")
	assert.Equal(t, 1, len(envs))
	assert.Equal(t, 0, len(apps))
	assert.Equal(t, 0, len(pips))
}

func Test_VarUsedInApplication(t *testing.T) {
	db, store := test.SetupPG(t, bootstrap.InitiliazeDB)
	if db == nil {
		t.FailNow()
	}
	u, _ := assets.InsertAdminUser(db)

	p := &sdk.Project{
		Key:  sdk.RandomString(10),
		Name: sdk.RandomString(10),
	}
	assert.NoError(t, project.Insert(db, store, p, u))

	a := &sdk.Application{
		Name:      sdk.RandomString(10),
		ProjectID: p.ID,
	}
	assert.NoError(t, application.Insert(db, store, p, a, u))

	prjVar := sdk.Variable{
		Name:  "foo",
		Type:  "string",
		Value: "world",
	}
	assert.NoError(t, project.InsertVariable(db, p, &prjVar, u))

	v := sdk.Variable{
		Name:  sdk.RandomString(3),
		Type:  "string",
		Value: "foo {{.cds.proj.foo}} bar",
	}
	assert.NoError(t, application.InsertVariable(db, store, a, v, u))

	envs, apps, pips := variableIsUsed(db, p.Key, "{{.cds.proj.foo}}")
	assert.Equal(t, 0, len(envs))
	assert.Equal(t, 1, len(apps))
	assert.Equal(t, 0, len(pips))
}

func Test_VarUsedInPipelineParameter(t *testing.T) {
	db, store := test.SetupPG(t, bootstrap.InitiliazeDB)
	if db == nil {
		t.FailNow()
	}
	u, _ := assets.InsertAdminUser(db)

	p := &sdk.Project{
		Key:  sdk.RandomString(10),
		Name: sdk.RandomString(10),
	}
	assert.NoError(t, project.Insert(db, store, p, u))

	pip := &sdk.Pipeline{
		Name:      sdk.RandomString(10),
		ProjectID: p.ID,
		Type:      "build",
	}
	assert.NoError(t, pipeline.InsertPipeline(db, store, p, pip, u))

	prjVar := sdk.Variable{
		Name:  "foo",
		Type:  "string",
		Value: "world",
	}
	assert.NoError(t, project.InsertVariable(db, p, &prjVar, u))

	param := sdk.Parameter{
		Name:  sdk.RandomString(3),
		Type:  "string",
		Value: "foo {{.cds.proj.foo}} bar",
	}
	assert.NoError(t, pipeline.InsertParameterInPipeline(db, pip.ID, &param))

	envs, apps, pips := variableIsUsed(db, p.Key, "{{.cds.proj.foo}}")
	assert.Equal(t, 0, len(envs))
	assert.Equal(t, 0, len(apps))
	assert.Equal(t, 1, len(pips))
}

func Test_VarUsedInPipelineJob(t *testing.T) {
	db, store := test.SetupPG(t, bootstrap.InitiliazeDB)
	if db == nil {
		t.FailNow()
	}
	u, _ := assets.InsertAdminUser(db)

	p := &sdk.Project{
		Key:  sdk.RandomString(10),
		Name: sdk.RandomString(10),
	}
	assert.NoError(t, project.Insert(db, store, p, u))

	pip := &sdk.Pipeline{
		Name:      sdk.RandomString(10),
		ProjectID: p.ID,
		Type:      "build",
	}
	assert.NoError(t, pipeline.InsertPipeline(db, store, p, pip, u))

	prjVar := sdk.Variable{
		Name:  "foo",
		Type:  "string",
		Value: "world",
	}
	assert.NoError(t, project.InsertVariable(db, p, &prjVar, u))

	s := &sdk.Stage{
		Name:       "Stage1",
		PipelineID: pip.ID,
	}
	assert.NoError(t, pipeline.InsertStage(db, s))

	j := &sdk.Job{
		PipelineStageID: s.ID,
		Action: sdk.Action{
			Name: "MyJOb",
		},
	}
	assert.NoError(t, pipeline.InsertJob(db, j, s.ID, pip))

	j.Action.Actions = []sdk.Action{
		{
			Name: sdk.ScriptAction,
			Parameters: []sdk.Parameter{
				{
					Name:  "script",
					Value: "hello {{.cds.proj.foo}}",
				},
			},
		},
	}
	assert.NoError(t, pipeline.UpdateJob(db, j, u.ID))

	envs, apps, pips := variableIsUsed(db, p.Key, "{{.cds.proj.foo}}")
	assert.Equal(t, 0, len(envs))
	assert.Equal(t, 0, len(apps))
	assert.Equal(t, 1, len(pips))
}
