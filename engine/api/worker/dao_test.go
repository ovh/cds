package worker_test

import (
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/accesstoken"
	"github.com/ovh/cds/engine/api/services"

	"github.com/ovh/cds/engine/api/workermodel"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/jws"
	"github.com/stretchr/testify/assert"
)

func insertGroup(t *testing.T, db gorp.SqlExecutor) *sdk.Group {
	g := &sdk.Group{
		Name: sdk.RandomString(10),
	}

	g1, _ := group.LoadGroup(db, g.Name)
	if g1 != nil {
		models, _ := workermodel.LoadAllByGroups(db, []int64{g.ID}, nil)
		for _, m := range models {
			workermodel.Delete(db, m.ID)
		}

		if err := group.DeleteGroupAndDependencies(db, g1); err != nil {
			t.Logf("unable to delete group: %v", err)
		}
	}

	if err := group.InsertGroup(db, g); err != nil {
		t.Fatalf("Unable to create group %s", err)
	}

	return g
}

func insertWorkerModel(t *testing.T, db gorp.SqlExecutor, name string, groupID int64) *sdk.Model {
	m := sdk.Model{
		Name: name,
		Type: sdk.Docker,
		ModelDocker: sdk.ModelDocker{
			Image: "foo/bar:3.4",
		},
		GroupID: groupID,
		RegisteredCapabilities: sdk.RequirementList{
			{
				Name:  "capa_1",
				Type:  sdk.BinaryRequirement,
				Value: "capa_1",
			},
		},
		UserLastModified: time.Now(),
	}

	if err := workermodel.Insert(db, &m); err != nil {
		t.Fatalf("Cannot insert worker model: %s", err)
	}

	assert.NotEqual(t, 0, m.ID)
	return &m
}

func insertHatchery(t *testing.T, db gorp.SqlExecutor, grp sdk.Group) *sdk.Service {
	usr1, _ := assets.InsertLambdaUser(db)

	exp := time.Now().Add(5 * time.Minute)
	token, signedToken, err := accesstoken.New(*usr1, []sdk.Group{grp}, []string{sdk.AccessTokenScopeHatchery}, "cds_test", "cds test", exp)
	test.NoError(t, err)

	test.NoError(t, accesstoken.Insert(db, &token))

	privateKey, err := jws.NewRandomRSAKey()
	test.NoError(t, err)
	publicKey, err := jws.ExportPublicKey(privateKey)
	test.NoError(t, err)

	var srv = sdk.Service{
		CanonicalService: sdk.CanonicalService{
			Name:       sdk.RandomString(10),
			Type:       services.TypeHatchery,
			PublicKey:  publicKey,
			Maintainer: *usr1,
			TokenID:    token.ID,
		},
		ClearJWT: signedToken,
	}

	test.NoError(t, services.Insert(db, &srv))

	return &srv
}

func TestInsert(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	workers, err := worker.LoadAll(db)
	test.NoError(t, err)
	for _, w := range workers {
		worker.DeleteWorker(db, w.ID)
	}

	g := insertGroup(t, db)
	h := insertHatchery(t, db, *g)
	m := insertWorkerModel(t, db, sdk.RandomString(5), g.ID)

	w := &sdk.Worker{
		ID:         "foofoo",
		Name:       "foo.bar.io",
		ModelID:    m.ID,
		HatcheryID: h.ID,
	}

	if err := worker.Insert(db, w); err != nil {
		t.Fatalf("Cannot insert worker %+v: %v", w, err)
	}

	wks, err := worker.LoadByHatcheryID(db, h.ID)
	test.NoError(t, err)
	assert.Len(t, wks, 1)

	if len(wks) == 1 {
		assert.Equal(t, "foofoo", wks[0].ID)
	}

}

func TestLoadWorkers(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	workers, errl := worker.LoadAll(db)
	test.NoError(t, errl)
	for _, w := range workers {
		worker.DeleteWorker(db, w.ID)
	}

	w := &sdk.Worker{ID: "foo1", Name: "aa.bar.io"}
	if err := worker.Insert(db, w); err != nil {
		t.Fatalf("Cannot insert worker: %s", err)
	}
	w = &sdk.Worker{ID: "foo2", Name: "zz.bar.io"}
	if err := worker.Insert(db, w); err != nil {
		t.Fatalf("Cannot insert worker: %s", err)
	}
	w = &sdk.Worker{ID: "foo3", Name: "bb.bar.io"}
	if err := worker.Insert(db, w); err != nil {
		t.Fatalf("Cannot insert worker: %s", err)
	}
	w = &sdk.Worker{ID: "foo4", Name: "aa.car.io"}
	if err := worker.Insert(db, w); err != nil {
		t.Fatalf("Cannot insert worker: %s", err)
	}

	var errlw error
	workers, errlw = worker.LoadAll(db)
	if errlw != nil {
		t.Fatalf("Cannot load workers: %s", errlw)
	}

	if len(workers) != 4 {
		t.Fatalf("Expected 4 workers, got %d", 4)
	}

	order := []string{
		"aa.bar.io",
		"aa.car.io",
		"bb.bar.io",
		"zz.bar.io",
	}
	for i := range order {
		if order[i] != workers[i].Name {
			t.Fatalf("Expected %s, got %s\n", order[i], workers[i].Name)
		}
	}
}
