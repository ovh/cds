package scheduler

import (
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/testwithdb"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLoadAllPipelineScheduler(t *testing.T) {
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	_db, _ := testwithdb.SetupPG(t)
	db := database.DBMap(_db)

	schedulers, err := LoadAllPipelineScheduler(db)
	assert.NoError(t, err)
	assert.NotNil(t, schedulers)
}
