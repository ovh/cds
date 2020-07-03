package worker_test

import (
	"context"
	"runtime"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
)

func TestDAO(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	workers, err := worker.LoadAll(context.TODO(), db)
	require.NoError(t, err)
	for _, w := range workers {
		require.NoError(t, worker.Delete(db, w.ID))
	}

	g := assets.InsertGroup(t, db)
	hSrv, _, hCons, _ := assets.InsertHatchery(t, db, *g)
	m := assets.InsertWorkerModel(t, db, sdk.RandomString(5), g.ID)

	w := &sdk.Worker{
		ID:           "foofoo",
		Name:         "foo.bar.io",
		ModelID:      &m.ID,
		HatcheryID:   &hSrv.ID,
		HatcheryName: hSrv.Name,
		ConsumerID:   hCons.ID,
		Status:       sdk.StatusWaiting,
	}

	if err := worker.Insert(context.TODO(), db, w); err != nil {
		t.Fatalf("Cannot insert worker %+v: %v", w, err)
	}

	wks, err := worker.LoadAllByHatcheryID(context.TODO(), db, hSrv.ID)
	require.NoError(t, err)
	require.Len(t, wks, 1)

	if len(wks) == 1 {
		require.Equal(t, "foofoo", wks[0].ID)
	}

	wk, err := worker.LoadByID(context.TODO(), db, "foofoo")
	require.NoError(t, err)
	require.NotNil(t, wk)
	if wk != nil {
		require.Equal(t, "foofoo", wk.ID)
	}

	require.NoError(t, worker.SetStatus(context.TODO(), db, wk.ID, sdk.StatusBuilding))
	require.NoError(t, worker.RefreshWorker(db, wk.ID))
}

func TestDeadWorkers(t *testing.T) {
	db, _ := test.SetupPG(t)

	require.NoError(t, worker.DisableDeadWorkers(context.TODO(), db))
	require.NoError(t, worker.DeleteDeadWorkers(context.TODO(), db))
}

func TestRegister(t *testing.T) {
	db, store := test.SetupPG(t, bootstrap.InitiliazeDB)

	g := assets.InsertGroup(t, db)
	h, _, c, _ := assets.InsertHatchery(t, db, *g)
	m := assets.InsertWorkerModel(t, db, sdk.RandomString(5), g.ID)

	w, err := worker.RegisterWorker(context.TODO(), db, store, hatchery.SpawnArguments{
		HatcheryName: h.Name,
		Model:        m,
		RegisterOnly: true,
		WorkerName:   sdk.RandomString(10),
	}, *h, c, sdk.WorkerRegistrationForm{
		Arch:               runtime.GOARCH,
		OS:                 runtime.GOOS,
		BinaryCapabilities: []string{"bash"},
	})
	require.NoError(t, err)
	require.NotNil(t, w)

	wk, err := worker.LoadByConsumerID(context.TODO(), db, w.ConsumerID)
	require.NoError(t, err)
	require.Equal(t, w.ID, wk.ID)
	require.Equal(t, w.ModelID, wk.ModelID)
	require.Equal(t, w.HatcheryID, wk.HatcheryID)

	// try to register a worker for a job, without a JobID
	w2, err := worker.RegisterWorker(context.TODO(), db, store, hatchery.SpawnArguments{
		HatcheryName: h.Name,
		Model:        m,
		WorkerName:   sdk.RandomString(10),
	}, *h, c, sdk.WorkerRegistrationForm{
		Arch:               runtime.GOARCH,
		OS:                 runtime.GOOS,
		BinaryCapabilities: []string{"bash"},
	})

	require.Error(t, err)
	require.Nil(t, w2)
}

func TestReleaseAllFromHatchery(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	// Remove all existing workers in database
	workers, err := worker.LoadAll(context.TODO(), db)
	require.NoError(t, err)
	for _, w := range workers {
		require.NoError(t, worker.Delete(db, w.ID))
	}

	g := assets.InsertGroup(t, db)
	m := assets.InsertWorkerModel(t, db, sdk.RandomString(5), g.ID)

	h1, _, h1Consumer, _ := assets.InsertHatchery(t, db, *g)
	h2, _, h2Consumer, _ := assets.InsertHatchery(t, db, *g)

	require.NoError(t, worker.Insert(context.TODO(), db, &sdk.Worker{
		ID:           sdk.UUID(),
		Name:         "worker-1",
		ModelID:      &m.ID,
		HatcheryID:   &h1.ID,
		HatcheryName: h1.Name,
		ConsumerID:   h1Consumer.ID,
		Status:       sdk.StatusWaiting,
	}))
	require.NoError(t, worker.Insert(context.TODO(), db, &sdk.Worker{
		ID:           sdk.UUID(),
		Name:         "worker-2",
		ModelID:      &m.ID,
		HatcheryID:   &h2.ID,
		HatcheryName: h2.Name,
		ConsumerID:   h2Consumer.ID,
		Status:       sdk.StatusWaiting,
	}))

	workers, err = worker.LoadAll(context.TODO(), db)
	require.NoError(t, err)
	require.Len(t, workers, 2)

	require.NoError(t, worker.ReleaseAllFromHatchery(db, h1.ID))

	workers, err = worker.LoadAll(context.TODO(), db)
	require.NoError(t, err)
	require.Len(t, workers, 2)
	sort.Slice(workers, func(i, j int) bool { return workers[i].Name < workers[i].Name })
	assert.Equal(t, "worker-1", workers[0].Name)
	assert.Nil(t, workers[0].HatcheryID)
	assert.Equal(t, "worker-2", workers[1].Name)
	assert.NotNil(t, workers[1].HatcheryID)
}
