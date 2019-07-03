package worker_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
)

func TestDAO(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	workers, err := worker.LoadAll(context.TODO(), db)
	test.NoError(t, err)
	for _, w := range workers {
		worker.Delete(db, w.ID)
	}

	g := assets.InsertGroup(t, db)
	hSrv, _, hCons, _ := assets.InsertHatchery(t, db, *g)
	m := assets.InsertWorkerModel(t, db, sdk.RandomString(5), g.ID)

	w := &sdk.Worker{
		ID:         "foofoo",
		Name:       "foo.bar.io",
		ModelID:    m.ID,
		HatcheryID: hSrv.ID,
		ConsumerID: hCons.ID,
		Status:     sdk.StatusWaiting,
	}

	if err := worker.Insert(db, w); err != nil {
		t.Fatalf("Cannot insert worker %+v: %v", w, err)
	}

	wks, err := worker.LoadByHatcheryID(context.TODO(), db, hSrv.ID)
	test.NoError(t, err)
	assert.Len(t, wks, 1)

	if len(wks) == 1 {
		assert.Equal(t, "foofoo", wks[0].ID)
	}

	wk, err := worker.LoadByID(context.TODO(), db, "foofoo")
	test.NoError(t, err)
	assert.NotNil(t, wk)
	if wk != nil {
		assert.Equal(t, "foofoo", wk.ID)
	}

	test.NoError(t, worker.SetStatus(db, wk.ID, sdk.StatusBuilding))
	test.NoError(t, worker.RefreshWorker(db, wk.ID))
}

func TestDeadWorkers(t *testing.T) {
	db, _, end := test.SetupPG(t)
	defer end()
	test.NoError(t, worker.DisableDeadWorkers(context.TODO(), db))
	test.NoError(t, worker.DeleteDeadWorkers(context.TODO(), db))
}

func TestRegister(t *testing.T) {
	db, store, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	g := assets.InsertGroup(t, db)
	h, _, c, _ := assets.InsertHatchery(t, db, *g)
	m := assets.InsertWorkerModel(t, db, sdk.RandomString(5), g.ID)

	w, err := worker.RegisterWorker(db, store, hatchery.SpawnArguments{
		HatcheryName: h.Name,
		Model:        *m,
		RegisterOnly: true,
		WorkerName:   sdk.RandomString(10),
	}, h.ID, c, sdk.WorkerRegistrationForm{
		Arch:               runtime.GOARCH,
		OS:                 runtime.GOOS,
		BinaryCapabilities: []string{"bash"},
	})

	test.NoError(t, err)
	assert.NotNil(t, w)

	wk, err := worker.LoadByAuthConsumerID(context.TODO(), db, w.ConsumerID)
	test.NoError(t, err)
	assert.Equal(t, w.ID, wk.ID)
	assert.Equal(t, w.ModelID, wk.ModelID)
	assert.Equal(t, w.HatcheryID, wk.HatcheryID)
}
