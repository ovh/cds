package worker_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
)

func TestCreateModel(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	g1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	u1, _ := assets.InsertLambdaUser(db, g2)
	assert.NoError(t, group.SetUserGroupAdmin(db, g2.ID, u1.ID))
	g2.Admins = append(g2.Admins, *u1)
	u1.Groups[0].Admins = g2.Admins

	u2, _ := assets.InsertAdminUser(db)

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
	test.NoError(t, worker.InsertWorkerModelPattern(db, &pattern))

	tests := []struct {
		Name   string
		User   *sdk.User
		Data   sdk.Model
		Result sdk.Model
		Error  bool
	}{
		{
			Name:  "given group id should be valid",
			User:  u1,
			Data:  sdk.Model{GroupID: 0},
			Error: true,
		},
		{
			Name:  "user should group admin",
			User:  u1,
			Data:  sdk.Model{GroupID: g1.ID},
			Error: true,
		},
		{
			Name: "no cds admin user should give a pattern name",
			User: u1,
			Data: sdk.Model{
				GroupID: g2.ID,
			},
			Error: true,
		},
		{
			Name: "no cds admin user can't set provision on no restricted model",
			User: u1,
			Data: sdk.Model{
				Type:        sdk.Docker,
				Name:        sdk.RandomString(10),
				GroupID:     g2.ID,
				Provision:   5,
				PatternName: pattern.Name,
				ModelDocker: sdk.ModelDocker{
					Envs: map[string]string{
						"ignored": "value",
					},
				},
			},
			Result: sdk.Model{
				Provision: 0,
				ModelDocker: sdk.ModelDocker{
					Cmd:   pattern.Model.Cmd,
					Shell: pattern.Model.Shell,
					Envs:  worker.MergeModelEnvsWithDefaultEnvs(pattern.Model.Envs),
				},
			},
		},
		{
			Name: "set provision on restricted model",
			User: u1,
			Data: sdk.Model{
				Type:       sdk.Docker,
				Name:       sdk.RandomString(10),
				GroupID:    g2.ID,
				Provision:  5,
				Restricted: true,
				ModelDocker: sdk.ModelDocker{
					Cmd:   "my custom cmd",
					Shell: "my custom shell",
					Envs: map[string]string{
						"custom": "value",
					},
				},
			},
			Result: sdk.Model{
				Provision: 5,
				ModelDocker: sdk.ModelDocker{
					Cmd:   "my custom cmd",
					Shell: "my custom shell",
					Envs: worker.MergeModelEnvsWithDefaultEnvs(map[string]string{
						"custom": "value",
					}),
				},
			},
		},
		{
			Name: "cds admin user can set provision on no restricted model",
			User: u2,
			Data: sdk.Model{
				Type:      sdk.Docker,
				Name:      sdk.RandomString(10),
				GroupID:   g1.ID,
				Provision: 5,
			},
			Result: sdk.Model{
				Provision: 5,
				ModelDocker: sdk.ModelDocker{
					Envs: worker.MergeModelEnvsWithDefaultEnvs(nil),
				},
			},
		},
		{
			Name: "cds admin user can set provision on no restricted model",
			User: u2,
			Data: sdk.Model{
				Type:      sdk.Docker,
				Name:      sdk.RandomString(10),
				GroupID:   g1.ID,
				Provision: 5,
			},
			Result: sdk.Model{
				Provision: 5,
				ModelDocker: sdk.ModelDocker{
					Envs: worker.MergeModelEnvsWithDefaultEnvs(nil),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			res, err := worker.CreateModel(db, test.User, test.Data)
			if test.Error {
				assert.Error(t, err)
			} else {
				if !assert.NoError(t, err) {
					t.FailNow()
				}

				// check model data
				assert.Equal(t, test.Result.Provision, res.Provision)
				assert.Equal(t, test.Result.ModelDocker, res.ModelDocker)
			}
		})
	}
}

func TestUpdateModel(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	g1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	u1, _ := assets.InsertLambdaUser(db, g2)
	assert.NoError(t, group.SetUserGroupAdmin(db, g2.ID, u1.ID))
	g2.Admins = append(g2.Admins, *u1)
	u1.Groups[0].Admins = g2.Admins

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
	test.NoError(t, worker.InsertWorkerModelPattern(db, &pattern))

	tests := []struct {
		Name   string
		User   *sdk.User
		Old    *sdk.Model
		Data   sdk.Model
		Result sdk.Model
		Error  bool
	}{
		{
			Name:  "given group id should be valid",
			User:  u1,
			Data:  sdk.Model{GroupID: 0},
			Error: true,
		},
		{
			Name: "change group, user should be admin of target group",
			User: u1,
			Old: &sdk.Model{
				Type:        sdk.Docker,
				Name:        sdk.RandomString(10),
				GroupID:     g2.ID,
				PatternName: pattern.Name,
			},
			Data:  sdk.Model{GroupID: g1.ID},
			Error: true,
		},
		{
			Name: "no cds admin user can't set provision on no restricted model",
			User: u1,
			Old: &sdk.Model{
				Type:        sdk.Docker,
				Name:        sdk.RandomString(10),
				GroupID:     g2.ID,
				PatternName: pattern.Name,
			},
			Data: sdk.Model{
				Type:        sdk.Docker,
				Name:        sdk.RandomString(10),
				GroupID:     g2.ID,
				Provision:   5,
				PatternName: pattern.Name,
				ModelDocker: sdk.ModelDocker{
					Envs: map[string]string{
						"ignored": "value",
					},
				},
			},
			Result: sdk.Model{
				Provision: 0,
				ModelDocker: sdk.ModelDocker{
					Cmd:   pattern.Model.Cmd,
					Shell: pattern.Model.Shell,
					Envs:  worker.MergeModelEnvsWithDefaultEnvs(pattern.Model.Envs),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			if test.Old != nil {
				var err error
				test.Old, err = worker.CreateModel(db, test.User, *test.Old)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				test.Old, err = worker.LoadWorkerModelByID(db, test.Old.ID)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
			}

			res, err := worker.UpdateModel(db, test.User, test.Old, test.Data)
			if test.Error {
				assert.Error(t, err)
			} else {
				if !assert.NoError(t, err) {
					t.FailNow()
				}

				// check model data
				assert.Equal(t, test.Result.Provision, res.Provision)
				assert.Equal(t, test.Result.ModelDocker, res.ModelDocker)
			}
		})
	}
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
	assert.Error(t, worker.CopyModelTypeData(&sdk.User{}, &old, &data))

	data.Type = sdk.Docker
	assert.NoError(t, worker.CopyModelTypeData(&sdk.User{}, &old, &data))
	assert.Equal(t, old.ModelDocker, data.ModelDocker)
}
