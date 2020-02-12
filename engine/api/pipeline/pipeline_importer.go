package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//ImportUpdate import and update the pipeline in the project
func ImportUpdate(ctx context.Context, db gorp.SqlExecutor, proj *sdk.Project, pip *sdk.Pipeline, msgChan chan<- sdk.Message, u sdk.Identifiable) error {
	t := time.Now()
	log.Debug("ImportUpdate> Begin")
	defer log.Debug("ImportUpdate> End (%d ns)", time.Since(t).Nanoseconds())

	oldPipeline, err := LoadPipeline(ctx, db,
		proj.Key,
		pip.Name, true)
	if err != nil {
		return sdk.WrapError(err, "Unable to load pipeline %s %s", proj.Key, pip.Name)
	}

	if oldPipeline.FromRepository != "" && pip.FromRepository != oldPipeline.FromRepository {
		return sdk.WrapError(sdk.ErrPipelineAsCodeOverride, "unable to update as code pipeline %s/%s.", oldPipeline.FromRepository, pip.FromRepository)
	}

	// check that action used by job can be used by pipeline's project
	groupIDs := make([]int64, 0, len(proj.ProjectGroups)+1)
	groupIDs = append(groupIDs, group.SharedInfraGroup.ID)
	for i := range proj.ProjectGroups {
		groupIDs = append(groupIDs, proj.ProjectGroups[i].Group.ID)

	}

	rx := sdk.NamePatternSpaceRegex
	pip.ID = oldPipeline.ID
	for i := range pip.Stages {
		s := &pip.Stages[i]
		// stage name mandatory if there are many stages
		if len(pip.Stages) > 1 && !rx.MatchString(s.Name) {
			return sdk.NewError(sdk.ErrInvalidName, fmt.Errorf("Invalid stage name '%s'. It should match %s", s.Name, sdk.NamePatternSpace))
		}
		var stageFound bool
		var oldStage *sdk.Stage
		for _, os := range oldPipeline.Stages {
			if s.Name == os.Name {
				oldStage = &os
				stageFound = true
				break
			}
		}
		if !stageFound {
			//Insert stage
			log.Debug("Inserting stage %s", s.Name)
			s.PipelineID = pip.ID
			if err := InsertStage(db, s); err != nil {
				return sdk.WrapError(err, "Unable to insert stage %s in %s", s.Name, pip.Name)
			}
			//Insert stage's Jobs
			for x := range s.Jobs {
				jobAction := &s.Jobs[x]
				if errs := CheckJob(ctx, db, jobAction); errs != nil {
					log.Debug("CheckJob > %s", errs)
					return errs
				}
				if err := action.CheckChildrenForGroupIDs(ctx, db, &jobAction.Action, groupIDs); err != nil {
					return err
				}
				jobAction.PipelineStageID = s.ID
				jobAction.Action.Type = sdk.JoinedAction
				log.Debug("Creating job %s on stage %s on pipeline %s", jobAction.Action.Name, s.Name, pip.Name)
				if err := InsertJob(db, jobAction, s.ID, pip); err != nil {
					return sdk.WrapError(err, "Unable to insert job %s in %s", jobAction.Action.Name, pip.Name)
				}
				if msgChan != nil {
					msgChan <- sdk.NewMessage(sdk.MsgPipelineJobAdded, jobAction.Action.Name, s.Name)
				}
			}
			if msgChan != nil {
				msgChan <- sdk.NewMessage(sdk.MsgPipelineStageAdded, s.Name)
			}
		} else {
			//Update
			log.Debug("> Updating stage %s", oldStage.Name)
			msgChan <- sdk.NewMessage(sdk.MsgPipelineStageUpdating, oldStage.Name)
			msgChan <- sdk.NewMessage(sdk.MsgPipelineStageDeletingOldJobs, oldStage.Name)
			for x := range s.Jobs {
				jobAction := &s.Jobs[x]
				//Check the job
				if errs := CheckJob(ctx, db, jobAction); errs != nil {
					log.Debug(">> CheckJob > %s", errs)
					return errs
				}
				if err := action.CheckChildrenForGroupIDs(ctx, db, &jobAction.Action, groupIDs); err != nil {
					return err
				}
			}
			// Delete all existing jobs in existing stage
			for _, oj := range oldStage.Jobs {
				if err := DeleteJob(db, oj); err != nil {
					return sdk.WrapError(err, "unable to delete job %s in %s", oj.Action.Name, pip.Name)
				}
				msgChan <- sdk.NewMessage(sdk.MsgPipelineJobDeleted, oj.Action.Name, s.Name)
			}
			msgChan <- sdk.NewMessage(sdk.MsgPipelineStageInsertingNewJobs, oldStage.Name)
			// then insert job from yml into existing stage
			for x := range s.Jobs {
				j := &s.Jobs[x]
				//Insert the job
				j.PipelineStageID = oldStage.ID
				j.Action.Type = sdk.JoinedAction
				log.Debug(">> Creating job %s on stage %s on pipeline %s stageID: %d", j.Action.Name, s.Name, pip.Name, oldStage.ID)
				if err := InsertJob(db, j, oldStage.ID, pip); err != nil {
					return sdk.WrapError(err, "Unable to insert job %s in %s", j.Action.Name, pip.Name)
				}
				if msgChan != nil {
					msgChan <- sdk.NewMessage(sdk.MsgPipelineJobAdded, j.Action.Name, s.Name)
				}
			}

			if oldStage.BuildOrder != s.BuildOrder {
				s.ID = oldStage.ID
				if err := updateStageOrder(db, s.ID, s.BuildOrder); err != nil {
					return sdk.WrapError(err, "Unable to update stage %s", s.Name)
				}
			}

			//Update stage
			if msgChan != nil {
				msgChan <- sdk.NewMessage(sdk.MsgPipelineStageUpdated, s.Name)
			}
		}
	}

	//Check if we have to delete stages
	for _, os := range oldPipeline.Stages {
		var stageFound bool
		var currentStage sdk.Stage
		for _, s := range pip.Stages {
			if s.Name == os.Name {
				stageFound = true
				currentStage = s
				currentStage.ID = os.ID
				break
			}
		}
		if !stageFound {
			for x := range os.Jobs {
				j := os.Jobs[x]
				if err := DeleteJob(db, j); err != nil {
					return sdk.WrapError(err, "unable to delete job %s in %s", j.Action.Name, pip.Name)
				}
				if msgChan != nil {
					msgChan <- sdk.NewMessage(sdk.MsgPipelineJobDeleted, j.Action.Name, os.Name)
				}
			}
			if err := DeleteStageByID(ctx, db, &os); err != nil {
				return sdk.WrapError(err, "unable to delete stage %d", os.ID)
			}
			if msgChan != nil {
				msgChan <- sdk.NewMessage(sdk.MsgPipelineStageDeleted, os.Name)
			}
		} else {
			// Update stage
			if err := UpdateStage(db, &currentStage); err != nil {
				return sdk.WrapError(err, "cannot update stage %s (id=%d) for conditions, build_order and name", currentStage.Name, currentStage.ID)
			}
		}
	}

	for _, param := range pip.Parameter {
		found := false
		for _, oldParam := range oldPipeline.Parameter {
			if param.Name == oldParam.Name {
				found = true
				if err := UpdateParameterInPipeline(db, pip.ID, oldParam.Name, param); err != nil {
					return sdk.WrapError(err, "cannot update parameter %s", param.Name)
				}
				break
			}
		}
		if !found {
			if err := InsertParameterInPipeline(db, pip.ID, &param); err != nil {
				return sdk.WrapError(err, "cannot insert parameter %s", param.Name)
			}
		}
	}

	errU := UpdatePipeline(db, pip)

	return sdk.WrapError(errU, "ImportUpdate> cannot update pipeline")
}

//Import insert the pipeline in the project
func Import(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, pip *sdk.Pipeline, msgChan chan<- sdk.Message, u sdk.Identifiable) error {
	//Set projectID and Key in pipeline
	pip.ProjectID = proj.ID
	pip.ProjectKey = proj.Key

	//Check if pipeline exists
	ok, errExist := ExistPipeline(db, proj.ID, pip.Name)
	if errExist != nil {
		return sdk.WrapError(errExist, "Import> Unable to check if pipeline %s %s exists", proj.Name, pip.Name)
	}
	if !ok {
		if err := importNew(ctx, db, store, proj, pip, u); err != nil {
			log.Error(ctx, "pipeline.Import> %s", err)
			if msgChan != nil {
				msgChan <- sdk.NewMessage(sdk.MsgPipelineCreationAborted, pip.Name)
			}
			return sdk.WrapError(sdk.NewError(sdk.ErrInvalidPipeline, err), "unable to import new pipeline")
		}
		if msgChan != nil {
			msgChan <- sdk.NewMessage(sdk.MsgPipelineCreated, pip.Name)
		}
	}

	//Reload the pipeline
	pip2, err := LoadPipeline(ctx, db, proj.Key, pip.Name, false)
	if err != nil {
		return sdk.WrapError(err, "Unable to load imported pipeline project:%s pipeline:%s", proj.Name, pip.Name)
	}
	//Be confident: use the pipeline
	*pip = *pip2
	if ok {
		msgChan <- sdk.NewMessage(sdk.MsgPipelineExists, pip.Name)
	}

	return nil
}

func importNew(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, pip *sdk.Pipeline, u sdk.Identifiable) error {
	// check that action used by job can be used by pipeline's project
	groupIDs := make([]int64, 0, len(proj.ProjectGroups)+1)
	groupIDs = append(groupIDs, group.SharedInfraGroup.ID)
	for i := range proj.ProjectGroups {
		groupIDs = append(groupIDs, proj.ProjectGroups[i].Group.ID)
	}

	log.Debug("pipeline.importNew> Creating pipeline %s", pip.Name)
	//Insert pipeline
	if err := InsertPipeline(db, store, proj, pip); err != nil {
		return err
	}

	//Insert stages
	for i, s := range pip.Stages {
		log.Debug("pipeline.importNew> Creating stage %s on pipeline %s", s.Name, pip.Name)
		if s.BuildOrder == 0 {
			//Set default build order
			s.BuildOrder = i + 1
		}
		//Default is enabled
		s.Enabled = true
		//Set relation with pipeline
		s.PipelineID = pip.ID
		//Insert stage
		if err := InsertStage(db, &s); err != nil {
			return err
		}
		//Insert stage's Jobs
		for i := range s.Jobs {
			jobAction := &s.Jobs[i]
			jobAction.Enabled = true
			jobAction.Action.Enabled = true
			if errs := CheckJob(ctx, db, jobAction); errs != nil {
				log.Warning(ctx, "pipeline.importNew.CheckJob > %s", errs)
				return errs
			}
			if err := action.CheckChildrenForGroupIDs(ctx, db, &jobAction.Action, groupIDs); err != nil {
				return err
			}

			jobAction.PipelineStageID = s.ID
			log.Debug("pipeline.importNew> Creating job %s on stage %s on pipeline %s", jobAction.Action.Name, s.Name, pip.Name)
			if err := InsertJob(db, jobAction, s.ID, pip); err != nil {
				return err
			}
		}
	}

	return nil
}
