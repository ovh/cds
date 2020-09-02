package pipeline

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

// LoadStage Get a stage from its ID and pipeline ID
func LoadStage(db gorp.SqlExecutor, pipelineID int64, stageID int64) (*sdk.Stage, error) {
	query := `
		SELECT pipeline_stage.id, pipeline_stage.pipeline_id, pipeline_stage.name, pipeline_stage.build_order, pipeline_stage.enabled
		FROM pipeline_stage
		WHERE pipeline_stage.pipeline_id = $1
		AND pipeline_stage.id = $2;
		`

	var stage sdk.Stage
	rows, err := db.Query(query, pipelineID, stageID)
	if err == sql.ErrNoRows {
		return nil, sdk.WrapError(sdk.ErrNotFound, "stage does not exist")
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&stage.ID, &stage.PipelineID, &stage.Name, &stage.BuildOrder, &stage.Enabled); err != nil {
			return nil, sdk.WithStack(err)
		}
	}

	return &stage, nil
}

// InsertStage insert given stage into given database
func InsertStage(db gorp.SqlExecutor, s *sdk.Stage) error {
	query := `INSERT INTO "pipeline_stage" (pipeline_id, name, build_order, enabled) VALUES($1,$2,$3,$4) RETURNING id`

	if err := db.QueryRow(query, s.PipelineID, s.Name, s.BuildOrder, s.Enabled).Scan(&s.ID); err != nil {
		return err
	}
	return insertStageConditions(db, s)
}

// insertStageConditions insert prequisite for given stage in database
func insertStageConditions(db gorp.SqlExecutor, s *sdk.Stage) error {
	if s.Conditions.LuaScript != "" {
		s.Conditions.PlainConditions = nil
	}
	query := "UPDATE pipeline_stage SET conditions = $1 WHERE id = $2"

	conditionsBts, err := json.Marshal(s.Conditions)

	if err != nil {
		return sdk.WithStack(err)
	}
	if _, err := db.Exec(query, conditionsBts, s.ID); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

// LoadPipelineStage loads pipeline stage
func LoadPipelineStage(ctx context.Context, db gorp.SqlExecutor, p *sdk.Pipeline) error {
	_, end := telemetry.Span(ctx, "pipeline.LoadPipelineStage")
	defer end()

	p.Stages = []sdk.Stage{}

	query := `
	SELECT pipeline_stage_R.id as stage_id, pipeline_stage_R.pipeline_id, pipeline_stage_R.name, pipeline_stage_R.last_modified,
			pipeline_stage_R.build_order, pipeline_stage_R.enabled, pipeline_stage_R.conditions,
			pipeline_action_R.id as pipeline_action_id, pipeline_action_R.action_id, pipeline_action_R.action_last_modified,
			pipeline_action_R.action_args, pipeline_action_R.action_enabled
	FROM (
		SELECT pipeline_stage.id, pipeline_stage.pipeline_id,
				pipeline_stage.name, pipeline_stage.last_modified, pipeline_stage.build_order,
				pipeline_stage.enabled,
				pipeline_stage.conditions
		FROM pipeline_stage
		WHERE pipeline_id = $1
	) as pipeline_stage_R
	LEFT OUTER JOIN (
		SELECT pipeline_action.id, action.id as action_id, action.name as action_name, action.last_modified as action_last_modified,
				pipeline_action.args as action_args, pipeline_action.enabled as action_enabled,
				pipeline_action.pipeline_stage_id
		FROM action
		JOIN pipeline_action ON pipeline_action.action_id = action.id
	) as pipeline_action_R ON pipeline_action_R.pipeline_stage_id = pipeline_stage_R.id
	ORDER BY pipeline_stage_R.build_order, pipeline_action_R.action_name, pipeline_action_R.id ASC`

	rows, err := db.Query(query, p.ID)
	if err != nil {
		return sdk.WithStack(err)
	}
	defer rows.Close()

	mapStages := map[int64]*sdk.Stage{}
	mapAllActions := map[int64]*sdk.Job{}
	mapActionsStages := map[int64][]sdk.Job{}
	mapArgs := map[int64][]string{}
	stagesPtr := []*sdk.Stage{}

	for rows.Next() {
		var stageID, pipelineID int64
		var stageBuildOrder int
		var pipelineActionID, actionID sql.NullInt64
		var stageName string
		var stageConditions, actionArgs sql.NullString
		var stageEnabled, actionEnabled sql.NullBool
		var stageLastModified, actionLastModified pq.NullTime

		err = rows.Scan(
			&stageID, &pipelineID, &stageName, &stageLastModified,
			&stageBuildOrder, &stageEnabled, &stageConditions, &pipelineActionID, &actionID, &actionLastModified,
			&actionArgs, &actionEnabled)
		if err != nil {
			return sdk.WithStack(err)
		}

		//Stage
		stageData := mapStages[stageID]
		if stageData == nil {
			stageData = &sdk.Stage{
				ID:           stageID,
				PipelineID:   pipelineID,
				Name:         stageName,
				Enabled:      stageEnabled.Bool,
				BuildOrder:   stageBuildOrder,
				LastModified: stageLastModified.Time,
			}
			mapStages[stageID] = stageData
			stagesPtr = append(stagesPtr, stageData)
		}

		if err := gorpmapping.JSONNullString(stageConditions, &stageData.Conditions); err != nil {
			return sdk.WrapError(err, "cannot unmarshal stage conditions for stage id %d", stageID)
		}

		//Get actions
		if pipelineActionID.Valid && actionID.Valid && actionEnabled.Valid && actionLastModified.Valid {
			j := mapAllActions[pipelineActionID.Int64]
			if j == nil {
				j = &sdk.Job{
					PipelineStageID:  stageID,
					PipelineActionID: pipelineActionID.Int64,
					LastModified:     actionLastModified.Time.Unix(),
					Enabled:          actionEnabled.Bool,
					Action: sdk.Action{
						ID: actionID.Int64,
					},
				}
				mapAllActions[pipelineActionID.Int64] = j
				mapActionsStages[stageID] = append(mapActionsStages[stageID], *j)

				if actionArgs.Valid {
					mapArgs[stageID] = append(mapArgs[stageID], actionArgs.String)
				} else {
					mapArgs[stageID] = append(mapArgs[stageID], "[]")
				}
			}
		}
	}

	// load all steps
	var stepsIDs []int64
	for id := range mapStages {
		for index := range mapActionsStages[id] {
			stepsIDs = append(stepsIDs, mapActionsStages[id][index].Action.ID)
		}
	}
	as, err := action.LoadAllByIDs(ctx, db, stepsIDs,
		action.LoadOptions.WithParameters,
		action.LoadOptions.WithRequirements,
		action.LoadOptions.WithChildren,
	)
	if err != nil {
		return sdk.WrapError(err, "cannot load pipeline steps")
	}
	mSteps := make(map[int64]sdk.Action, len(as))
	for i := range as {
		mSteps[as[i].ID] = as[i]
	}

	//load job
	for id := range mapStages {
		for index := range mapActionsStages[id] {
			job := mapActionsStages[id][index]

			a := mSteps[mapActionsStages[id][index].Action.ID]

			var pipelineActionParameter []sdk.Parameter
			var isUpdated bool
			err = json.Unmarshal([]byte(mapArgs[id][index]), &pipelineActionParameter)
			if err != nil {
				return sdk.WithStack(err)
			}

			for i := range a.Parameters {
				isUpdated, pipelineActionParameter = updateParamInList(pipelineActionParameter, a.Parameters[i])
				if !isUpdated {
					pipelineActionParameter = append(pipelineActionParameter, a.Parameters[i])
				}
			}

			job.Action = a

			// Insert job also
			mapStages[id].Jobs = append(mapStages[id].Jobs, job)
		}
	}
	for _, s := range stagesPtr {
		p.Stages = append(p.Stages, *s)
	}

	return nil
}

// updateStageOrder update only Stage order
func updateStageOrder(db gorp.SqlExecutor, id int64, order int) error {
	query := `UPDATE pipeline_stage SET build_order=$1 WHERE id=$2`
	_, err := db.Exec(query, order, id)
	return sdk.WithStack(err)
}

// UpdateStage update Stage and all its prequisites
func UpdateStage(db gorp.SqlExecutor, s *sdk.Stage) error {
	query := `UPDATE pipeline_stage SET name=$1, build_order=$2, enabled=$3 WHERE id=$4`
	if _, err := db.Exec(query, s.Name, s.BuildOrder, s.Enabled, s.ID); err != nil {
		return err
	}

	//Insert all prequisites
	return insertStageConditions(db, s)
}

// DeleteStageByID Delete stage with associated pipeline action
func DeleteStageByID(ctx context.Context, tx gorp.SqlExecutor, s *sdk.Stage) error {
	nbOfStages, err := CountStageByPipelineID(tx, s.PipelineID)
	if err != nil {
		return err
	}

	if err := DeletePipelineActionByStage(ctx, tx, s.ID); err != nil {
		return err
	}

	if err := deleteStageByID(tx, s); err != nil {
		return err
	}

	return moveDownStages(tx, s.PipelineID, s.BuildOrder, nbOfStages)
}

func deleteStageByID(tx gorp.SqlExecutor, s *sdk.Stage) error {
	//Delete stage
	query := `DELETE FROM pipeline_stage WHERE id = $1`
	_, err := tx.Exec(query, s.ID)
	return sdk.WithStack(err)
}

// CountStageByPipelineID Count the number of stages for the given pipeline
func CountStageByPipelineID(db gorp.SqlExecutor, pipelineID int64) (int, error) {
	var countStages int
	query := `SELECT count(id) FROM "pipeline_stage"
	 		  WHERE pipeline_id = $1`
	err := db.QueryRow(query, pipelineID).Scan(&countStages)
	return countStages, sdk.WithStack(err)
}

func seleteAllStageID(db gorp.SqlExecutor, pipelineID int64) ([]int64, error) {
	var stageIDs []int64
	query := `SELECT id FROM "pipeline_stage"
	 		  WHERE pipeline_id = $1`

	rows, err := db.Query(query, pipelineID)
	if err != nil {
		return stageIDs, sdk.WithStack(err)
	}
	defer rows.Close()

	for rows.Next() {
		var stageID int64
		err = rows.Scan(&stageID)
		if err != nil {
			return stageIDs, sdk.WithStack(err)
		}
		stageIDs = append(stageIDs, stageID)
	}
	return stageIDs, nil
}

// DeleteAllStage  Delete all stages from pipeline ID
func DeleteAllStage(ctx context.Context, db gorp.SqlExecutor, pipelineID int64) error {
	stageIDs, err := seleteAllStageID(db, pipelineID)
	if err != nil {
		return err
	}

	for _, id := range stageIDs {
		if err := DeletePipelineActionByStage(ctx, db, id); err != nil {
			return err
		}
	}

	queryDelete := `DELETE FROM pipeline_stage WHERE pipeline_id = $1`
	_, err = db.Exec(queryDelete, pipelineID)
	return sdk.WithStack(err)
}

// MoveStage Move a stage
func MoveStage(db gorp.SqlExecutor, stageToMove *sdk.Stage, newBuildOrder int, p *sdk.Pipeline) error {
	if stageToMove.BuildOrder > newBuildOrder {
		if err := moveUpStages(db, stageToMove.PipelineID, stageToMove.BuildOrder, newBuildOrder); err != nil {
			return err
		}
	} else if stageToMove.BuildOrder < newBuildOrder {
		if err := moveDownStages(db, stageToMove.PipelineID, stageToMove.BuildOrder, newBuildOrder); err != nil {
			return err
		}
	}
	stageToMove.BuildOrder = newBuildOrder
	return UpdateStage(db, stageToMove)
}

func moveUpStages(db gorp.SqlExecutor, pipelineID int64, oldPosition, newPosition int) error {
	query := `UPDATE pipeline_stage
		  SET build_order=build_order+1
		  WHERE build_order < $1
		  AND build_order >= $2
		  AND pipeline_id = $3`
	_, err := db.Exec(query, oldPosition, newPosition, pipelineID)
	return sdk.WithStack(err)
}

func moveDownStages(db gorp.SqlExecutor, pipelineID int64, oldPosition, newPosition int) error {
	query := `UPDATE pipeline_stage
		  SET build_order=build_order-1
		  WHERE build_order <= $1
		  AND build_order > $2
		  AND pipeline_id = $3`
	_, err := db.Exec(query, newPosition, oldPosition, pipelineID)
	return sdk.WithStack(err)
}
