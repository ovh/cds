package workermodel_test

import (
	"context"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/worker"

	"github.com/go-gorp/gorp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/sdk"
)

func deleteAllWorkerModel(t *testing.T, db gorp.SqlExecutor) {
	wks, err := worker.LoadAll(context.TODO(), db)
	require.NoError(t, err)

	for _, wk := range wks {
		require.NoError(t, worker.Delete(db, wk.ID))
	}

	models, err := workermodel.LoadAll(context.TODO(), db, nil)
	require.NoError(t, err)

	for _, m := range models {
		test.NoError(t, workermodel.Delete(db, m.ID))
	}
}

func insertGroup(t *testing.T, db gorp.SqlExecutor) *sdk.Group {
	g := &sdk.Group{
		Name: "test-group-model",
	}

	g1, _ := group.LoadByName(context.TODO(), db, g.Name)
	if g1 != nil {
		require.NoError(t, group.Delete(context.TODO(), db, g1))
	}

	if err := group.Insert(context.TODO(), db, g); err != nil {
		t.Fatalf("Unable to create group %s", err)
	}

	return g
}

func insertWorkerModel(t *testing.T, db gorp.SqlExecutor, name string, groupID int64, req ...sdk.Requirement) *sdk.Model {
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
		UserLastModified: time.Now(),
	}

	test.NoError(t, workermodel.Insert(db, &m))

	return &m
}

func TestInsert(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	deleteAllWorkerModel(t, db)

	g := assets.InsertGroup(t, db)

	src := insertWorkerModel(t, db, sdk.RandomString(10), g.ID)
	test.NotEqual(t, 0, src.ID)

	res, err := workermodel.LoadByID(db, src.ID)
	test.NoError(t, err)

	// lastregistration is LOCALTIMESTAMP (at sql insert)
	// set it manually to allow use EqualValues on others fields
	src.LastRegistration = res.LastRegistration
	src.UserLastModified = res.UserLastModified

	// remove group from result
	res.Group = nil

	assert.EqualValues(t, *src, *res)
}

func TestLoadByNameAndGroupID(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	deleteAllWorkerModel(t, db)

	g, err := group.LoadByName(context.TODO(), db, "shared.infra")
	test.NoError(t, err)

	src := insertWorkerModel(t, db, sdk.RandomString(10), g.ID)

	res, err := workermodel.LoadByNameAndGroupID(db, src.Name, g.ID)
	test.NoError(t, err)
	test.Equal(t, src.ID, res.ID)

	_, err = workermodel.LoadByNameAndGroupID(db, "NotExisting", g.ID)
	test.Equal(t, true, sdk.ErrorIs(err, sdk.ErrNoWorkerModel))
}

func TestLoadWorkerModelsByNameAndGroupIDs(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	deleteAllWorkerModel(t, db)

	g1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	insertWorkerModel(t, db, "SameName", g1.ID)
	insertWorkerModel(t, db, "SameName", g2.ID)
	insertWorkerModel(t, db, "DiffName", g2.ID)

	wms, err := workermodel.LoadAllByNameAndGroupIDs(context.TODO(), db, "SameName", []int64{g1.ID})
	test.NoError(t, err)
	test.Equal(t, 1, len(wms))

	wms, err = workermodel.LoadAllByNameAndGroupIDs(context.TODO(), db, "SameName", []int64{g1.ID, g2.ID})
	test.NoError(t, err)
	test.Equal(t, 2, len(wms))

	wms, err = workermodel.LoadAllByNameAndGroupIDs(context.TODO(), db, "DiffName", []int64{g1.ID, g2.ID})
	test.NoError(t, err)
	test.Equal(t, 1, len(wms))

	wms, err = workermodel.LoadAllByNameAndGroupIDs(context.TODO(), db, "Unknown", []int64{g1.ID, g2.ID})
	test.NoError(t, err)
	test.Equal(t, 0, len(wms))
}

func TestLoadAll(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	deleteAllWorkerModel(t, db)

	g := insertGroup(t, db)

	m1 := insertWorkerModel(t, db, "abc", g.ID)
	m2 := sdk.Model{
		Name:         "def",
		GroupID:      g.ID,
		IsDeprecated: true,
	}
	test.NoError(t, workermodel.Insert(db, &m2))
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
	test.NoError(t, workermodel.Insert(db, &m3))

	models, err := workermodel.LoadAll(context.TODO(), db, nil)
	test.NoError(t, err)
	test.Equal(t, 3, len(models))
	test.Equal(t, m1.ID, models[0].ID)
	test.Equal(t, m2.ID, models[1].ID)
	test.Equal(t, m3.ID, models[2].ID)

	models, err = workermodel.LoadAll(context.TODO(), db, &workermodel.LoadFilter{})
	test.NoError(t, err)
	test.Equal(t, 3, len(models))

	models, err = workermodel.LoadAll(context.TODO(), db, &workermodel.LoadFilter{
		State: workermodel.StateActive,
	})
	test.NoError(t, err)
	test.Equal(t, 1, len(models))
	test.Equal(t, m1.ID, models[0].ID)

	models, err = workermodel.LoadAll(context.TODO(), db, &workermodel.LoadFilter{
		Binary: "unknown",
	})
	test.NoError(t, err)
	test.Equal(t, 0, len(models))

	models, err = workermodel.LoadAll(context.TODO(), db, &workermodel.LoadFilter{
		Binary: "capa_1",
	})
	test.NoError(t, err)
	test.Equal(t, 2, len(models))
	test.Equal(t, m1.ID, models[0].ID)
	test.Equal(t, m3.ID, models[1].ID)

	models, err = workermodel.LoadAll(context.TODO(), db, &workermodel.LoadFilter{
		State:  workermodel.StateActive,
		Binary: "capa_1",
	})
	test.NoError(t, err)
	test.Equal(t, 1, len(models))
	test.Equal(t, m1.ID, models[0].ID)
}

func TestLoadAllByGroupIDs(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	deleteAllWorkerModel(t, db)

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
	test.NoError(t, workermodel.Insert(db, &m3))
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
	test.NoError(t, workermodel.Insert(db, &m4))

	wms, err := workermodel.LoadAllByGroupIDs(context.TODO(), db, []int64{g1.ID}, nil)
	test.NoError(t, err)
	test.Equal(t, 1, len(wms))
	test.Equal(t, m1.ID, wms[0].ID)

	wms, err = workermodel.LoadAllByGroupIDs(context.TODO(), db, []int64{g1.ID, g2.ID}, nil)
	test.NoError(t, err)
	test.Equal(t, 4, len(wms))
	test.Equal(t, m1.ID, wms[0].ID)
	test.Equal(t, m2.ID, wms[1].ID)
	test.Equal(t, m3.ID, wms[2].ID)
	test.Equal(t, m4.ID, wms[3].ID)

	wms, err = workermodel.LoadAllByGroupIDs(context.TODO(), db, []int64{g1.ID, g2.ID}, &workermodel.LoadFilter{
		Binary: "capa_2",
	})
	test.NoError(t, err)
	test.Equal(t, 2, len(wms))
	test.Equal(t, m2.ID, wms[0].ID)
	test.Equal(t, m4.ID, wms[1].ID)

	wms, err = workermodel.LoadAllByGroupIDs(context.TODO(), db, []int64{g1.ID, g2.ID}, &workermodel.LoadFilter{
		State: workermodel.StateDisabled,
	})
	test.NoError(t, err)
	test.Equal(t, 2, len(wms))
	test.Equal(t, m3.ID, wms[0].ID)
	test.Equal(t, m4.ID, wms[1].ID)

	wms, err = workermodel.LoadAllByGroupIDs(context.TODO(), db, []int64{g1.ID, g2.ID}, &workermodel.LoadFilter{
		Binary: "capa_2",
		State:  workermodel.StateDisabled,
	})
	test.NoError(t, err)
	test.Equal(t, 1, len(wms))
	test.Equal(t, m4.ID, wms[0].ID)
}

func TestLoadAllByBinary(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	deleteAllWorkerModel(t, db)

	g := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	insertWorkerModel(t, db, sdk.RandomString(10), g.ID)
	m2 := insertWorkerModel(t, db, sdk.RandomString(10), g.ID, sdk.Requirement{
		Name:  "capa_2",
		Type:  sdk.BinaryRequirement,
		Value: "capa_2",
	})

	models, err := workermodel.LoadAllByBinary(db, "capa_0")
	test.NoError(t, err)
	test.Equal(t, 0, len(models))

	models, err = workermodel.LoadAllByBinary(db, "capa_1")
	test.NoError(t, err)
	test.Equal(t, 2, len(models))

	models, err = workermodel.LoadAllByBinary(db, "capa_2")
	test.NoError(t, err)
	test.Equal(t, 1, len(models))
	test.Equal(t, m2.ID, models[0].ID)
}

func TestLoadAllByBinaryAndGroupIDs(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	deleteAllWorkerModel(t, db)

	g1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	m1 := insertWorkerModel(t, db, "abc", g1.ID)
	m2 := insertWorkerModel(t, db, "def", g2.ID)
	m3 := insertWorkerModel(t, db, "ghi", g2.ID, sdk.Requirement{
		Name:  "capa_2",
		Type:  sdk.BinaryRequirement,
		Value: "capa_2",
	})

	wms, err := workermodel.LoadAllByBinaryAndGroupIDs(db, "capa_0", []int64{g1.ID})
	test.NoError(t, err)
	test.Equal(t, 0, len(wms))

	wms, err = workermodel.LoadAllByBinaryAndGroupIDs(db, "capa_1", []int64{g1.ID})
	test.NoError(t, err)
	test.Equal(t, 1, len(wms))
	test.Equal(t, m1.ID, wms[0].ID)

	wms, err = workermodel.LoadAllByBinaryAndGroupIDs(db, "capa_1", []int64{g1.ID, g2.ID})
	test.NoError(t, err)
	test.Equal(t, 3, len(wms))
	test.Equal(t, m1.ID, wms[0].ID)
	test.Equal(t, m2.ID, wms[1].ID)
	test.Equal(t, m3.ID, wms[2].ID)

	wms, err = workermodel.LoadAllByBinaryAndGroupIDs(db, "capa_2", []int64{g1.ID, g2.ID})
	test.NoError(t, err)
	test.Equal(t, 1, len(wms))
	test.Equal(t, m3.ID, wms[0].ID)
}

func TestLoadCapabilities(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	deleteAllWorkerModel(t, db)

	g := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	m := insertWorkerModel(t, db, sdk.RandomString(10), g.ID, sdk.Requirement{
		Name:  "capa_2",
		Type:  sdk.BinaryRequirement,
		Value: "capa_2",
	})

	cs, err := workermodel.LoadCapabilities(db, m.ID)
	test.NoError(t, err)
	test.Equal(t, 2, len(cs))
	test.Equal(t, "capa_1", cs[0].Name)
	test.Equal(t, "capa_2", cs[1].Name)
}

func TestUpdate(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	deleteAllWorkerModel(t, db)

	g := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	src := insertWorkerModel(t, db, sdk.RandomString(10), g.ID)
	data := *src

	data.Type = sdk.Openstack
	data.RegisteredCapabilities = append(data.RegisteredCapabilities, sdk.Requirement{
		Name:  "capa_2",
		Type:  sdk.BinaryRequirement,
		Value: "capa_2",
	})

	test.NoError(t, workermodel.UpdateDB(db, &data))

	res, err := workermodel.LoadByID(db, src.ID)
	test.NoError(t, err)
	test.Equal(t, sdk.Openstack, res.Type)
	test.Equal(t, 2, len(res.RegisteredCapabilities))
}

func TestLoadWorkerModelsForGroupIDs(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	deleteAllWorkerModel(t, db)

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
	test.NoError(t, workermodel.Insert(db, &m3))

	models, err := workermodel.LoadAllActiveAndNotDeprecatedForGroupIDs(db, []int64{g1.ID})
	test.NoError(t, err)
	test.Equal(t, 1, len(models))
	test.Equal(t, m1.ID, models[0].ID)

	models, err = workermodel.LoadAllActiveAndNotDeprecatedForGroupIDs(db, []int64{g1.ID, g2.ID})
	test.NoError(t, err)
	test.Equal(t, 2, len(models))
	test.Equal(t, m1.ID, models[0].ID)
	test.Equal(t, m2.ID, models[1].ID)
}
