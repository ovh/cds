package worker_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/stretchr/testify/assert"
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
	h, _ := assets.InsertHatchery(t, db, *g)
	m := assets.InsertWorkerModel(t, db, sdk.RandomString(5), g.ID)

	w := &sdk.Worker{
		ID:         "foofoo",
		Name:       "foo.bar.io",
		ModelID:    m.ID,
		HatcheryID: h.ID,
		Status:     sdk.StatusWaiting,
	}

	if err := worker.Insert(db, w); err != nil {
		t.Fatalf("Cannot insert worker %+v: %v", w, err)
	}

	wks, err := worker.LoadByHatcheryID(context.TODO(), db, h.ID)
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

func TestAuthentication(t *testing.T) {
	/*db, _, end := test.SetupPG(t)
	defer end()

	g := assets.InsertGroup(t, db)
	h, pk := assets.InsertHatchery(t, db, *g)
	u, _ := assets.InsertLambdaUser(db, g)
	m := assets.InsertWorkerModel(t, db, sdk.RandomString(5), g.ID)

	w := &sdk.Worker{
		ID:         sdk.RandomString(10),
		Name:       sdk.RandomString(10),
		ModelID:    m.ID,
		HatcheryID: h.ID,
		Status:     sdk.StatusWaiting,
	}

	if err := worker.Insert(db, w); err != nil {
		t.Fatalf("Cannot insert worker %+v: %v", w, err)
	}

	token, jwt, err := hatchery.NewWorkerToken(h.Name, pk, *u, time.Now().Add(time.Minute), hatchery.SpawnArguments{
		HatcheryName: h.Name,
		Model:        *m,
		WorkerName:   sdk.RandomString(10),
	})
	test.NoError(t, err)
	assert.NotNil(t, token)

	test.NoError(t, authentication.Insert(db, &token))

	w.AccessTokenID = &token.ID
	if err := worker.Update(db, w); err != nil {
		t.Fatalf("Cannot update worker %+v: %v", w, err)
	}

	_, err = worker.VerifyToken(db, jwt)
	test.NoError(t, err)

	_, err = worker.VerifyToken(db, "this is not a jwt token")
	assert.Error(t, err)

	wk, err := worker.LoadByAccessTokenID(context.TODO(), db, token.ID)
	assert.NoError(t, err)
	assert.NotNil(t, wk)
	assert.Equal(t, w.Name, wk.Name)
	assert.Equal(t, w.ID, wk.ID)
	assert.Equal(t, w.AccessTokenID, wk.AccessTokenID)*/
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
	h, _ := assets.InsertHatchery(t, db, *g)
	m := assets.InsertWorkerModel(t, db, sdk.RandomString(5), g.ID)

	w, s, err := worker.RegisterWorker(db, store, hatchery.SpawnArguments{
		HatcheryName: h.Name,
		Model:        *m,
		RegisterOnly: true,
		WorkerName:   sdk.RandomString(10),
	}, h, sdk.WorkerRegistrationForm{
		Arch:               runtime.GOARCH,
		OS:                 runtime.GOOS,
		BinaryCapabilities: []string{"bash"},
	}, []sdk.Group{*g})

	test.NoError(t, err)
	assert.NotNil(t, w)
	t.Logf("jwt: %s", s)

	/*unsafeToken, _, err := new(jwt.Parser).ParseUnverified(s, &sdk.AccessTokenJWTClaims{})
	test.NoError(t, err)

	claims, ok := unsafeToken.Claims.(*sdk.AccessTokenJWTClaims)
	if ok {
		t.Logf("Token isValid %v %v", claims.Issuer, claims.StandardClaims.ExpiresAt)
	}
	assert.True(t, ok)

	wk, err := worker.LoadByAccessTokenID(context.TODO(), db, claims.ID)
	test.NoError(t, err)

	assert.Equal(t, w.ID, wk.ID)
	assert.Equal(t, w.ModelID, wk.ModelID)
	assert.Equal(t, w.HatcheryID, wk.HatcheryID)*/
}
