package pipeline

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// DeletePipelineActionByStage Delete all action from a stage
func DeletePipelineActionByStage(db gorp.SqlExecutor, stageID int64, userID int64) error {
	pipelineActionsID, errSelect := selectAllPipelineActionID(db, stageID)
	if errSelect != nil {
		return errSelect
	}

	// For all pipeline_action in stage
	for i := range pipelineActionsID {
		var id int64
		var actionType string
		// Fetch id and type of action linked to pipeline_action so we can delete it if it's a joined action
		query := `SELECT action.id, action.type FROM action JOIN pipeline_action ON pipeline_action.action_id = action.id WHERE pipeline_action.id = $1`
		if err := db.QueryRow(query, pipelineActionsID[i]).Scan(&id, &actionType); err != nil {
			return err
		}

		// Delete pipeline_action
		query = `DELETE FROM pipeline_action WHERE id = $1`
		if _, err := db.Exec(query, pipelineActionsID[i]); err != nil {
			return err
		}

		// Then if action is a Joined Action delete action as well
		if actionType != sdk.JoinedAction {
			continue
		}
		log.Debug("DeletePipelineActionByStage> Deleting action %d\n", id)
		if err := action.DeleteAction(db, id, userID); err != nil {
			return err
		}
	}

	return nil
}

func selectAllPipelineActionID(db gorp.SqlExecutor, pipelineStageID int64) ([]int64, error) {
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

// InsertJob  Insert a new Job ( pipeline_action + joinedAction )
func InsertJob(db gorp.SqlExecutor, job *sdk.Job, stageID int64, pip *sdk.Pipeline) error {
	// Insert Joined Action
	job.Action.Type = sdk.JoinedAction
	log.Debug("InsertJob> Insert Action %s on pipeline %s with %d children", job.Action.Name, pip.Name, len(job.Action.Actions))
	if err := action.InsertAction(db, &job.Action, false); err != nil {
		return err
	}

	// Create Stage if needed
	var stage *sdk.Stage
	if stageID == 0 {
		stage = &sdk.Stage{
			Name:       fmt.Sprintf("Stage %d", len(pip.Stages)+1),
			PipelineID: pip.ID,
			BuildOrder: len(pip.Stages) + 1,
			Enabled:    true,
		}
		log.Debug("InsertJob> Creating stage %s on pipeline %s", stage.Name, pip.Name)
		if err := InsertStage(db, stage); err != nil {
			return fmt.Errorf("Cannot InsertStage on pipeline %d> %s", pip.ID, err)
		}
	} else {
		//Else load the stage
		var errLoad error
		stage, errLoad = LoadStage(db, pip.ID, stageID)
		if errLoad != nil {
			return errLoad
		}
		log.Debug("InsertJob> Load existing stage %s on pipeline %s", stage.Name, pip.Name)
	}
	job.PipelineStageID = stage.ID

	// Create pipeline action
	query := `INSERT INTO pipeline_action (pipeline_stage_id, action_id, enabled) VALUES ($1, $2, $3) RETURNING id`
	return db.QueryRow(query, job.PipelineStageID, job.Action.ID, job.Enabled).Scan(&job.PipelineActionID)
}

// UpdateJob  updates the job by actionData.PipelineActionID and actionData.ID
func UpdateJob(db gorp.SqlExecutor, job *sdk.Job, userID int64) error {
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
	job.Action.Enabled = job.Enabled
	return action.UpdateActionDB(db, &job.Action, userID)
}

// DeleteJob Delete a job ( action + pipeline_action )
func DeleteJob(db gorp.SqlExecutor, job sdk.Job, userID int64) error {
	return action.DeleteAction(db, job.Action.ID, userID)
}

// UpdatePipelineAction Update an action in a pipeline
func UpdatePipelineAction(db gorp.SqlExecutor, job sdk.Job) error {
	query := `UPDATE pipeline_action set action_id=$1, pipeline_stage_id=$2, enabled=$4  WHERE id=$3`

	_, err := db.Exec(query, job.Action.ID, job.PipelineStageID, job.PipelineActionID, job.Enabled)
	if err != nil {
		return err
	}

	return nil
}

// DeletePipelineAction Delete an action in a pipeline
func DeletePipelineAction(db gorp.SqlExecutor, pipelineActionID int64) error {

	// Delete pipelineAction by buildOrder
	query := `DELETE FROM pipeline_action WHERE id = $1`
	_, err := db.Exec(query, pipelineActionID)
	if err != nil {
		return err
	}

	return nil
}

//CheckJob validate a job
func CheckJob(db gorp.SqlExecutor, job *sdk.Job) error {
	t := time.Now()
	log.Debug("CheckJob> Begin")
	defer log.Debug("CheckJob> End (%d ns)", time.Since(t).Nanoseconds())
	errs := new(sdk.Errors)
	//Check steps
	for i := range job.Action.Actions {
		step := &job.Action.Actions[i]
		log.Debug("CheckJob> Checking step %s", step.Name)
		a, err := action.LoadPublicAction(db, step.Name)
		if err != nil {
			if err == sdk.ErrNoAction {
				*errs = append(*errs, sdk.NewMessage(sdk.MsgJobNotValidActionNotFound, job.Action.Name, step.Name, i+1))
				continue
			}
			return sdk.WrapError(err, "CheckJob> Unable to load public action %s", step.Name)
		}

		a.Parameters, err = action.LoadActionParameters(db, a.ID)
		if err != nil {
			return sdk.WrapError(err, "CheckJob> Unable to load public action %s parameters", step.Name)
		}

		for x := range step.Parameters {
			sp := &step.Parameters[x]
			log.Debug("CheckJob> Checking step parameter %s = %s", sp.Name, sp.Value)
			var found bool
			for y := range a.Parameters {
				ap := a.Parameters[y]
				if strings.ToLower(sp.Name) == strings.ToLower(ap.Name) {
					found = true
					break
				}
			}
			if !found {
				*errs = append(*errs, sdk.NewMessage(sdk.MsgJobNotValidInvalidActionParameter, job.Action.Name, sp.Name, i+1, step.Name))
			}
		}

		//Set default values
		for y := range a.Parameters {
			ap := a.Parameters[y]
			var found bool
			for x := range step.Parameters {
				sp := &step.Parameters[x]
				if strings.ToLower(sp.Name) == strings.ToLower(ap.Name) {
					found = true
					break
				}
			}
			if !found {
				step.Parameters = append(step.Parameters, sdk.Parameter{
					Name:  ap.Name,
					Type:  ap.Type,
					Value: ap.Value,
				})
			}
		}

	}

	if len(*errs) > 0 {
		return errs
	}
	return nil
}

// GetPipelineIDFromJoinedActionID returns the pipeline id from any joined action
func GetPipelineIDFromJoinedActionID(db gorp.SqlExecutor, id int64) (int64, error) {
	query := `
	SELECT 	pipeline_stage.pipeline_id
	FROM 	pipeline_action, pipeline_stage
	WHERE 	pipeline_action.pipeline_stage_id = pipeline_stage.id
	AND 	pipeline_action.action_id = $1
	`
	id, err := db.SelectInt(query, id)
	if err != nil {
		return 0, sdk.WrapError(err, "GetPipelineIDFromJoinedActionID")
	}
	return id, nil
}

// CountInValueVarData represents the result of CountInVarValue function
type CountInPipelineData struct {
	PipName    string
	StageName  string
	ActionName string
	Count      int64
}

// CountInPipelines count how many times a text is used on all pipeline for the given project
func CountInPipelines(db gorp.SqlExecutor, key string, element string) ([]CountInPipelineData, error) {
	query := `
	WITH RECURSIVE parent(pipName, stageName, actionName, id, child_id) as (

		SELECT pipeline.name, pipeline_stage.name, action.name, action_edge.id as id, action_edge.child_id as child_id, action_edge_parameter.value
		FROM pipeline
		JOIN pipeline_stage on pipeline_stage.pipeline_id = pipeline.id
		JOIN pipeline_action on pipeline_action.pipeline_stage_id = pipeline_stage.id
		JOIN project on project.id = pipeline.project_id
		JOIN action on action.id = pipeline_action.action_id
		LEFT JOIN action_edge ON action_edge.parent_id = action.id
		LEFT JOIN action_edge_parameter on action_edge_parameter.action_edge_id = action_edge.id
		WHERE project.projectkey = $1 AND action_edge.id IS NOT NULL

		UNION

		SELECT p.pipName, p.stageName, p.actionName, c.id, c.child_id, action_edge_parameter.value FROM parent as p, action_edge as c
		LEFT JOIN action_edge_parameter ON action_edge_parameter.action_edge_id = c.id
		WHERE p.child_id = c.parent_id
	)
	SELECT pipName, stageName, actionName, id, child_id,
		count(*) as nb
	)
	FROM parent
	WHERE value LIKE '%$2%'
	GROUP BY pipName, stageName, actionName, id, child_id;
	`
	rows, err := db.Query(query, key, element)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []CountInPipelineData{}
	for rows.Next() {
		var d CountInPipelineData
		var id, childID int64
		if err := rows.Scan(&d.PipName, &d.StageName, &d.ActionName, &id, &childID, &d.Count); err != nil {
			return nil, err
		}
		results = append(results, d)
	}
	return results, nil
}
