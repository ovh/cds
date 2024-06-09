package worker_test

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
)

func TestReleaseAllFromHatchery(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitializeDB)

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

	w1Consumer, err := authentication.NewConsumerWorker(context.TODO(), db, "worker-1", h1Consumer)
	require.NoError(t, err)
	w2Consumer, err := authentication.NewConsumerWorker(context.TODO(), db, "worker-2", h2Consumer)
	require.NoError(t, err)

	require.NoError(t, worker.Insert(context.TODO(), db, &sdk.Worker{
		ID:           sdk.UUID(),
		Name:         "worker-1",
		ModelID:      &m.ID,
		HatcheryID:   &h1.ID,
		HatcheryName: h1.Name,
		ConsumerID:   w1Consumer.ID,
		Status:       sdk.StatusWaiting,
	}))
	require.NoError(t, worker.Insert(context.TODO(), db, &sdk.Worker{
		ID:           sdk.UUID(),
		Name:         "worker-2",
		ModelID:      &m.ID,
		HatcheryID:   &h2.ID,
		HatcheryName: h2.Name,
		ConsumerID:   w2Consumer.ID,
		Status:       sdk.StatusWaiting,
	}))

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

func TestReAttachAllToHatchery(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitializeDB)

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

	w1Consumer, err := authentication.NewConsumerWorker(context.TODO(), db, "worker-1", h1Consumer)
	require.NoError(t, err)
	w2Consumer, err := authentication.NewConsumerWorker(context.TODO(), db, "worker-2", h2Consumer)
	require.NoError(t, err)

	require.NoError(t, worker.Insert(context.TODO(), db, &sdk.Worker{
		ID:           sdk.UUID(),
		Name:         "worker-1",
		ModelID:      &m.ID,
		HatcheryID:   &h1.ID,
		HatcheryName: h1.Name,
		ConsumerID:   w1Consumer.ID,
		Status:       sdk.StatusWaiting,
	}))
	require.NoError(t, worker.Insert(context.TODO(), db, &sdk.Worker{
		ID:           sdk.UUID(),
		Name:         "worker-2",
		ModelID:      &m.ID,
		HatcheryName: h2.Name,
		ConsumerID:   w2Consumer.ID,
		Status:       sdk.StatusWaiting,
	}))

	require.NoError(t, worker.ReAttachAllToHatchery(context.TODO(), db, *h2))

	workers, err = worker.LoadAll(context.TODO(), db)
	require.NoError(t, err)
	require.Len(t, workers, 2)
	sort.Slice(workers, func(i, j int) bool { return workers[i].Name < workers[i].Name })
	require.Equal(t, "worker-1", workers[0].Name)
	require.NotNil(t, workers[0].HatcheryID)
	require.Equal(t, h1.ID, *workers[0].HatcheryID)
	require.Equal(t, "worker-2", workers[1].Name)
	require.NotNil(t, workers[1].HatcheryID)
	require.Equal(t, h2.ID, *workers[1].HatcheryID)
}
