package sanity

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
)

// CheckPipeline loads all PipelineAction and checks them all
func CheckPipeline(db *gorp.DbMap, store cache.Store, project *sdk.Project, pip *sdk.Pipeline) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, s := range pip.Stages {
		for _, j := range s.Jobs {
			_, err := CheckAction(tx, store, project, pip, j.Action.ID)
			if err != nil {
				return err
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return err
}
