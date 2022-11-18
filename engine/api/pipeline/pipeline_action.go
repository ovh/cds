package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/sdk"
)

// DeletePipelineActionByStage Delete all action from a stage
func DeletePipelineActionByStage(ctx context.Context, db gorp.SqlExecutor, stageID int64) error {
	pas, err := getPipelineActionsByStageID(ctx, db, stageID)
	if err != nil {
		return err
	}

	actionIDs := pipelineActionsToActionIDs(pas)

	if err := deletePipelineActionsByIDs(db, pipelineActionsToIDs(pas)); err != nil {
		return err
	}

	if err := action.DeleteAllTypeJoinedByIDs(db, actionIDs); err != nil {
		return err
	}

	return nil
}

// InsertJob  Insert a new Job ( pipeline_action + joinedAction )
func InsertJob(db gorp.SqlExecutor, job *sdk.Job, stageID int64, pip *sdk.Pipeline) error {
	// Insert Joined Action
	job.Action.Type = sdk.JoinedAction
	log.Debug(context.Background(), "InsertJob> Insert Action %s on pipeline %s with %d children", job.Action.Name, pip.Name, len(job.Action.Actions))
	if err := action.Insert(db, &job.Action); err != nil {
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
		log.Debug(context.Background(), "InsertJob> Creating stage %s on pipeline %s", stage.Name, pip.Name)
		if err := InsertStage(db, stage); err != nil {
			return sdk.WrapError(err, "cannot insert stage on pipeline %d", pip.ID)
		}
	} else {
		//Else load the stage
		var errLoad error
		stage, errLoad = LoadStage(db, pip.ID, stageID)
		if errLoad != nil {
			return errLoad
		}
		log.Debug(context.Background(), "InsertJob> Load existing stage %s on pipeline %s", stage.Name, pip.Name)
	}
	job.PipelineStageID = stage.ID

	// Create pipeline action
	query := `INSERT INTO pipeline_action (pipeline_stage_id, action_id, enabled) VALUES ($1, $2, $3) RETURNING id`
	return sdk.WithStack(db.QueryRow(query, job.PipelineStageID, job.Action.ID, job.Enabled).Scan(&job.PipelineActionID))
}

// UpdateJob  updates the job by actionData.PipelineActionID and actionData.ID
func UpdateJob(ctx context.Context, db gorp.SqlExecutor, job *sdk.Job) error {
	clearJoinedAction, err := action.LoadByID(ctx, db, job.Action.ID, action.LoadOptions.Default)
	if err != nil {
		return err
	}

	if clearJoinedAction.Type != sdk.JoinedAction {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	if err := UpdatePipelineAction(db, *job); err != nil {
		return err
	}
	job.Action.Enabled = job.Enabled
	return action.Update(db, &job.Action)
}

// DeleteJob Delete a job ( action + pipeline_action )
func DeleteJob(db gorp.SqlExecutor, job sdk.Job) error {
	if err := deletePipelineActionByActionID(db, job.Action.ID); err != nil {
		return err
	}
	return action.DeleteTypeJoinedByID(db, job.Action.ID)
}

// UpdatePipelineAction Update an action in a pipeline
func UpdatePipelineAction(db gorp.SqlExecutor, job sdk.Job) error {
	query := `UPDATE pipeline_action set action_id=$1, pipeline_stage_id=$2, enabled=$3 WHERE id=$4`
	_, err := db.Exec(query, job.Action.ID, job.PipelineStageID, job.Enabled, job.PipelineActionID)
	return sdk.WithStack(err)
}

// CheckJob validate a job
func CheckJob(ctx context.Context, db gorp.SqlExecutor, job *sdk.Job) error {
	t := time.Now()
	log.Debug(ctx, "CheckJob> Begin")
	defer log.Debug(ctx, "CheckJob> End (%d ns)", time.Since(t).Nanoseconds())
	errs := []sdk.Message{}
	//Check steps
	for i := range job.Action.Actions {
		step := &job.Action.Actions[i]
		log.Debug(ctx, "CheckJob> Checking step %s", step.Name)

		a, err := action.RetrieveForGroupAndName(ctx, db, step.Group, step.Name)
		if err != nil {
			if sdk.ErrorIs(err, sdk.ErrNoAction) {
				errs = append(errs, sdk.NewMessage(sdk.MsgJobNotValidActionNotFound, job.Action.Name, step.Name, i+1))
				continue
			}
			return err
		}
		job.Action.Actions[i].ID = a.ID
		if len(errs) > 0 {
			return sdk.MessagesToError(errs)
		}
	}

	return nil
}

// CountInPipelineData represents the result of CountInVarValue function
type CountInPipelineData struct {
	PipName   string
	StageName string
	JobName   string
	Count     int64
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
	FROM parent
	WHERE value LIKE $2
	GROUP BY pipName, stageName, actionName, id, child_id;
	`
	rows, err := db.Query(query, key, fmt.Sprintf("%%%s%%", element))
	if err != nil {
		return nil, sdk.WrapError(err, "unable to count usage")
	}
	defer rows.Close()

	results := []CountInPipelineData{}
	for rows.Next() {
		var d CountInPipelineData
		var id, childID int64
		if err := rows.Scan(&d.PipName, &d.StageName, &d.JobName, &id, &childID, &d.Count); err != nil {
			return nil, sdk.WrapError(err, "unable to scan")
		}
		results = append(results, d)
	}
	return results, nil
}
