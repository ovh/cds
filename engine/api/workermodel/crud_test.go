package workermodel_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/bootstrap"
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
