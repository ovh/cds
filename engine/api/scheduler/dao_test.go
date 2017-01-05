package scheduler

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/testwithdb"
	"github.com/ovh/cds/sdk"
)

func TestLoadAll(t *testing.T) {
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	_db, _ := testwithdb.SetupPG(t)
	db := database.DBMap(_db)

	schedulers, err := LoadAll(db)
	assert.NoError(t, err)
	assert.NotNil(t, schedulers)

	t.Logf("%v", schedulers)
}

func TestInsert(t *testing.T) {
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	_db, _ := testwithdb.SetupPG(t)
	db := database.DBMap(_db)

	//Insert Project
	pkey := testwithdb.RandomString(t, 10)
	proj, err := testwithdb.InsertTestProject(t, db, pkey, pkey)
	assert.NoError(t, err)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       pkey + "_PIP",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	if err := pipeline.InsertPipeline(db, pip); err != nil {
		t.Fatal(err)
	}

	//Insert Application
	app := &sdk.Application{
		Name: "TEST_APP",
	}
	t.Logf("Insert Application %s for Project %s", app.Name, proj.Name)
	if err := application.InsertApplication(db, proj, app); err != nil {
		t.Fatal(err)
	}

	t.Logf("Attach Pipeline %s on Application %s", pip.Name, app.Name)
	if err := application.AttachPipeline(db, app.ID, pip.ID); err != nil {
		t.Fatal(err)
	}

	s := &sdk.PipelineScheduler{
		ApplicationID: app.ID,
		EnvironmentID: sdk.DefaultEnv.ID,
		PipelineID:    pip.ID,
		Crontab:       "@hourly",
		Disabled:      false,
		Args: []sdk.Parameter{
			{
				Name:  "p1",
				Type:  sdk.StringParameter,
				Value: "v1",
			},
			{
				Name:  "p2",
				Type:  sdk.StringParameter,
				Value: "v2",
			},
		},
	}
	if err := Insert(db, s); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(db, s.ID)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%v", s)
	t.Logf("%v", loaded)

	assert.True(t, reflect.DeepEqual(s, loaded))
}

func TestUpdate(t *testing.T) {
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	_db, _ := testwithdb.SetupPG(t)
	db := database.DBMap(_db)

	//Insert Project
	pkey := testwithdb.RandomString(t, 10)
	proj, err := testwithdb.InsertTestProject(t, db, pkey, pkey)
	assert.NoError(t, err)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       pkey + "_PIP",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	if err := pipeline.InsertPipeline(db, pip); err != nil {
		t.Fatal(err)
	}

	//Insert Application
	app := &sdk.Application{
		Name: "TEST_APP",
	}
	t.Logf("Insert Application %s for Project %s", app.Name, proj.Name)
	if err := application.InsertApplication(db, proj, app); err != nil {
		t.Fatal(err)
	}

	t.Logf("Attach Pipeline %s on Application %s", pip.Name, app.Name)
	if err := application.AttachPipeline(db, app.ID, pip.ID); err != nil {
		t.Fatal(err)
	}

	s := &sdk.PipelineScheduler{
		ApplicationID: app.ID,
		EnvironmentID: sdk.DefaultEnv.ID,
		PipelineID:    pip.ID,
		Crontab:       "@hourly",
		Disabled:      false,
		Args: []sdk.Parameter{
			{
				Name:  "p1",
				Type:  sdk.StringParameter,
				Value: "v1",
			},
			{
				Name:  "p2",
				Type:  sdk.StringParameter,
				Value: "v2",
			},
		},
	}
	if err := Insert(db, s); err != nil {
		t.Fatal(err)
	}

	if err := Update(db, s); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(db, s.ID)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%v", s)
	t.Logf("%v", loaded)

	assert.True(t, reflect.DeepEqual(s, loaded))

	if err := Delete(db, s); err != nil {
		t.Fatal(err)
	}
}

func TestLoadPendingExecutions(t *testing.T) {
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	_db, _ := testwithdb.SetupPG(t)
	db := database.DBMap(_db)
	pe, err := LoadPendingExecutions(db)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v", pe)
}

func TestGetByApplication(t *testing.T) {
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	_db, _ := testwithdb.SetupPG(t)
	db := database.DBMap(_db)

	//Insert Project
	pkey := testwithdb.RandomString(t, 10)
	proj, err := testwithdb.InsertTestProject(t, db, pkey, pkey)
	assert.NoError(t, err)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       pkey + "_PIP",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	if err := pipeline.InsertPipeline(db, pip); err != nil {
		t.Fatal(err)
	}

	//Insert Application
	app := &sdk.Application{
		Name: "TEST_APP",
	}
	t.Logf("Insert Application %s for Project %s", app.Name, proj.Name)
	if err := application.InsertApplication(db, proj, app); err != nil {
		t.Fatal(err)
	}

	t.Logf("Attach Pipeline %s on Application %s", pip.Name, app.Name)
	if err := application.AttachPipeline(db, app.ID, pip.ID); err != nil {
		t.Fatal(err)
	}

	s := &sdk.PipelineScheduler{
		ApplicationID: app.ID,
		EnvironmentID: sdk.DefaultEnv.ID,
		PipelineID:    pip.ID,
		Crontab:       "@hourly",
		Disabled:      false,
		Args: []sdk.Parameter{
			{
				Name:  "p1",
				Type:  sdk.StringParameter,
				Value: "v1",
			},
			{
				Name:  "p2",
				Type:  sdk.StringParameter,
				Value: "v2",
			},
		},
	}
	if err := Insert(db, s); err != nil {
		t.Fatal(err)
	}

	actuals, err := GetByApplication(db, app)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, len(actuals))
}

func TestGetByPipeline(t *testing.T) {
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	_db, _ := testwithdb.SetupPG(t)
	db := database.DBMap(_db)

	//Insert Project
	pkey := testwithdb.RandomString(t, 10)
	proj, err := testwithdb.InsertTestProject(t, db, pkey, pkey)
	assert.NoError(t, err)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       pkey + "_PIP",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	if err := pipeline.InsertPipeline(db, pip); err != nil {
		t.Fatal(err)
	}

	//Insert Application
	app := &sdk.Application{
		Name: "TEST_APP",
	}
	t.Logf("Insert Application %s for Project %s", app.Name, proj.Name)
	if err := application.InsertApplication(db, proj, app); err != nil {
		t.Fatal(err)
	}

	t.Logf("Attach Pipeline %s on Application %s", pip.Name, app.Name)
	if err := application.AttachPipeline(db, app.ID, pip.ID); err != nil {
		t.Fatal(err)
	}

	s := &sdk.PipelineScheduler{
		ApplicationID: app.ID,
		EnvironmentID: sdk.DefaultEnv.ID,
		PipelineID:    pip.ID,
		Crontab:       "@hourly",
		Disabled:      false,
		Args: []sdk.Parameter{
			{
				Name:  "p1",
				Type:  sdk.StringParameter,
				Value: "v1",
			},
			{
				Name:  "p2",
				Type:  sdk.StringParameter,
				Value: "v2",
			},
		},
	}
	if err := Insert(db, s); err != nil {
		t.Fatal(err)
	}

	actuals, err := GetByPipeline(db, pip)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, len(actuals))
}

func TestGetByApplicationPipeline(t *testing.T) {
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	_db, _ := testwithdb.SetupPG(t)
	db := database.DBMap(_db)

	//Insert Project
	pkey := testwithdb.RandomString(t, 10)
	proj, err := testwithdb.InsertTestProject(t, db, pkey, pkey)
	assert.NoError(t, err)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       pkey + "_PIP",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	if err := pipeline.InsertPipeline(db, pip); err != nil {
		t.Fatal(err)
	}

	//Insert Application
	app := &sdk.Application{
		Name: "TEST_APP",
	}
	t.Logf("Insert Application %s for Project %s", app.Name, proj.Name)
	if err := application.InsertApplication(db, proj, app); err != nil {
		t.Fatal(err)
	}

	t.Logf("Attach Pipeline %s on Application %s", pip.Name, app.Name)
	if err := application.AttachPipeline(db, app.ID, pip.ID); err != nil {
		t.Fatal(err)
	}

	s := &sdk.PipelineScheduler{
		ApplicationID: app.ID,
		EnvironmentID: sdk.DefaultEnv.ID,
		PipelineID:    pip.ID,
		Crontab:       "@hourly",
		Disabled:      false,
		Args: []sdk.Parameter{
			{
				Name:  "p1",
				Type:  sdk.StringParameter,
				Value: "v1",
			},
			{
				Name:  "p2",
				Type:  sdk.StringParameter,
				Value: "v2",
			},
		},
	}
	if err := Insert(db, s); err != nil {
		t.Fatal(err)
	}

	actuals, err := GetByApplicationPipeline(db, app, pip)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, len(actuals))
}

func TestGetByApplicationPipelineEnv(t *testing.T) {
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	_db, _ := testwithdb.SetupPG(t)
	db := database.DBMap(_db)

	//Insert Project
	pkey := testwithdb.RandomString(t, 10)
	proj, err := testwithdb.InsertTestProject(t, db, pkey, pkey)
	assert.NoError(t, err)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       pkey + "_PIP",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	if err := pipeline.InsertPipeline(db, pip); err != nil {
		t.Fatal(err)
	}

	//Insert Application
	app := &sdk.Application{
		Name: "TEST_APP",
	}
	t.Logf("Insert Application %s for Project %s", app.Name, proj.Name)
	if err := application.InsertApplication(db, proj, app); err != nil {
		t.Fatal(err)
	}

	t.Logf("Attach Pipeline %s on Application %s", pip.Name, app.Name)
	if err := application.AttachPipeline(db, app.ID, pip.ID); err != nil {
		t.Fatal(err)
	}

	s := &sdk.PipelineScheduler{
		ApplicationID: app.ID,
		EnvironmentID: sdk.DefaultEnv.ID,
		PipelineID:    pip.ID,
		Crontab:       "@hourly",
		Disabled:      false,
		Args: []sdk.Parameter{
			{
				Name:  "p1",
				Type:  sdk.StringParameter,
				Value: "v1",
			},
			{
				Name:  "p2",
				Type:  sdk.StringParameter,
				Value: "v2",
			},
		},
	}
	if err := Insert(db, s); err != nil {
		t.Fatal(err)
	}

	actuals, err := GetByApplicationPipelineEnv(db, app, pip, sdk.DefaultEnv)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, len(actuals))
}
