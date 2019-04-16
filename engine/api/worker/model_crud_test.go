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

	//	u2, _ := assets.InsertAdminUser(db)

	pattern := sdk.ModelPattern{
		Name:  sdk.RandomString(10),
		Type:  sdk.Docker,
		Model: sdk.ModelCmds{},
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
			Name: "no cds admin user can't set provision on no restricted model and should",
			User: u1,
			Data: sdk.Model{
				Type:        sdk.Docker,
				Name:        sdk.RandomString(10),
				GroupID:     g2.ID,
				Provision:   5,
				PatternName: pattern.Name,
			},
			Result: sdk.Model{
				Provision: 0,
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
			}
		})
	}
}
