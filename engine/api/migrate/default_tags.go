package migrate

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// DefaultTagsMigration useful to set default tags to git.branch git.author
func DefaultTagsMigration(store cache.Store, DBFunc func() *gorp.DbMap, u *sdk.User) {
	defaultTagsKey := "default_tags"
	db := DBFunc()

	log.Info("DefaultTagsMigration> Begin")

	projs, errP := project.LoadAll(db, store, u)
	if errP != nil {
		log.Warning("DefaultTagsMigration> Cannot load all project: %s", errP)
		return
	}

	for _, proj := range projs {
		workflows, err := workflow.LoadAll(db, proj.Key)
		if err != nil {
			log.Warning("DefaultTagsMigration> Cannot load all workflows for project %s : %s", proj.Key, err)
			continue
		}

		for _, wf := range workflows {
			tx, errTx := db.Begin()
			if errTx != nil {
				log.Warning("DefaultTagsMigration> cannot begin a transaction : %s", errTx)
				continue
			}

			var metadataStr sql.NullString
			if err := tx.SelectOne(&metadataStr, "SELECT metadata FROM workflow WHERE id = $1 FOR UPDATE NOWAIT", wf.ID); err != nil {
				tx.Rollback()
				log.Warning("DefaultTagsMigration> Cannot load metadata for workflow %s/%s : %s", proj.Key, wf.Name, err)
				continue
			}

			metadata := sdk.Metadata{}
			if err := gorpmapping.JSONNullString(metadataStr, &metadata); err != nil {
				tx.Rollback()
				log.Warning("DefaultTagsMigration> Cannot unmarshall metadata for workflow %s/%s : %s", proj.Key, wf.Name, err)
				continue
			}

			if metadata == nil || metadata[defaultTagsKey] == "" {
				nodeCtx, errLn := workflow.LoadNodeContext(db, store, proj.Key, wf.RootID, u, workflow.LoadOptions{})
				if errLn != nil {
					log.Warning("DefaultTagsMigration> Cannot load root node context for workflow %s/%s node id %d : %s", proj.Key, wf.Name, wf.RootID, errLn)
					tx.Rollback()
					continue
				}

				if nodeCtx.Application == nil || nodeCtx.Application.RepositoryFullname == "" {
					tx.Rollback()
					continue
				}

				metadata[defaultTagsKey] = "git.branch,git.author"

				if err := workflow.UpdateMetadata(tx, wf.ID, metadata); err != nil {
					tx.Rollback()
					log.Warning("DefaultTagsMigration> Cannot update metadata for workflow %s/%s node id %d : %s", proj.Key, wf.Name, wf.RootID, err)
					continue
				}

				if err := tx.Commit(); err != nil {
					tx.Rollback()
					log.Warning("DefaultTagsMigration> Cannot commit transaction for workflow %s/%s node id %d : %s", proj.Key, wf.Name, wf.RootID, err)
					continue
				}
			} else {
				tx.Rollback()
			}
		}
	}
	log.Info("DefaultTagsMigration> Done")
}
