package pipeline

import (
	"context"
	"fmt"
	"github.com/lib/pq"
	"strings"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getJobs(ctx context.Context, db gorp.SqlExecutor, stagesIDs []int64) ([]sdk.Job, error) {

	// Load pipeline action table
	pipActions, err := getPipelineActionsByStageIDs(ctx, db, stagesIDs)
	if err != nil {
		return nil, err
	}
	jobIDs := make([]int64, len(pipActions))
	for i, pa := range pipActions {
		jobIDs[i] = pa.ActionID
	}

	// Load Joined Action
	rootAction, err := action.LoadByIDs(ctx, db, jobIDs, action.LoadOptions.WithRequirements, action.LoadOptions.WithEdge)
	if err != nil {
		return nil, err
	}

	// Load steps
	actionSteps, err := action.LoadActionsAndChildrenByActionJobIDs(ctx, db, jobIDs,
		action.LoadOptions.WithRequirements, action.LoadOptions.WithParameters,
		action.LoadOptions.WithGroup, action.LoadOptions.WithFlatChildren)
	if err != nil {
		return nil, err
	}

	// Links alls datas ( pipeline_action, joined action, steps )
	jobs := make([]sdk.Job, len(rootAction))
	for _, act := range rootAction {
		for _, pipAct := range pipActions {
			if pipAct.ActionID != act.ID {
				continue
			}
			job := sdk.Job{
				Action:           act,
				PipelineStageID:  pipAct.PipelineStageID,
				LastModified:     pipAct.LastModified.Unix(),
				Enabled:          pipAct.Enabled,
				PipelineActionID: pipAct.ID,
			}

			// add steps
			steps := make([]sdk.Action, 0)

			// Browser Root action child to find good action edge id and compute parameter
			for _, rootStepAction := range act.Actions {
				for i := range actionSteps {
					step := actionSteps[i]
					if rootStepAction.ActionEdgeID == step.ActionEdgeID {
						for j := range step.Parameters {
							for k := range rootStepAction.Parameters {
								if rootStepAction.Parameters[k].Name == step.Parameters[j].Name {
									step.Parameters[j].Value = rootStepAction.Parameters[k].Value
									break
								}
							}
						}
						steps = append(steps, step)
						break
					}
				}
			}
			job.Action.Actions = steps

			jobs = append(jobs, job)
			break
		}
	}
	return jobs, nil
}

func getPipelineActionsByStageIDs(ctx context.Context, db gorp.SqlExecutor, stagesIDs []int64) ([]pipelineAction, error) {
	var pas []pipelineAction
	query := gorpmapping.NewQuery(
		"SELECT * FROM pipeline_action WHERE pipeline_stage_id = ANY($1)",
	).Args(pq.Int64Array(stagesIDs))

	if err := gorpmapping.GetAll(ctx, db, query, &pas); err != nil {
		return nil, sdk.WrapError(err, "cannot get pipeline actions links for stages %v", stagesIDs)
	}
	return pas, nil
}

func getPipelineActionsByStageID(ctx context.Context, db gorp.SqlExecutor, stageID int64) ([]pipelineAction, error) {
	var pas []pipelineAction

	query := gorpmapping.NewQuery(
		"SELECT * FROM pipeline_action WHERE pipeline_stage_id = $1",
	).Args(stageID)
	if err := gorpmapping.GetAll(ctx, db, query, &pas); err != nil {
		return nil, sdk.WrapError(err, "cannot get pipeline action links for stage id %d", stageID)
	}

	return pas, nil
}

func deletePipelineActionsByIDs(db gorp.SqlExecutor, ids []int64) error {
	_, err := db.Exec(
		"DELETE FROM pipeline_action WHERE id = ANY(string_to_array($1, ',')::int[])",
		gorpmapping.IDsToQueryString(ids),
	)
	return sdk.WithStack(err)
}

func deletePipelineActionByActionID(db gorp.SqlExecutor, actionID int64) error {
	_, err := db.Exec("DELETE FROM pipeline_action WHERE action_id = $1", actionID)
	return sdk.WithStack(err)
}

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
func InsertJob(ctx context.Context, db gorp.SqlExecutor, job *sdk.Job, stageID int64, pip *sdk.Pipeline) error {
	// Insert Joined Action
	job.Action.Type = sdk.JoinedAction
	log.Debug("InsertJob> Insert Action %s on pipeline %s with %d children", job.Action.Name, pip.Name, len(job.Action.Actions))
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
		log.Debug("InsertJob> Creating stage %s on pipeline %s", stage.Name, pip.Name)
		if err := InsertStage(db, stage); err != nil {
			return sdk.WrapError(err, "cannot insert stage on pipeline %d", pip.ID)
		}
	} else {
		//Else load the stage
		var errLoad error
		stage, errLoad = LoadStage(ctx, db, pip.ID, stageID)
		if errLoad != nil {
			return errLoad
		}
		log.Debug("InsertJob> Load existing stage %s on pipeline %s", stage.Name, pip.Name)
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

//CheckJob validate a job
func CheckJob(ctx context.Context, db gorp.SqlExecutor, job *sdk.Job) error {
	t := time.Now()
	log.Debug("CheckJob> Begin")
	defer log.Debug("CheckJob> End (%d ns)", time.Since(t).Nanoseconds())
	errs := []sdk.Message{}
	//Check steps
	for i := range job.Action.Actions {
		step := &job.Action.Actions[i]
		log.Debug("CheckJob> Checking step %s", step.Name)

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
				errs = append(errs, sdk.NewMessage(sdk.MsgJobNotValidInvalidActionParameter, job.Action.Name, sp.Name, i+1, step.Name))
			}
		}

		if len(errs) > 0 {
			return sdk.MessagesToError(errs)
		}
	}

	return nil
}
