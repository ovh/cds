package bootstrap

import (
	"database/sql"
	"encoding/json"

	"github.com/ovh/cds/engine/api/artifact"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

//InitiliazeDB inits the database
func InitiliazeDB(db *sql.DB) error {
	dbGorp := database.DBMap(db)
	if err := artifact.CreateBuiltinArtifactActions(dbGorp); err != nil {
		log.Critical("Cannot setup builtin Artifact actions: %s\n", err)
		return err
	}

	if err := group.CreateDefaultGlobalGroup(dbGorp); err != nil {
		log.Critical("Cannot setup default global group: %s\n", err)
		return err
	}

	if err := worker.CreateBuiltinActions(dbGorp); err != nil {
		log.Critical("Cannot setup builtin actions: %s\n", err)
		return err
	}

	if err := worker.CreateBuiltinEnvironments(dbGorp); err != nil {
		log.Critical("Cannot setup builtin environments: %s\n", err)
		return err
	}
	return nil
}

// MigratePipelineBuild Migrate pipeline_build
func MigratePipelineBuild(db *sql.DB) error {
	log.Notice("Start migrate pipeline build")
	dbGorp := database.DBMap(db)
	query := `
		SELECT id, pipeline_id
		FROM pipeline_build
		WHERE stages is null
	`

	rows, errQ := dbGorp.Query(query)
	if errQ != nil {
		log.Critical("Cannot query pipeline build: %s", errQ)
		return errQ
	}
	defer rows.Close()
	for rows.Next() {
		var pbID, pipID int64
		if err := rows.Scan(&pbID, &pipID); err != nil {
			log.Critical("Cannot scan pipeline build: %s", err)
			continue
		}

		//
		tx, errBegin := dbGorp.Begin()
		if errBegin != nil {
			log.Critical("Cannot start transaction: %s", errBegin)
			continue
		}

		var stagesNull sql.NullString
		queryForUpdate := "SELECT stages FROM pipeline_build WHERE id = $1 FOR UPDATE NOWAIT"
		if err := tx.QueryRow(queryForUpdate, pbID).Scan(&stagesNull); err != nil {
			log.Critical("Cannot Lock pb %d: %s", pbID, err)
			continue
		}

		if stagesNull.Valid {
			continue
		}

		p := sdk.Pipeline{
			ID: pipID,
		}

		if err := pipeline.LoadPipelineStage(tx, &p); err != nil {
			log.Critical("Cannot load pipeline stage: %s", err)
			continue
		}

		// Init Action build
		for stageIndex := range p.Stages {
			stage := &p.Stages[stageIndex]
			if stageIndex == 0 {
				stage.Status = sdk.StatusWaiting
			}
		}

		stages, errJSON := json.Marshal(p.Stages)
		if errJSON != nil {
			log.Critical("Cannot unmarshal pipeline stage: %s", errJSON)
			continue
		}

		queryInsertPipelineBuild := `
			UPDATE pipeline_build set stages = $1 WHERE id = $2
		`
		_, errUpdate := tx.Exec(queryInsertPipelineBuild, stages, pbID)
		if errUpdate != nil {
			log.Critical("Cannot update pipeline build %d: %s", pbID, errUpdate)
			continue
		}

		if err := tx.Commit(); err != nil {
			log.Critical("Cannot commit transaction", err)
			continue
		}
	}
	log.Notice("End migrate pipeline build")
	return nil
}
