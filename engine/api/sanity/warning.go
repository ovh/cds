package sanity

import (
	"sync"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/sdk"
)

// CheckProjectPipelines checks all pipelines in project
func CheckProjectPipelines(db *gorp.DbMap, store cache.Store, project *sdk.Project) error {
	// Load all pipelines
	pips, err := pipeline.LoadPipelines(db, project.ID, true, &sdk.User{Admin: true})
	if err != nil {
		return err
	}

	wg := &sync.WaitGroup{}
	for i := range pips {
		wg.Add(1)
		go func(p *sdk.Pipeline) {
			defer wg.Done()
			CheckPipeline(db, store, project, p)
		}(&pips[i])
	}

	wg.Wait()
	return nil
}

// CheckPipeline loads all PipelineAction and checks them all
func CheckPipeline(db *gorp.DbMap, store cache.Store, project *sdk.Project, pip *sdk.Pipeline) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, s := range pip.Stages {
		for _, j := range s.Jobs {
			warnings, err := CheckAction(tx, store, project, pip, j.Action.ID)
			if err != nil {
				return err
			}
			err = InsertActionWarnings(tx, project.ID, pip.ID, j.Action.ID, warnings)
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
