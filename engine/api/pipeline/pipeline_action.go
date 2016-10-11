package pipeline

import (
	"fmt"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/build"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// DeletePipelineActionByStage Delete all action from a stage
func DeletePipelineActionByStage(db database.QueryExecuter, stageID int64, userID int64) error {
	pipelineActionsID, err := selectAllPipelineActionID(db, stageID)
	if err != nil {
		return err
	}

	err = build.DeleteActionBuild(db, pipelineActionsID)
	if err != nil {
		return err
	}

	// For all pipeline_action in stage
	for i := range pipelineActionsID {
		var id int64
		var actionType string
		// Fetch id and type of action linked to pipeline_action so we can delete it if it's a joined action
		query := `SELECT action.id, action.type FROM action JOIN pipeline_action ON pipeline_action.action_id = action.id WHERE pipeline_action.id = $1`
		err = db.QueryRow(query, pipelineActionsID[i]).Scan(&id, &actionType)
		if err != nil {
			return err
		}

		// Delete pipeline_action
		query = `DELETE FROM pipeline_action WHERE id = $1`
		_, err = db.Exec(query, pipelineActionsID[i])
		if err != nil {
			return err
		}

		// Then if action is a Joined Action delete action as well
		if actionType != sdk.JoinedAction {
			continue
		}
		log.Info("DeletePipelineActionByStage> Deleting action %d\n", id)
		err = action.DeleteAction(db, id, userID)
		if err != nil {
			return err
		}
	}

	return nil
}

func selectAllPipelineActionID(db database.QueryExecuter, pipelineStageID int64) ([]int64, error) {
	var pipelineActionIDs []int64
	query := `SELECT id FROM "pipeline_action"
	 		  WHERE pipeline_stage_id = $1`
	rows, err := db.Query(query, pipelineStageID)
	if err != nil {
		return pipelineActionIDs, err
	}
	defer rows.Close()

	for rows.Next() {
		var pipelineActionID int64
		err = rows.Scan(&pipelineActionID)
		if err != nil {
			return pipelineActionIDs, err
		}
		pipelineActionIDs = append(pipelineActionIDs, pipelineActionID)
	}
	return pipelineActionIDs, nil
}

/*
//func loadPipelineAction(db database.Querier, stageData *sdk.Stage, args ...FuncArg) error {
func loadPipelineAction(db database.Querier, stageData *sdk.Stage) error {

	stageData.Actions = []sdk.Action{}
	query := `SELECT
			pipeline_action.id,
			action.id,
			pipeline_action.args,
			pipeline_action.enabled
		  FROM action
		  JOIN pipeline_action ON pipeline_action.action_id = action.id
      WHERE pipeline_action.pipeline_stage_id = $1 ORDER BY action.name,pipeline_action.id ASC`

	rows, err := db.Query(query, stageData.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var actions []sdk.Action
	var argss []string
	for rows.Next() {
		var args string
		var a sdk.Action
		err = rows.Scan(&a.PipelineActionID, &a.ID, &args, &a.Enabled)
		if err != nil {
			return err
		}
		actions = append(actions, a)
		argss = append(argss, args)
	}
	rows.Close()

	for index := range actions {
		var a *sdk.Action
		a, err = action.LoadActionByID(db, actions[index].ID)
		if err != nil {
			return fmt.Errorf("cannot LoadActionByID> %s", err)
		}
		a.Enabled = actions[index].Enabled
		a.PipelineStageID = stageData.ID
		a.PipelineActionID = actions[index].PipelineActionID

		var pipelineActionParameter []sdk.Parameter
		var isUpdated bool
		err = json.Unmarshal([]byte(argss[index]), &pipelineActionParameter)
		if err != nil {
			return err
		}

		for i := range a.Parameters {
			isUpdated, pipelineActionParameter = updateParamInList(pipelineActionParameter, a.Parameters[i])
			if !isUpdated {
				pipelineActionParameter = append(pipelineActionParameter, a.Parameters[i])
			}
		}
		a.Parameters = pipelineActionParameter

		stageData.Actions = append(stageData.Actions, *a)
	}

	return nil
}
*/

// InsertPipelineAction insert an action in a pipeline
func InsertPipelineAction(db database.QueryExecuter, projectKey, pipelineName string, actionID int64, args string, stageID int64) (int64, error) {
	query := `INSERT INTO pipeline_action (pipeline_stage_id, action_id, args, enabled) VALUES ($1, $2, $3, $4) RETURNING id`

	p, err := LoadPipeline(db, projectKey, pipelineName, true)
	if err != nil {
		return 0, fmt.Errorf("Cannot LoadPipeline> %s", err)
	}

	if stageID == 0 {
		newStage := sdk.Stage{
			Name:       fmt.Sprintf("Stage %d", len(p.Stages)+1),
			PipelineID: p.ID,
			BuildOrder: len(p.Stages) + 1,
			Enabled:    true,
		}
		err = InsertStage(db, &newStage)
		if err != nil {
			return 0, fmt.Errorf("Cannot InsertStage on pipeline %d> %s", p.ID, err)
		}
		stageID = newStage.ID
	}

	var pipelineActionID int64
	err = db.QueryRow(query, stageID, actionID, args, true).Scan(&pipelineActionID)
	if err != nil {
		return pipelineActionID, err
	}

	err = UpdatePipelineLastModified(db, p.ID)
	return pipelineActionID, err
}

// UpdatePipelineAction Update an action in a pipeline
func UpdatePipelineAction(db database.Executer, action sdk.Action, args string) error {
	query := `UPDATE pipeline_action set action_id=$1, args=$2, pipeline_stage_id=$3, enabled=$5  WHERE id=$4`

	_, err := db.Exec(query, action.ID, args, action.PipelineStageID, action.PipelineActionID, action.Enabled)
	if err != nil {
		return err
	}

	return nil
}

// DeletePipelineAction Delete an action in a pipeline
func DeletePipelineAction(db database.QueryExecuter, pipelineActionID int64) error {

	// Delete pipelineAction by buildOrder
	query := `DELETE FROM pipeline_action WHERE id = $1`
	_, err := db.Exec(query, pipelineActionID)
	if err != nil {
		return err
	}

	return nil
}
