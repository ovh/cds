package workermodel_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/sdk"
)

// create handler tests
func TestCreateModel(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	g := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	u, _ := assets.InsertLambdaUser(t, db)

	pattern := sdk.ModelPattern{
		Name: sdk.RandomString(10),
		Type: sdk.Docker,
		Model: sdk.ModelCmds{
			Cmd:   "my cmd",
			Shell: "my shell",
			Envs: map[string]string{
				"one": "value",
			},
		},
	}
	require.NoError(t, workermodel.InsertPattern(db, &pattern))

	res, err := workermodel.Create(context.TODO(), db, sdk.Model{
		Type:        sdk.Docker,
		Name:        sdk.RandomString(10),
		PatternName: pattern.Name,
		GroupID:     g.ID,
	}, u)
	require.NoError(t, err)
	assert.Equal(t, sdk.Docker, res.Type)
	assert.Equal(t, pattern.Model.Cmd, res.ModelDocker.Cmd)
	assert.Equal(t, u.Username, res.Author.Username)
}

func TestUpdateModel(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	g1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	u, _ := assets.InsertLambdaUser(t, db)

	pattern := sdk.ModelPattern{
		Name: sdk.RandomString(10),
		Type: sdk.Docker,
		Model: sdk.ModelCmds{
			Cmd:   "pattern cmd",
			Shell: "pattern shell",
		},
	}
	require.NoError(t, workermodel.InsertPattern(db, &pattern))

	model1Name := sdk.RandomString(10)
	model1, err := workermodel.Create(context.TODO(), db, sdk.Model{
		Type:    sdk.Docker,
		Name:    model1Name,
		GroupID: g1.ID,
		ModelDocker: sdk.ModelDocker{
			Cmd:      "cmd",
			Private:  true,
			Password: "12345678",
		},
	}, u)
	require.NoError(t, err)
	assert.Equal(t, "cmd", model1.ModelDocker.Cmd)

	model1Clear, err := workermodel.LoadByIDWithClearPassword(db, model1.ID)
	require.NoError(t, err)
	assert.Equal(t, "12345678", model1Clear.ModelDocker.Password)

	model2Name := sdk.RandomString(10)
	_, err = workermodel.Create(context.TODO(), db, sdk.Model{
		Name:    model2Name,
		GroupID: g2.ID,
	}, u)
	require.NoError(t, err)

	// Test update some fields
	res, err := workermodel.Update(context.TODO(), db, model1, sdk.Model{
		Type:        sdk.Docker,
		Name:        model1Name,
		PatternName: pattern.Name,
		GroupID:     g1.ID,
		ModelDocker: sdk.ModelDocker{
			Private:  true,
			Password: sdk.PasswordPlaceholder,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, sdk.Docker, res.Type)
	assert.Equal(t, u.Username, res.Author.Username)
	assert.Equal(t, pattern.Model.Cmd, res.ModelDocker.Cmd)

	resClear, err := workermodel.LoadByIDWithClearPassword(db, res.ID)
	require.NoError(t, err)
	assert.Equal(t, "12345678", resClear.ModelDocker.Password, "password should be preserved")

	// Test change group and name
	cpy := *res
	cpy.Name = model2Name
	res, err = workermodel.Update(context.TODO(), db, res, cpy)
	require.NoError(t, err)
	assert.Equal(t, model2Name, res.Name)

	cpy = *res
	cpy.GroupID = g2.ID
	res, err = workermodel.Update(context.TODO(), db, res, cpy)
	require.Error(t, err)

}

// create a worker model aaa
// a pipeline use worker model aaa-foo
// rename worker model to aaa-bar
// the pipeline should keep the name aaa-foo
func TestUpdateModelInPipeline(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	g1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	u, _ := assets.InsertLambdaUser(t, db)

	pattern := sdk.ModelPattern{
		Name: sdk.RandomString(10),
		Type: sdk.Docker,
		Model: sdk.ModelCmds{
			Cmd:   "pattern cmd",
			Shell: "pattern shell",
		},
	}
	require.NoError(t, workermodel.InsertPattern(db, &pattern))

	model1Name := sdk.RandomString(10)
	model1, err := workermodel.Create(context.TODO(), db, sdk.Model{
		Type:        sdk.Docker,
		Name:        model1Name,
		Group:       g1,
		GroupID:     g1.ID,
		ModelDocker: sdk.ModelDocker{},
	}, u)
	require.NoError(t, err)

	model1NameFoo := model1Name + "-foo"
	model1Foo, err := workermodel.Create(context.TODO(), db, sdk.Model{
		Type:        sdk.Docker,
		Name:        model1NameFoo,
		Group:       g1,
		GroupID:     g1.ID,
		ModelDocker: sdk.ModelDocker{},
	}, u)
	require.NoError(t, err)

	projectKey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, projectKey, projectKey)

	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g1.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionReadWriteExecute,
	}))

	pip1 := sdk.Pipeline{ProjectID: proj.ID, ProjectKey: proj.Key, Name: sdk.RandomString(10)}
	test.NoError(t, pipeline.InsertPipeline(db, &pip1))
	job1 := sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
			Requirements: []sdk.Requirement{{
				Type:  sdk.ModelRequirement,
				Name:  fmt.Sprintf("%s/%s-foo --privileged", g1.Name, model1.Name),
				Value: fmt.Sprintf("%s/%s-foo --privileged", g1.Name, model1.Name),
			}},
		},
	}
	test.NoError(t, pipeline.InsertJob(db, &job1, 0, &pip1))

	model1FooLoad, err := workermodel.LoadByNameAndGroupID(db, model1Foo.Name, g1.ID)
	require.NoError(t, err)

	pips, err := pipeline.LoadByWorkerModel(context.TODO(), db, model1FooLoad)
	assert.NoError(t, err)
	require.Equal(t, 1, len(pips))

	model1Load, err := workermodel.LoadByIDWithClearPassword(db, model1.ID)
	require.NoError(t, err)

	pips, err = pipeline.LoadByWorkerModel(context.TODO(), db, model1Load)
	assert.NoError(t, err)
	require.Equal(t, 0, len(pips))

	// Test rename worker model
	res, err := workermodel.Update(context.TODO(), db, model1Load, sdk.Model{
		Type:        sdk.Docker,
		Name:        model1Name + "-bar",
		PatternName: pattern.Name,
		GroupID:     g1.ID,
		ModelDocker: sdk.ModelDocker{
			Private:  true,
			Password: sdk.PasswordPlaceholder,
		},
	})
	require.NoError(t, err)

	model2Load, err := workermodel.LoadByIDWithClearPassword(db, res.ID)
	require.NoError(t, err)

	pips, err = pipeline.LoadByWorkerModel(context.TODO(), db, model1Load)
	assert.NoError(t, err)
	require.Equal(t, 0, len(pips))

	pips, err = pipeline.LoadByWorkerModel(context.TODO(), db, model2Load)
	assert.NoError(t, err)
	require.Equal(t, 0, len(pips))

	pips, err = pipeline.LoadByWorkerModel(context.TODO(), db, model1FooLoad)
	assert.NoError(t, err)
	require.Equal(t, 1, len(pips))

}

func TestUpdateModelInPipelineSimple(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	g1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	u, _ := assets.InsertLambdaUser(t, db)

	pattern := sdk.ModelPattern{
		Name: sdk.RandomString(10),
		Type: sdk.Docker,
		Model: sdk.ModelCmds{
			Cmd:   "pattern cmd",
			Shell: "pattern shell",
		},
	}
	require.NoError(t, workermodel.InsertPattern(db, &pattern))

	model1Name := sdk.RandomString(10)
	model1, err := workermodel.Create(context.TODO(), db, sdk.Model{
		Type:        sdk.Docker,
		Name:        model1Name,
		Group:       g1,
		GroupID:     g1.ID,
		ModelDocker: sdk.ModelDocker{},
	}, u)
	require.NoError(t, err)

	projectKey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, projectKey, projectKey)

	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g1.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionReadWriteExecute,
	}))

	pip1 := sdk.Pipeline{ProjectID: proj.ID, ProjectKey: proj.Key, Name: sdk.RandomString(10)}
	test.NoError(t, pipeline.InsertPipeline(db, &pip1))
	job1 := sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
			Requirements: []sdk.Requirement{{
				Type:  sdk.ModelRequirement,
				Name:  fmt.Sprintf("%s/%s --privileged", g1.Name, model1.Name),
				Value: fmt.Sprintf("%s/%s --privileged", g1.Name, model1.Name),
			}},
		},
	}
	test.NoError(t, pipeline.InsertJob(db, &job1, 0, &pip1))

	model1Load, err := workermodel.LoadByIDWithClearPassword(db, model1.ID)
	require.NoError(t, err)

	pips, err := pipeline.LoadByWorkerModel(context.TODO(), db, model1Load)
	assert.NoError(t, err)
	require.Equal(t, 1, len(pips))

	model1NameFoo := model1Name + "-foo"
	res, err := workermodel.Update(context.TODO(), db, model1Load, sdk.Model{
		Type:        sdk.Docker,
		Name:        model1NameFoo,
		PatternName: pattern.Name,
		GroupID:     g1.ID,
		ModelDocker: sdk.ModelDocker{
			Private:  true,
			Password: sdk.PasswordPlaceholder,
		},
	})
	require.NoError(t, err)

	model1FooLoad, err := workermodel.LoadByIDWithClearPassword(db, res.ID)
	require.NoError(t, err)

	pips, err = pipeline.LoadByWorkerModel(context.TODO(), db, model1FooLoad)
	assert.NoError(t, err)
	require.Equal(t, 1, len(pips))

	pip, err := pipeline.LoadPipelineByID(context.TODO(), db, pips[0].ID, true)
	require.Equal(t, fmt.Sprintf("%s/%s-foo --privileged", g1.Name, model1.Name), pip.Stages[0].Jobs[0].Action.Requirements[0].Value)
}

func TestCopyModelTypeData(t *testing.T) {
	old := sdk.Model{
		Type: sdk.Docker,
		ModelDocker: sdk.ModelDocker{
			Cmd:   "my cmd",
			Shell: "my shell",
			Envs: map[string]string{
				"one": "value",
			},
		},
	}
	data := sdk.Model{}

	// model type cannot be different
	assert.Error(t, workermodel.CopyModelTypeData(&old, &data))

	data.Type = sdk.Docker
	assert.NoError(t, workermodel.CopyModelTypeData(&old, &data))
	assert.Equal(t, old.ModelDocker, data.ModelDocker)
}

func TestCopyModelTypeData_OldRestricted(t *testing.T) {
	old := sdk.Model{
		Type:       sdk.Docker,
		Restricted: true,
	}

	assert.Error(t, workermodel.CopyModelTypeData(&old, &sdk.Model{
		Type:        sdk.Docker,
		Restricted:  false,
		PatternName: "",
	}), "an error should occured as the is no pattern given and we can't reuse custom commands from old not restricted model")

	assert.NoError(t, workermodel.CopyModelTypeData(&old, &sdk.Model{
		Type:        sdk.Docker,
		Restricted:  false,
		PatternName: "my-pattern",
	}))
}
