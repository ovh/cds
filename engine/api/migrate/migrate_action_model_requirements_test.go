package migrate

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/sdk"
)

func TestActionModelRequirements_WithExistingRequirements(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	g := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	p := sdk.ModelPattern{
		Name: sdk.RandomString(10),
		Type: sdk.Docker,
	}
	test.NoError(t, workermodel.InsertPattern(db, &p))

	m, err := workermodel.Create(context.TODO(), db, sdk.Model{
		Type:        sdk.Docker,
		Name:        sdk.RandomString(10),
		GroupID:     g.ID,
		PatternName: p.Name,
	}, sdk.AuthentifiedUser{})
	test.NoError(t, err)

	a := sdk.Action{
		GroupID: &g.ID,
		Type:    sdk.DefaultAction,
		Name:    sdk.RandomString(10),
		Requirements: []sdk.Requirement{
			{
				Name:  fmt.Sprintf("%s", m.Name),
				Type:  sdk.ModelRequirement,
				Value: fmt.Sprintf("%s", m.Name),
			},
			{
				Name:  fmt.Sprintf("%s --privileged", m.Name),
				Type:  sdk.ModelRequirement,
				Value: fmt.Sprintf("%s --privileged", m.Name),
			},
		},
	}
	test.NoError(t, action.Insert(db, &a))

	test.NoError(t, ActionModelRequirements(context.TODO(), nil, func() *gorp.DbMap { return db }))

	aUpdated, err := action.LoadByID(context.TODO(), db, a.ID, action.LoadOptions.WithRequirements)
	test.NoError(t, err)

	test.Equal(t, 2, len(aUpdated.Requirements))
	test.Equal(t, fmt.Sprintf("%s/%s", g.Name, m.Name), aUpdated.Requirements[0].Name)
	test.Equal(t, fmt.Sprintf("%s/%s", g.Name, m.Name), aUpdated.Requirements[0].Value)
	test.Equal(t, fmt.Sprintf("%s/%s --privileged", g.Name, m.Name), aUpdated.Requirements[1].Name)
	test.Equal(t, fmt.Sprintf("%s/%s --privileged", g.Name, m.Name), aUpdated.Requirements[1].Value)
}

func TestActionModelRequirements_WithoutExistingRequirements(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	p := sdk.ModelPattern{
		Name: sdk.RandomString(10),
		Type: sdk.Docker,
	}
	test.NoError(t, workermodel.InsertPattern(db, &p))

	m := sdk.Model{
		Name: sdk.RandomString(10),
		Group: &sdk.Group{
			Name: sdk.RandomString(10),
		},
	}

	test.NoError(t, migrateActionRequirementForModel(context.TODO(), db, m))
}

func TestActionModelRequirements_WithLockedExistingRequirements(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	g := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	p := sdk.ModelPattern{
		Name: sdk.RandomString(10),
		Type: sdk.Docker,
	}
	test.NoError(t, workermodel.InsertPattern(db, &p))

	m, err := workermodel.Create(context.TODO(), db, sdk.Model{
		Type:        sdk.Docker,
		Name:        sdk.RandomString(10),
		GroupID:     g.ID,
		PatternName: p.Name,
	}, sdk.AuthentifiedUser{})
	test.NoError(t, err)

	a := sdk.Action{
		GroupID: &g.ID,
		Type:    sdk.DefaultAction,
		Name:    sdk.RandomString(10),
		Requirements: []sdk.Requirement{
			{
				Name:  fmt.Sprintf("%s", m.Name),
				Type:  sdk.ModelRequirement,
				Value: fmt.Sprintf("%s", m.Name),
			},
		},
	}
	test.NoError(t, action.Insert(db, &a))

	tx, err := db.Begin()
	test.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	rs, err := action.GetRequirementsTypeModelAndValueStartByWithLock(context.TODO(), tx, m.Name)
	test.NoError(t, err)
	test.Equal(t, 1, len(rs))

	rs, err = action.GetRequirementsTypeModelAndValueStartByWithLock(context.TODO(), db, m.Name)
	test.NoError(t, err)
	test.Equal(t, 0, len(rs))
}
