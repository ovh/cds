package pipeline

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// LoadPipelineHistoryRequest  Load pipeline history request without json data
const LoadPipelineHistoryRequest = `
SELECT  ph.pipeline_id, ph.application_id, ph.environment_id, ph.pipeline_build_id, project.id as project_id,
	environment.name as envName, application.name as appName, pipeline.name as pipName, project.projectkey,
	pipeline.type,
	ph.build_number, ph.version, ph.status,
	ph.start, ph.done,
	ph.manual_trigger, ph.triggered_by, ph.parent_pipeline_build_id, ph.vcs_changes_branch, ph.vcs_changes_hash, ph.vcs_changes_author,
	"user".username, pipTriggerFrom.name as pipTriggerFrom, pbTriggerFrom.version as versionTriggerFrom
FROM pipeline_history ph
JOIN environment ON environment.id = ph.environment_id
JOIN application ON application.id = ph.application_id
JOIN pipeline ON pipeline.id = ph.pipeline_id
JOIN project ON project.id = pipeline.project_id
LEFT JOIN "user" ON "user".id = ph.triggered_by
LEFT JOIN pipeline_build as pbTriggerFrom ON pbTriggerFrom.id = ph.parent_pipeline_build_id
LEFT JOIN pipeline as pipTriggerFrom ON pipTriggerFrom.id = pbTriggerFrom.pipeline_id
%s
WHERE %s
ORDER BY start DESC
%s
`

// SelectBuildInHistory  load history of the given build
func SelectBuildInHistory(db database.Querier, pipelineID int64, applicationID int64, buildNumber int64, environmentID int64) (sdk.PipelineBuild, error) {
	var result sdk.PipelineBuild
	var data string

	query := `SELECT data FROM pipeline_history WHERE pipeline_id = $1 AND build_number = $2 AND application_id = $3 AND environment_id = $4`
	err := db.QueryRow(query, pipelineID, buildNumber, applicationID, environmentID).Scan(&data)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal([]byte(data), &result)
	if err != nil {
		return result, err
	}

	// TODO: DELETE ME when old build are discarded
	// No more passwords in database \o
	for si := range result.Stages {
		for abi := range result.Stages[si].ActionBuilds {
			for i, p := range result.Stages[si].ActionBuilds[abi].Args {
				if string(p.Type) == string(sdk.SecretVariable) {
					result.Stages[si].ActionBuilds[abi].Args[i].Value = sdk.PasswordPlaceholder
				}
			}
		}
	}

	return result, nil
}

// SelectBuildsInHistory  load history
func SelectBuildsInHistory(db database.Querier, pipelineID int64, applicationID int64, environmentID int64, limit int, status string) ([]sdk.PipelineBuild, error) {
	var results []sdk.PipelineBuild
	var rows *sql.Rows
	var err error

	query := `SELECT
			data
		  FROM pipeline_history
		  WHERE pipeline_id = $1 AND application_id = $2 AND environment_id = $3`
	if status != "" {
		query = fmt.Sprintf("%s %s", query, " AND status = $5")
	}
	query = fmt.Sprintf("%s %s", query, " ORDER BY start DESC LIMIT $4")

	if status != "" {
		rows, err = db.Query(query, pipelineID, applicationID, environmentID, limit, status)
	} else {
		rows, err = db.Query(query, pipelineID, applicationID, environmentID, limit)
	}
	if err != nil {
		return results, err
	}
	defer rows.Close()
	for rows.Next() {
		var data string
		err = rows.Scan(&data)
		if err != nil {
			return results, err
		}

		var pb sdk.PipelineBuild
		err = json.Unmarshal([]byte(data), &pb)
		if err != nil {
			return results, err
		}
		// TODO: DELETE ME when old build are discarded
		// No more passwords in database \o
		for si := range pb.Stages {
			for abi := range pb.Stages[si].ActionBuilds {
				for i, p := range pb.Stages[si].ActionBuilds[abi].Args {
					if string(p.Type) == string(sdk.SecretVariable) {
						pb.Stages[si].ActionBuilds[abi].Args[i].Value = sdk.PasswordPlaceholder
					}
				}
			}
		}
		results = append(results, pb)
	}
	return results, nil
}

func exist(db database.Querier, pipelineID int64, buildNumber int64, applicationID int64, environmentID int64) (bool, error) {

	query := `SELECT count(pipeline_id) FROM pipeline_history WHERE pipeline_id = $1 AND build_number = $2 AND application_id = $3 AND environment_id = $4`
	var nbItem int
	exist := false
	err := db.QueryRow(query, pipelineID, buildNumber, applicationID, environmentID).Scan(&nbItem)
	if err != nil {
		log.Warning("Cannot check if pipeline build history exist : %s", err)
		return exist, err
	}
	return nbItem > 0, nil
}

func updateHistory(db database.Executer, data string, status sdk.Status, pipelineID int64, buildNumber int64, applicationID int64, environmentID int64, version int64, done time.Time) error {

	query := `UPDATE pipeline_history SET status=$1, data=$2, done=$3, version=$4 WHERE pipeline_id=$5 AND build_number=$6 AND application_id = $7 AND environment_id = $8`
	_, err := db.Exec(query, string(status), data, done, version, pipelineID, buildNumber, applicationID, environmentID)
	return err
}

func insertHistory(db database.Executer, data string, pb sdk.PipelineBuild) error {
	var userID, pbParentID *int64
	if pb.Trigger.TriggeredBy != nil {
		userID = &pb.Trigger.TriggeredBy.ID
	}
	if pb.Trigger.ParentPipelineBuild != nil {
		pbParentID = &pb.Trigger.ParentPipelineBuild.ID
	}
	query := `INSERT INTO pipeline_history (
		pipeline_id, application_id,
		build_number, status,
		data,
		environment_id,
		version, done, manual_trigger,
		triggered_by, parent_pipeline_build_id,
		vcs_changes_branch, vcs_changes_hash, vcs_changes_author,
		start, pipeline_build_id) VALUES (
		$1, $2,
		$3, $4,
		$5,
		$6,
		$7, $8, $9,
		$10, $11,
		$12, $13, $14,
		$15, $16)`
	_, err := db.Exec(query,
		pb.Pipeline.ID, pb.Application.ID,
		pb.BuildNumber, string(pb.Status),
		data,
		pb.Environment.ID,
		pb.Version, pb.Done, pb.Trigger.ManualTrigger,
		userID, pbParentID,
		pb.Trigger.VCSChangesBranch, pb.Trigger.VCSChangesHash, pb.Trigger.VCSChangesAuthor,
		pb.Start, pb.ID,
	)
	return err
}

// SavePipelineBuildHistory Archive current build
func SavePipelineBuildHistory(db database.QueryExecuter, pb sdk.PipelineBuild) error {
	// TODO remove me, useless now
	// No need to have clear passwords in db
	for si := range pb.Stages {
		for abi := range pb.Stages[si].ActionBuilds {
			for i, p := range pb.Stages[si].ActionBuilds[abi].Args {
				if string(p.Type) == string(sdk.SecretVariable) {
					pb.Stages[si].ActionBuilds[abi].Args[i].Value = sdk.PasswordPlaceholder
				}
			}
		}
	}

	data, err := json.Marshal(pb)
	if err != nil {
		log.Warning("SavePipelineBuildHistory> Cannot marshal pipelineBuild: %s", err)
		return err
	}

	ex, err := exist(db, pb.Pipeline.ID, pb.BuildNumber, pb.Application.ID, pb.Environment.ID)
	if err != nil {
		log.Warning("Cannot check if history already exist: %s", err)
		return err
	}

	if ex {
		err = updateHistory(db, string(data), pb.Status, pb.Pipeline.ID, pb.BuildNumber, pb.Application.ID, pb.Environment.ID, pb.Version, pb.Done)
		if err != nil {
			log.Warning("SavePipelineBuildHistory> Cannot update pipelineBuild history: %s", err)
			return err
		}
	} else {
		err = insertHistory(db, string(data), pb)
		if err != nil {
			return fmt.Errorf("SavePipelineBuildHistory> Cannot insert history: %s", err)
		}
	}

	return nil
}
