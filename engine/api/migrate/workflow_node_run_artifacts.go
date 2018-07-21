package migrate

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk/log"
)

// WorkflowNodeRunArtifacts add ref into workflow_node_run_artifacts table
func WorkflowNodeRunArtifacts(store cache.Store, DBFunc func() *gorp.DbMap) {
	db := DBFunc()
	log.Info("WorkflowNodeRunArtifacts> Begin")

	wfrArtifacts := []struct {
		ID  int64          `db:"id"`
		Ref sql.NullString `db:"ref"`
		Tag string         `db:"tag"`
	}{}

	if _, err := db.Select(&wfrArtifacts, "SELECT id, ref, tag FROM workflow_node_run_artifacts WHERE ref IS NULL"); err != nil {
		log.Error("WorkflowNodeRunArtifacts> Cannot load workflow_node_run_artifacts : %v", err)
		return
	}

	for _, art := range wfrArtifacts {
		tx, errTx := db.Begin()
		if errTx != nil {
			log.Warning("WorkflowNodeRunArtifacts> cannot create a transaction : %v", errTx)
			continue
		}

		if _, err := tx.Select(&art, "SELECT id, ref, tag FROM workflow_node_run_artifacts WHERE id = $1 FOR UPDATE NOWAIT", art.ID); err != nil {
			log.Warning("WorkflowNodeRunArtifacts> cannot load single workflow node run artifact %d : %v", art.ID, err)
			continue
		}

		if art.Ref.Valid {
			_ = tx.Rollback()
			continue
		}

		if _, err := tx.Exec("UPDATE workflow_node_run_artifacts SET ref = tag WHERE id = $1", art.ID); err != nil {
			_ = tx.Rollback()
			log.Error("WorkflowNodeRunArtifacts> cannot update workflow node run artifact %d : %v", art.ID, err)
			continue
		}

		if err := tx.Commit(); err != nil {
			log.Warning("WorkflowNodeRunArtifacts> cannot commit tx for workflow_node_run_artifacts %d : %v", art.ID, err)
			_ = tx.Rollback()
		}
	}

	log.Info("WorkflowNodeRunArtifacts> Done")
}
