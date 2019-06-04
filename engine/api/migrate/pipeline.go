package migrate

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func CleanArtifactBuiltinActions(ctx context.Context, store cache.Store, DBFunc func() *gorp.DbMap) error {
	db := DBFunc()

	log.Info("migrate>CleanArtifactBuiltinActions> Start migration")

	var all []struct {
		ProjectKey   string `db:"projectkey"`
		PipelineName string `db:"name"`
	}

	query := `SELECT project.projectkey, pipeline.name FROM pipeline JOIN project on project.id = pipeline.project_id`
	if _, err := db.Select(&all, query); err != nil {
		return sdk.WithStack(err)
	}

	log.Info("migrate>CleanArtifactBuiltinActions> %d pipelines to migrate", len(all))
	for i := range all {
		if err := migratePipelineCleanArtifactBuiltinActions(ctx, db, store, all[i].ProjectKey, all[i].PipelineName); err != nil {
			log.Error("cannot migrate pipeline %s/%s: %v", all[i].ProjectKey, all[i].PipelineName, err)
			continue
		}
	}

	log.Info("migrate>End CleanArtifactBuiltinActions migration")
	return nil
}

func migratePipelineCleanArtifactBuiltinActions(ctx context.Context, db *gorp.DbMap, store cache.Store, projetKey, pipelineName string) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	pip, err := pipeline.LoadPipeline(tx, projetKey, pipelineName, true)
	if err != nil {
		return sdk.WithStack(err)
	}

	paramsDownload := []string{"path", "tag", "enabled", "pattern"}
	paramsUpload := []string{"path", "tag", "enabled", "destination"}

	for _, s := range pip.Stages {
		for _, j := range s.Jobs {
			for i := range j.Action.Actions {
				step := &j.Action.Actions[i]
				var paramIDXToRemove []int
				if step.Name == sdk.ArtifactDownload && step.Type == sdk.BuiltinAction { // Artifact download

					for ip := range step.Parameters {
						if !sdk.IsInArray(step.Parameters[ip].Name, paramsDownload) {
							paramIDXToRemove = append(paramIDXToRemove, ip)
						}
					}
				} else if step.Name == sdk.ArtifactUpload && step.Type == sdk.BuiltinAction { // Artifact upload

					for ip := range step.Parameters {
						if !sdk.IsInArray(step.Parameters[ip].Name, paramsUpload) {
							paramIDXToRemove = append(paramIDXToRemove, ip)
						}
					}
				}
				for _, id := range paramIDXToRemove { // Remove the deprecated params
					step.Parameters = append(step.Parameters[:id], step.Parameters[id+1:]...)
				}
			}
			if err := pipeline.UpdateJob(ctx, tx, &j); err != nil {
				return sdk.WithStack(err)
			}
		}
	}

	return sdk.WithStack(tx.Commit())
}
