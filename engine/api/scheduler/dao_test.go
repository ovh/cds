package scheduler

import (
	"testing"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/testwithdb"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func TestLoadAllPipelineScheduler(t *testing.T) {
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	_db, _ := testwithdb.SetupPG(t)
	db := database.DBMap(_db)

	schedulers, err := LoadAll(db)
	assert.NoError(t, err)
	assert.NotNil(t, schedulers)
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

	s := &sdk.PipelineScheduler{}
	if err := Insert(db, s); err != nil {
		t.Fatal(err)
	}
}
