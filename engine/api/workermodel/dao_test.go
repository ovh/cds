package workermodel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func TestInsertAndUpdate_WithRegistryPassword(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	g := assets.InsertGroup(t, db)

	// Insert a model with a registry password
	m := sdk.Model{
		Name: sdk.RandomString(10),
		Type: sdk.Docker,
		ModelDocker: sdk.ModelDocker{
			Image:    "foo/bar:3.4",
			Private:  true,
			Password: "my-clear-password",
		},
		GroupID: g.ID,
	}
	require.NoError(t, workermodel.Insert(context.TODO(), db, &m))

	wm, err := workermodel.LoadByID(context.TODO(), db, m.ID)
	require.NoError(t, err)
	assert.Equal(t, "{{.secrets.registry_password}}", wm.ModelDocker.Password)
	s, err := workermodel.LoadSecretByModelIDAndName(context.TODO(), db, wm.ID, "secrets.registry_password")
	require.NoError(t, err)
	assert.Equal(t, "my-clear-password", s.Value)

	// No change expected when updating with password input that contains placeholder value
	require.NoError(t, workermodel.UpdateDB(context.TODO(), db, &m))

	wm, err = workermodel.LoadByID(context.TODO(), db, m.ID)
	require.NoError(t, err)
	assert.Equal(t, "{{.secrets.registry_password}}", wm.ModelDocker.Password)
	s, err = workermodel.LoadSecretByModelIDAndName(context.TODO(), db, m.ID, "secrets.registry_password")
	require.NoError(t, err)
	assert.Equal(t, "my-clear-password", s.Value)

	// Empty password should be stored
	m.ModelDocker.Password = ""
	require.NoError(t, workermodel.UpdateDB(context.TODO(), db, &m))

	wm, err = workermodel.LoadByID(context.TODO(), db, m.ID)
	require.NoError(t, err)
	assert.Equal(t, "{{.secrets.registry_password}}", wm.ModelDocker.Password)
	s, err = workermodel.LoadSecretByModelIDAndName(context.TODO(), db, m.ID, "secrets.registry_password")
	require.NoError(t, err)
	assert.Equal(t, "", s.Value)

	// Disabling private registry should delete the secret
	m.ModelDocker.Private = false
	require.NoError(t, workermodel.UpdateDB(context.TODO(), db, &m))

	wm, err = workermodel.LoadByID(context.TODO(), db, m.ID)
	require.NoError(t, err)
	assert.Equal(t, "", wm.ModelDocker.Password)
	s, err = workermodel.LoadSecretByModelIDAndName(context.TODO(), db, m.ID, "secrets.registry_password")
	require.Error(t, err)

	// Enabling private registry should create the secret even if empty string
	m.ModelDocker.Private = true
	require.NoError(t, workermodel.UpdateDB(context.TODO(), db, &m))

	wm, err = workermodel.LoadByID(context.TODO(), db, m.ID)
	require.NoError(t, err)
	assert.Equal(t, "{{.secrets.registry_password}}", wm.ModelDocker.Password)
	s, err = workermodel.LoadSecretByModelIDAndName(context.TODO(), db, m.ID, "secrets.registry_password")
	require.NoError(t, err)
	assert.Equal(t, "", s.Value)

	// Changing the type of the model should delete the secret
	m.Type = sdk.Openstack
	require.NoError(t, workermodel.UpdateDB(context.TODO(), db, &m))

	wm, err = workermodel.LoadByID(context.TODO(), db, m.ID)
	require.NoError(t, err)
	assert.Equal(t, "", wm.ModelDocker.Password)
	s, err = workermodel.LoadSecretByModelIDAndName(context.TODO(), db, m.ID, "secrets.registry_password")
	require.Error(t, err)
}

func insertWorkerModel(t *testing.T, db gorpmapper.SqlExecutorWithTx, name string, groupID int64, req ...sdk.Requirement) *sdk.Model {
	m := sdk.Model{
		Name: name,
		Type: sdk.Docker,
		ModelDocker: sdk.ModelDocker{
			Image: "foo/bar:3.4",
		},
		GroupID: groupID,
		RegisteredCapabilities: append(req, sdk.Requirement{
			Name:  "capa_1",
			Type:  sdk.BinaryRequirement,
			Value: "capa_1",
		}),
	}
	require.NoError(t, workermodel.Insert(context.TODO(), db, &m))
	return &m
}

func TestInsert(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	g := assets.InsertGroup(t, db)

	src := insertWorkerModel(t, db, sdk.RandomString(10), g.ID)
	require.NotEqual(t, 0, src.ID)

	res, err := workermodel.LoadByID(context.TODO(), db, src.ID, workermodel.LoadOptions.Default)
	require.NoError(t, err)

	// lastregistration is LOCALTIMESTAMP (at sql insert)
	// set it manually to allow use EqualValues on others fields
	src.LastRegistration = res.LastRegistration
	src.UserLastModified = res.UserLastModified

	// remove group from result
	res.Group = nil

	assert.EqualValues(t, *src, *res)
}

func TestLoadByNameAndGroupID(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	g := assets.InsertGroup(t, db)

	src := insertWorkerModel(t, db, sdk.RandomString(10), g.ID)

	res, err := workermodel.LoadByNameAndGroupID(context.TODO(), db, src.Name, g.ID)
	require.NoError(t, err)
	assert.Equal(t, src.ID, res.ID)

	_, err = workermodel.LoadByNameAndGroupID(context.TODO(), db, "NotExisting", g.ID)
	assert.True(t, sdk.ErrorIs(err, sdk.ErrNotFound))
}

func TestLoadWorkerModelsByNameAndGroupIDs(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	g1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	insertWorkerModel(t, db, "SameName", g1.ID)
	insertWorkerModel(t, db, "SameName", g2.ID)
	insertWorkerModel(t, db, "DiffName", g2.ID)

	wms, err := workermodel.LoadAllByNameAndGroupIDs(context.TODO(), db, "SameName", []int64{g1.ID})
	require.NoError(t, err)
	assert.Len(t, wms, 1)

	wms, err = workermodel.LoadAllByNameAndGroupIDs(context.TODO(), db, "SameName", []int64{g1.ID, g2.ID})
	require.NoError(t, err)
	assert.Len(t, wms, 2)

	wms, err = workermodel.LoadAllByNameAndGroupIDs(context.TODO(), db, "DiffName", []int64{g1.ID, g2.ID})
	require.NoError(t, err)
	assert.Len(t, wms, 1)

	wms, err = workermodel.LoadAllByNameAndGroupIDs(context.TODO(), db, "Unknown", []int64{g1.ID, g2.ID})
	require.NoError(t, err)
	assert.Len(t, wms, 0)
}

func TestLoadAll(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	// delete all workers
	wks, err := worker.LoadAll(context.TODO(), db)
	require.NoError(t, err)
	for _, wk := range wks {
		require.NoError(t, worker.Delete(db, wk.ID))
	}
	models, err := workermodel.LoadAll(context.TODO(), db, nil)
	require.NoError(t, err)
	for _, m := range models {
		require.NoError(t, workermodel.DeleteByID(db, m.ID))
	}

	g := assets.InsertGroup(t, db)

	m1 := insertWorkerModel(t, db, "abc", g.ID)
	m2 := sdk.Model{
		Name:         "def",
		GroupID:      g.ID,
		IsDeprecated: true,
	}
	require.NoError(t, workermodel.Insert(context.TODO(), db, &m2))
	m3 := sdk.Model{
		Name:         "ghi",
		GroupID:      g.ID,
		IsDeprecated: true,
		RegisteredCapabilities: []sdk.Requirement{{
			Name:  "capa_1",
			Type:  sdk.BinaryRequirement,
			Value: "capa_1",
		}},
	}
	require.NoError(t, workermodel.Insert(context.TODO(), db, &m3))

	models, err = workermodel.LoadAll(context.TODO(), db, nil)
	require.NoError(t, err)
	require.Len(t, models, 3)
	assert.Equal(t, m1.ID, models[0].ID)
	assert.Equal(t, m2.ID, models[1].ID)
	assert.Equal(t, m3.ID, models[2].ID)

	models, err = workermodel.LoadAll(context.TODO(), db, &workermodel.LoadFilter{})
	require.NoError(t, err)
	assert.Len(t, models, 3)

	models, err = workermodel.LoadAll(context.TODO(), db, &workermodel.LoadFilter{
		State: workermodel.StateActive,
	})
	require.NoError(t, err)
	require.Len(t, models, 1)
	assert.Equal(t, m1.ID, models[0].ID)

	models, err = workermodel.LoadAll(context.TODO(), db, &workermodel.LoadFilter{
		Binary: "unknown",
	})
	require.NoError(t, err)
	assert.Len(t, models, 0)

	models, err = workermodel.LoadAll(context.TODO(), db, &workermodel.LoadFilter{
		Binary: "capa_1",
	})
	require.NoError(t, err)
	require.Len(t, models, 2)
	assert.Equal(t, m1.ID, models[0].ID)
	assert.Equal(t, m3.ID, models[1].ID)

	models, err = workermodel.LoadAll(context.TODO(), db, &workermodel.LoadFilter{
		State:  workermodel.StateActive,
		Binary: "capa_1",
	})
	require.NoError(t, err)
	require.Len(t, models, 1)
	assert.Equal(t, m1.ID, models[0].ID)
}

func TestLoadAllByGroupIDs(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	g1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	m1 := insertWorkerModel(t, db, "abc", g1.ID)
	m2 := insertWorkerModel(t, db, "def", g2.ID, sdk.Requirement{
		Name:  "capa_2",
		Type:  sdk.BinaryRequirement,
		Value: "capa_2",
	})
	m3 := sdk.Model{
		Name:     "ghi",
		GroupID:  g2.ID,
		Disabled: true,
	}
	require.NoError(t, workermodel.Insert(context.TODO(), db, &m3))
	m4 := sdk.Model{
		Name:     "jkl",
		GroupID:  g2.ID,
		Disabled: true,
		RegisteredCapabilities: []sdk.Requirement{{
			Name:  "capa_2",
			Type:  sdk.BinaryRequirement,
			Value: "capa_2",
		}},
	}
	require.NoError(t, workermodel.Insert(context.TODO(), db, &m4))

	wms, err := workermodel.LoadAllByGroupIDs(context.TODO(), db, []int64{g1.ID}, nil)
	require.NoError(t, err)
	require.Len(t, wms, 1)
	assert.Equal(t, m1.ID, wms[0].ID)

	wms, err = workermodel.LoadAllByGroupIDs(context.TODO(), db, []int64{g1.ID, g2.ID}, nil)
	require.NoError(t, err)
	require.Len(t, wms, 4)
	assert.Equal(t, m1.ID, wms[0].ID)
	assert.Equal(t, m2.ID, wms[1].ID)
	assert.Equal(t, m3.ID, wms[2].ID)
	assert.Equal(t, m4.ID, wms[3].ID)

	wms, err = workermodel.LoadAllByGroupIDs(context.TODO(), db, []int64{g1.ID, g2.ID}, &workermodel.LoadFilter{
		Binary: "capa_2",
	})
	require.NoError(t, err)
	require.Len(t, wms, 2)
	assert.Equal(t, m2.ID, wms[0].ID)
	assert.Equal(t, m4.ID, wms[1].ID)

	wms, err = workermodel.LoadAllByGroupIDs(context.TODO(), db, []int64{g1.ID, g2.ID}, &workermodel.LoadFilter{
		State: workermodel.StateDisabled,
	})
	require.NoError(t, err)
	require.Len(t, wms, 2)
	assert.Equal(t, m3.ID, wms[0].ID)
	assert.Equal(t, m4.ID, wms[1].ID)

	wms, err = workermodel.LoadAllByGroupIDs(context.TODO(), db, []int64{g1.ID, g2.ID}, &workermodel.LoadFilter{
		Binary: "capa_2",
		State:  workermodel.StateDisabled,
	})
	require.NoError(t, err)
	require.Len(t, wms, 1)
	assert.Equal(t, m4.ID, wms[0].ID)
}

func TestLoadCapabilities(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	g := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	m := insertWorkerModel(t, db, sdk.RandomString(10), g.ID, sdk.Requirement{
		Name:  "capa_2",
		Type:  sdk.BinaryRequirement,
		Value: "capa_2",
	})

	cs, err := workermodel.LoadCapabilitiesByModelID(context.TODO(), db, m.ID)
	require.NoError(t, err)
	require.Len(t, cs, 2)
	assert.Equal(t, "capa_1", cs[0].Name)
	assert.Equal(t, "capa_2", cs[1].Name)
}

func TestUpdate(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	g := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	src := insertWorkerModel(t, db, sdk.RandomString(10), g.ID)

	data := *src
	data.Type = sdk.Openstack
	data.RegisteredCapabilities = append(data.RegisteredCapabilities, sdk.Requirement{
		Name:  "capa_2",
		Type:  sdk.BinaryRequirement,
		Value: "capa_2",
	})
	require.NoError(t, workermodel.UpdateDB(context.TODO(), db, &data))

	res, err := workermodel.LoadByID(context.TODO(), db, src.ID, workermodel.LoadOptions.Default)
	require.NoError(t, err)
	assert.Equal(t, sdk.Openstack, res.Type)
	assert.Len(t, res.RegisteredCapabilities, 2)
}

func TestLoadWorkerModelsForGroupIDs(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	g1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	m1 := insertWorkerModel(t, db, "abc", g1.ID)
	m2 := insertWorkerModel(t, db, "def", g2.ID)
	m3 := sdk.Model{
		Name:             "ghi",
		Type:             sdk.Docker,
		ModelDocker:      sdk.ModelDocker{Image: "foo/bar:3.4"},
		GroupID:          g2.ID,
		UserLastModified: time.Now(),
		Disabled:         true,
	}
	require.NoError(t, workermodel.Insert(context.TODO(), db, &m3))

	models, err := workermodel.LoadAllActiveAndNotDeprecatedForGroupIDs(context.TODO(), db, []int64{g1.ID})
	require.NoError(t, err)
	require.Len(t, models, 1)
	assert.Equal(t, m1.ID, models[0].ID)

	models, err = workermodel.LoadAllActiveAndNotDeprecatedForGroupIDs(context.TODO(), db, []int64{g1.ID, g2.ID})
	require.NoError(t, err)
	require.Len(t, models, 2)
	assert.Equal(t, m1.ID, models[0].ID)
	assert.Equal(t, m2.ID, models[1].ID)
}

// TestDisableExpired encodes the point of the end of life date: once it is reached, a deprecated
// model must stop being usable on its own, and nothing else must be touched. The second call
// asserts idempotence, which is what makes the lifecycle routine safe to run concurrently on
// several API instances without a lock.
func TestDisableExpired(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	models, err := workermodel.LoadAll(context.TODO(), db, nil)
	require.NoError(t, err)
	for _, m := range models {
		require.NoError(t, workermodel.DeleteByID(db, m.ID))
	}

	g := assets.InsertGroup(t, db)
	past := time.Now().Add(-24 * time.Hour)
	future := time.Now().Add(24 * time.Hour)

	expired := insertWorkerModelWithEOL(t, db, "expired", g.ID, true, &past)
	notYet := insertWorkerModelWithEOL(t, db, "not-yet", g.ID, true, &future)
	noEOL := insertWorkerModelWithEOL(t, db, "no-eol", g.ID, true, nil)
	notDeprecated := insertWorkerModelWithEOL(t, db, "not-deprecated", g.ID, false, nil)

	now := time.Now()

	// only the deprecated model past its end of life date is a candidate
	candidates, err := workermodel.LoadAllPastEOL(context.TODO(), db, now)
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	assert.Equal(t, expired.ID, candidates[0].ID)

	disabled, err := workermodel.DisableExpired(context.TODO(), db, expired.ID, now)
	require.NoError(t, err)
	assert.True(t, disabled)

	reloaded, err := workermodel.LoadByID(context.TODO(), db, expired.ID)
	require.NoError(t, err)
	assert.True(t, reloaded.Disabled, "a deprecated model past its end of life date must be disabled")
	assert.True(t, reloaded.IsDeprecated, "disabling must not clear the deprecated flag")
	require.NotNil(t, reloaded.EOL)

	// a second pass must not report a change, so a concurrent API instance stays a no-op
	disabled, err = workermodel.DisableExpired(context.TODO(), db, expired.ID, now)
	require.NoError(t, err)
	assert.False(t, disabled)

	// the three other models are left untouched
	for _, id := range []int64{notYet.ID, noEOL.ID, notDeprecated.ID} {
		m, err := workermodel.LoadByID(context.TODO(), db, id)
		require.NoError(t, err)
		assert.False(t, m.Disabled, "model %s must not be disabled", m.Name)

		disabled, err := workermodel.DisableExpired(context.TODO(), db, id, now)
		require.NoError(t, err)
		assert.False(t, disabled, "model %s must not be a candidate", m.Name)
	}
}

func insertWorkerModelWithEOL(t *testing.T, db gorpmapper.SqlExecutorWithTx, name string, groupID int64, deprecated bool, eol *time.Time) *sdk.Model {
	m := sdk.Model{
		Name:         name,
		Type:         sdk.Docker,
		ModelDocker:  sdk.ModelDocker{Image: "foo/bar:3.4"},
		GroupID:      groupID,
		IsDeprecated: deprecated,
		EOL:          eol,
	}
	require.NoError(t, workermodel.Insert(context.TODO(), db, &m))
	return &m
}
