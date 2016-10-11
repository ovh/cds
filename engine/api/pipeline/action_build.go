package pipeline

import (
	"database/sql"

	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/sdk"
)

func loadActionBuildsByStagePosition(db database.Querier, pipelineBuildID int64, stagePosition int) ([]sdk.ActionBuild, error) {
	actionBuilds := []sdk.ActionBuild{}
	query := `SELECT action_build.status, action_build.id, action.name
		  FROM action_build
		  JOIN pipeline_action ON pipeline_action.id = action_build.pipeline_action_id
		  JOIN action ON action.id = pipeline_action.action_id
		  JOIN pipeline_stage ON pipeline_stage.id = pipeline_action.pipeline_stage_id
		  WHERE action_build.pipeline_build_id = $1 and pipeline_stage.build_order = $2`
	rows, err := db.Query(query, pipelineBuildID, stagePosition)
	if err != nil {
		return actionBuilds, err
	}
	defer rows.Close()
	for rows.Next() {
		var actionBuild sdk.ActionBuild
		var status string
		err = rows.Scan(&status, &actionBuild.ID, &actionBuild.ActionName)
		if err != nil {
			return actionBuilds, err
		}
		actionBuild.Status = sdk.StatusFromString(status)
		actionBuilds = append(actionBuilds, actionBuild)
	}
	return actionBuilds, nil
}

// LoadActionStatus  Load status of action_build for the given pipeline_action
func LoadActionStatus(db database.Querier, pipelineActionID int64, pipelineBuildID int64) (sdk.Status, error) {
	query := `SELECT status FROM action_build WHERE pipeline_action_id = $1 AND pipeline_build_id = $2`
	var status string

	err := db.QueryRow(query, pipelineActionID, pipelineBuildID).Scan(&status)
	statusAction := sdk.StatusFromString(status)
	return statusAction, err
}

func loadStageAndActionBuilds(db database.Querier, pb *sdk.PipelineBuild) error {
	query := LoadPipelineBuildStage
	stagesRows, err := db.Query(query, pb.ID)
	if err != nil {
		return err
	}
	defer stagesRows.Close()

	stages := []sdk.Stage{}
	for stagesRows.Next() {
		var start, done pq.NullTime
		var pipelineActionID sql.NullInt64
		var buildOrder int
		var stagename string
		err = stagesRows.Scan(&start, &done, &pipelineActionID, &stagename, &buildOrder)
		if err != nil {
			return err
		}
		if len(stages) < buildOrder {
			stages = append(stages, sdk.Stage{
				Name:         stagename,
				BuildOrder:   buildOrder,
				ActionBuilds: []sdk.ActionBuild{},
			})
		}
		if start.Valid && done.Valid && pipelineActionID.Valid {
			stages[buildOrder-1].ActionBuilds = append(stages[buildOrder-1].ActionBuilds, sdk.ActionBuild{
				Start:            start.Time,
				Done:             done.Time,
				PipelineActionID: pipelineActionID.Int64,
			})
		}
	}
	pb.Stages = stages
	return nil
}

func loadAllActionBuilds(db database.Querier, pipelineBuildID int64) ([]sdk.ActionBuild, error) {
	actionBuilds := []sdk.ActionBuild{}
	query := `SELECT action_build.status, action_build.id, action.name
		  FROM action_build
		  JOIN pipeline_action ON pipeline_action.id = action_build.pipeline_action_id
		  JOIN action ON action.id = pipeline_action.action_id
		  WHERE action_build.pipeline_build_id = $1`
	rows, err := db.Query(query, pipelineBuildID)
	if err != nil {
		return actionBuilds, err
	}
	defer rows.Close()
	for rows.Next() {
		var actionBuild sdk.ActionBuild
		var status string
		err = rows.Scan(&status, &actionBuild.ID, &actionBuild.ActionName)
		if err != nil {
			return actionBuilds, err
		}
		actionBuild.Status = sdk.StatusFromString(status)
		actionBuilds = append(actionBuilds, actionBuild)
	}
	return actionBuilds, nil
}
