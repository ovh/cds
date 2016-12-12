package scheduler

import (
	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// LoadAllPipelineScheduler retrieves all pipeline scheduler from database
func LoadAllPipelineScheduler(db gorp.SqlExecutor) ([]sdk.PipelineScheduler, error) {
	s := []database.PipelineScheduler{}
	if _, err := db.Select(&s, "select * from pipeline_scheduler"); err != nil {
		log.Warning("LoadAllPipelineScheduler> Unable to load worker models : %T %s", err, err)
		return nil, err
	}
	ps := []sdk.PipelineScheduler{}
	for i := range s {
		if err := s[i].PostSelect(db); err != nil {
			return nil, err
		}
		ps = append(ps, sdk.PipelineScheduler(s[i]))
	}
	return ps, nil
}
