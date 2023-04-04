package pipeline

import (
	"context"
	"fmt"
	"strings"
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

		if step.Type != sdk.AsCodeAction {
			a, err := action.RetrieveForGroupAndName(ctx, db, step.Group, step.Name)
			if err != nil {
				if sdk.ErrorIs(err, sdk.ErrNoAction) {
					errs = append(errs, sdk.NewMessage(sdk.MsgJobNotValidActionNotFound, job.Action.Name, step.Name, i+1))
					continue
				}
				return err
			}
			job.Action.Actions[i].ID = a.ID

			// FIXME better check for params
			for x := range step.Parameters {
				sp := &step.Parameters[x]
				log.Debug(ctx, "CheckJob> Checking step parameter %s = %s", sp.Name, sp.Value)
				var found bool
				for y := range a.Parameters {
					ap := a.Parameters[y]
					if strings.ToLower(sp.Name) == strings.ToLower(ap.Name) {
						found = true
						break
					}
				}
				if !found {
					errs = append(errs, sdk.NewMessage(sdk.MsgJobNotValidInvalidActionParameter, job.Action.Name, sp.Name, i+1, step.Name))
				}
			}

		} else {
			ascodeAction, err := action.LoadAllByTypes(ctx, db, []string{sdk.AsCodeAction})
			if err != nil {
				return err
			}
			if len(ascodeAction) != 1 {
				return sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to find ascode action")
			}
			job.Action.Actions[i].ID = ascodeAction[0].ID
		}

		if len(errs) > 0 {
			return sdk.MessagesToError(errs)
		}
	}

	return nil
}
