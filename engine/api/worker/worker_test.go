package worker_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication"
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

	require.NoError(t, worker.DisableDeadWorkers(context.TODO(), db.DbMap))
	require.NoError(t, worker.DeleteDeadWorkers(context.TODO(), db.DbMap))
}

func TestRegister(t *testing.T) {
	db, store := test.SetupPG(t, bootstrap.InitiliazeDB)

	g := assets.InsertGroup(t, db)
	h, _, hatcheryConsumer, _ := assets.InsertHatchery(t, db, *g)
	workerConsumer, err := authentication.NewConsumerWorker(context.TODO(), db, sdk.RandomString(10), h, hatcheryConsumer)
	require.NoError(t, err)
	m := assets.InsertWorkerModel(t, db, sdk.RandomString(5), g.ID)

	spawnArgs := hatchery.SpawnArgumentsJWT{
		HatcheryName: h.Name,
		RegisterOnly: true,
		WorkerName:   sdk.RandomString(10),
	}
	spawnArgs.Model.ID = m.ID
	w, err := worker.RegisterWorker(context.TODO(), db, store, spawnArgs, *h, hatcheryConsumer, workerConsumer, sdk.WorkerRegistrationForm{
		Arch:               runtime.GOARCH,
		OS:                 runtime.GOOS,
		BinaryCapabilities: []string{"bash"},
	}, nil)
	require.NoError(t, err)
	require.NotNil(t, w)

	wk, err := worker.LoadByConsumerID(context.TODO(), db, w.ConsumerID)
	require.NoError(t, err)
	require.Equal(t, w.ID, wk.ID)
	require.Equal(t, w.ModelID, wk.ModelID)
	require.Equal(t, w.HatcheryID, wk.HatcheryID)

	// try to register a worker for a job, without a JobID
	spawnArgs = hatchery.SpawnArgumentsJWT{
		HatcheryName: h.Name,
		WorkerName:   sdk.RandomString(10),
	}
	spawnArgs.Model.ID = m.ID
	w2, err := worker.RegisterWorker(context.TODO(), db, store, spawnArgs, *h, hatcheryConsumer, workerConsumer, sdk.WorkerRegistrationForm{
		Arch:               runtime.GOARCH,
		OS:                 runtime.GOOS,
		BinaryCapabilities: []string{"bash"},
	}, nil)

	require.Error(t, err)
	require.Nil(t, w2)
}
