package pipeline

import (
	"encoding/json"
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

//InsertPipelineJob insert data in pipeline_action table
// DEPRECATED
func InsertPipelineJob(db database.QueryExecuter, pip *sdk.Pipeline, s *sdk.Stage, a *sdk.Action) error {
	query := `INSERT INTO pipeline_action (pipeline_stage_id, action_id, args, enabled) VALUES ($1, $2, $3, $4) RETURNING id`
	args, err := json.Marshal(a.Parameters)
	if err != nil {
		return err
	}
	if err := db.QueryRow(query, s.ID, a.ID, string(args), a.Enabled).Scan(&a.PipelineActionID); err != nil {
		return err
	}
	return nil
}

// InsertPipelineAction insert an action in a pipeline
func InsertPipelineAction(db database.QueryExecuter, projectKey, pipelineName string, actionID int64, args string, stageID int64) (int64, error) {
	p, err := LoadPipeline(db, projectKey, pipelineName, true)
	if err != nil {
		return 0, fmt.Errorf("Cannot LoadPipeline> %s", err)
	}

	var stage *sdk.Stage
	//Create stage if stageID == 0
	if stageID == 0 {
		stage = &sdk.Stage{
			Name:       fmt.Sprintf("Stage %d", len(p.Stages)+1),
			PipelineID: p.ID,
			BuildOrder: len(p.Stages) + 1,
			Enabled:    true,
		}
		if err := InsertStage(db, stage); err != nil {
			return 0, fmt.Errorf("Cannot InsertStage on pipeline %d> %s", p.ID, err)
		}
		stageID = stage.ID
	} else {
		//Else load the stage
		stage, err = LoadStage(db, p.ID, stageID)
		if err != nil {
			return 0, err
		}
	}

	//Reload action
	a, err := action.LoadActionByID(db, actionID)
	if err != nil {
		return 0, err
	}

	//Insert in pipeline_action table
	if err := InsertPipelineJob(db, p, stage, a); err != nil {
		return 0, err
	}

	return a.PipelineActionID, UpdatePipelineLastModified(db, p)
}

// InsertJob  Insert a new Job ( pipeline_action + joinedAction )
func InsertJob(db database.QueryExecuter, job *sdk.Job, stageID int64, pip *sdk.Pipeline) error {
	// Insert Joined Action
	job.Action.Type = sdk.JoinedAction
	job.Action.Enabled = true
	if err := action.InsertAction(db, &job.Action, false); err != nil {
		return err
	}

	// Create Stage if needed
	var stage *sdk.Stage
	var err error
	if stageID == 0 {
		stage = &sdk.Stage{
			Name:       fmt.Sprintf("Stage %d", len(pip.Stages)+1),
			PipelineID: pip.ID,
			BuildOrder: len(pip.Stages) + 1,
			Enabled:    true,
		}
		if err := InsertStage(db, stage); err != nil {
			return fmt.Errorf("Cannot InsertStage on pipeline %d> %s", pip.ID, err)
		}
	} else {
		//Else load the stage
		stage, err = LoadStage(db, pip.ID, stageID)
		if err != nil {
			return err
		}
	}
	job.PipelineStageID = stage.ID

	// Create pipeline action
	query := `INSERT INTO pipeline_action (pipeline_stage_id, action_id, args, enabled) VALUES ($1, $2, '[]', $4) RETURNING id`
	if err := db.QueryRow(query, job.PipelineStageID, job.Action.ID, job.Enabled).Scan(&job.PipelineActionID); err != nil {
		return err
	}
	return nil
}

// UpdateJob  updates the job by actionData.PipelineActionID and actionData.ID
func UpdateJob(db database.QueryExecuter, job *sdk.Job, userID int64) error {
	clearJoinedAction, err := action.LoadActionByID(db, job.Action.ID)
	if err != nil {
		return err
	}

	if clearJoinedAction.Type != sdk.JoinedAction {
		return sdk.ErrForbidden
	}

	query := `UPDATE pipeline_action set action_id=$1, pipeline_stage_id=$2, enabled=$4  WHERE id=$3`
	_, err = db.Exec(query, job.Action.ID, job.PipelineStageID, job.PipelineActionID, job.Enabled)
	if err != nil {
		return err
	}
	return action.UpdateActionDB(db, &job.Action, userID)
}

// DeleteJob Delete a job ( action + pipeline_action )
func DeleteJob(db database.QueryExecuter, actionID int64, userID int64) error {
	return action.DeleteAction(db, actionID, userID)
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
