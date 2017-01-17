package bootstrap

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func MigratePipelineHistory(_db *sql.DB) error {
	db := database.DBMap(_db)

	// Get all distinct app/pip/env/branch
	queryDistinct := `
		SELECT distinct application_id, pipeline_id, environment_id, vcs_changes_branch
		FROM pipeline_history_old
		ORDER by application_id, pipeline_id, environment_id, vcs_changes_branch
	`
	rows, errDistinct := db.Query(queryDistinct)
	if errDistinct != nil {
		log.Critical("MigratePipelineHistory>  Cannot select distinct pipeline history")
		return errDistinct
	}
	defer rows.Close()
	for rows.Next() {
		var appID, pipID, envID int64
		var branchName string
		if err := rows.Scan(&appID, &pipID, &envID, &branchName); err != nil {
			log.Critical("MigratePipelineHistory>  Cannot get rows for distinct pipeline history: %s", err)
			continue
		}

		// Select the 10 last
		querySelectByCriteria := `
			SELECT pipeline_build_id FROM pipeline_history_old
			WHERE application_id = $1 AND pipeline_id = $2 AND environment_id = $3 AND vcs_changes_branch = $4
			ORDER BY version DESC
			LIMIT 10
		`

		rowsSelectCriteria, errCriteria := db.Query(querySelectByCriteria, appID, pipID, envID, branchName)
		if errCriteria != nil {
			log.Critical("MigratePipelineHistory>  Cannot get pipeline history by criteria: %s", errCriteria)
			continue
		}

	rowsLoop:
		for rowsSelectCriteria.Next() {
			var pbHistoryID int64
			if err := rowsSelectCriteria.Scan(&pbHistoryID); err != nil {
				log.Critical("MigratePipelineHistory>  Cannot get pipeline history ID %s", errCriteria)
				continue
			}

			// Begin working on 1 pipHistory
			tx, errBegin := db.Begin()
			if errBegin != nil {
				log.Critical("MigratePipelineHistory>  Cannot start transaction: %s", errBegin)
				continue
			}

			// Get json DATA
			queryForUpdate := `SELECT data FROM pipeline_history_old WHERE pipeline_build_id = $1 FOR UPDATE NOWAIT`
			var data string
			if err := tx.QueryRow(queryForUpdate, pbHistoryID).Scan(&data); err != nil {
				log.Critical("MigratePipelineHistory>  Cannot select data from  pipeline history %d: %s", pbHistoryID, err)
				tx.Rollback()
				continue
			}

			// Unmarshal in pipeline BUILD struct
			var pb sdk.PipelineBuild
			if err := json.Unmarshal([]byte(data), &pb); err != nil {
				log.Critical("MigratePipelineHistory>  Cannot unmarshal pipeline History %d: %s", pbHistoryID, errCriteria)
				tx.Rollback()
				continue
			}

			// Check if pipeline build already exist
			queryCount := "SELECT count(1) FROM pipeline_build where id = $1"
			var nb int
			if err := tx.QueryRow(queryCount, pb.ID).Scan(&nb); err != nil {
				log.Critical("MigratePipelineHistory>  Cannot count pipeline build %d: %s", pbHistoryID, err)
				tx.Rollback()
				continue
			}
			if nb != 0 {
				// pb already migrated
				tx.Rollback()
				continue
			}

			// Start rebuilding stages struct

			var mapPB map[string]interface{}
			if err := json.Unmarshal([]byte(data), &mapPB); err != nil {
				log.Critical("MigratePipelineHistory>  Cannot unmarshal mapStringInterface pipeline History %d: %s", pbHistoryID, errCriteria)
				tx.Rollback()
				continue
			}

			if _, ok := mapPB["stages"]; !ok {
				log.Critical("MigratePipelineHistory>  No stages on pipeline build %d", pb.ID)
				tx.Rollback()
				continue
			}

			// Get stages
			if mapPB["stages"] != nil {

				for _, jsonStageString := range mapPB["stages"].([]interface{}) {
					stageString := jsonStageString.(map[string]interface{})

					sID := stageString["id"].(float64)

					// retrieve stage in Pipeline Build
					var stageToUpdate *sdk.Stage
					for i := range pb.Stages {
						if pb.Stages[i].ID == int64(sID) {
							stageToUpdate = &pb.Stages[i]
							stageToUpdate.PipelineBuildJobs = []sdk.PipelineBuildJob{}
							stageToUpdate.Jobs = []sdk.Job{}
						}
					}

					if stageToUpdate == nil {
						log.Critical("MigratePipelineHistory>  Cannot get stage to update %d", sID)
						tx.Rollback()
						continue rowsLoop
					}

					for _, buildString := range stageString["builds"].([]interface{}) {
						bString := buildString.(map[string]interface{})

						startTimeS := bString["start"].(string)
						doneTimeS := bString["done"].(string)

						start := time.Now()
						start.Format(startTimeS)
						done := time.Now()
						done.Format(doneTimeS)

						parameterJSON, errJSON := json.Marshal(bString["args"])
						if errJSON != nil {
							log.Critical("MigratePipelineHistory>  Cannot marshall parameters: %s", errJSON)
							tx.Rollback()
							continue rowsLoop
						}
						var parameters []sdk.Parameter
						if errParam := json.Unmarshal([]byte(parameterJSON), &parameters); errParam != nil {
							log.Critical("MigratePipelineHistory>  Cannot unmarshall parameters: %s", errParam)
							tx.Rollback()
							continue rowsLoop
						}

						pbJob := sdk.PipelineBuildJob{
							ID:              int64(bString["id"].(float64)),
							Parameters:      parameters,
							PipelineBuildID: pb.ID,
							Model:           0,
							Status:          bString["status"].(string),
							Job: sdk.Job{
								Action: sdk.Action{
									Name: bString["action_name"].(string),
								},
								Enabled:          true,
								PipelineActionID: int64(bString["pipeline_action_id"].(float64)),
							},
							Start: start,
							Done:  done,
						}
						stageToUpdate.Jobs = append(stageToUpdate.Jobs, pbJob.Job)
						stageToUpdate.PipelineBuildJobs = append(stageToUpdate.PipelineBuildJobs, pbJob)
					}
				}
			} else {
				pb.Stages = []sdk.Stage{}
			}

			args, errArgs := json.Marshal(pb.Parameters)
			if errArgs != nil {
				log.Critical("MigratePipelineHistory>  Cannot Marshal pb parameter: %s", errArgs)
				tx.Rollback()
				continue
			}
			var parentID int64
			if pb.PreviousPipelineBuild != nil {
				parentID = pb.PreviousPipelineBuild.ID
			}
			var userID int64
			if pb.Trigger.TriggeredBy != nil {
				userID = pb.Trigger.TriggeredBy.ID
			}

			// Calcul stage status
			for i := range pb.Stages {
				stage := &pb.Stages[i]
				finalStatus := sdk.StatusSuccess
				for _, pbJob := range stage.PipelineBuildJobs {
					if pbJob.Status == sdk.StatusFail.String() {
						finalStatus = sdk.StatusFail
						break
					}
				}
				stage.Status = finalStatus
			}

			stagesJSONByte, errSJSON := json.Marshal(pb.Stages)
			if errSJSON != nil {
				log.Critical("MigratePipelineHistory>  Cannot Marshal pb stages: %s", errSJSON)
				tx.Rollback()
				continue
			}

			queryInsert := `INSERT INTO pipeline_build (id, pipeline_id, build_number, version, status, args, start,
			application_id,environment_id, done, manual_trigger, triggered_by,
			parent_pipeline_build_id, vcs_changes_branch, vcs_changes_hash, vcs_changes_author,
			scheduled_trigger, stages)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`
			_, err := db.Exec(queryInsert, pb.ID, pb.Pipeline.ID, pb.BuildNumber, pb.Version, pb.Status.String(), args, pb.Start,
				pb.Application.ID, pb.Environment.ID, pb.Done, pb.Trigger.ManualTrigger, userID,
				parentID, pb.Trigger.VCSChangesBranch, pb.Trigger.VCSChangesHash, pb.Trigger.VCSChangesAuthor,
				pb.Trigger.ScheduledTrigger, stagesJSONByte)
			if err != nil {
				log.Critical("MigratePipelineHistory>  Cannot insert pipeline build: %s", err)
				tx.Rollback()
				continue
			}

			if err := tx.Commit(); err != nil {
				log.Critical("MigratePipelineHistory>  Cannot commit transaction: %s", err)
				tx.Rollback()
				continue
			}

		}
		rowsSelectCriteria.Close()

	}
	return nil
}
