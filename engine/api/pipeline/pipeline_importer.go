package pipeline

import (
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//ImportUpdate import and update the pipeline in the project
func ImportUpdate(db gorp.SqlExecutor, proj *sdk.Project, pip *sdk.Pipeline, msgChan chan<- sdk.Message, u *sdk.User) error {
	t := time.Now()
	log.Debug("ImportUpdate> Begin")
	defer log.Debug("ImportUpdate> End (%d ns)", time.Since(t).Nanoseconds())

	oldPipeline, err := LoadPipeline(db,
		proj.Key,
		pip.Name, true)
	if err != nil {
		return sdk.WrapError(err, "ImportUpdate> Unable to load pipeline %s %s", proj.Key, pip.Name)
	}

	pip.ID = oldPipeline.ID

	if len(pip.GroupPermission) > 0 {
		//Browse all new persmission to know if we had to insert of update
		for _, gp := range pip.GroupPermission {
			var gpFound bool
			for _, ogp := range oldPipeline.GroupPermission {
				if gp.Group.Name == ogp.Group.Name {
					gpFound = true
					if gp.Permission != ogp.Permission {
						//Update group permission
						g, err := group.LoadGroup(db, gp.Group.Name)
						if err != nil {
							return sdk.WrapError(err, "ImportUpdate> Unable to load group %s", gp.Group.Name)
						}
						if err := group.UpdateGroupRoleInPipeline(db, pip.ID, g.ID, gp.Permission); err != nil {
							return sdk.WrapError(err, "ImportUpdate> Unable to udapte group %s in %s", gp.Group.Name, pip.Name)
						}
						if msgChan != nil {
							msgChan <- sdk.NewMessage(sdk.MsgPipelineGroupUpdated, gp.Group.Name, pip.Name)
						}
					}
					break
				}
			}
			if !gpFound {
				//Insert group permission
				g, err := group.LoadGroup(db, gp.Group.Name)
				if err != nil {
					return sdk.WrapError(err, "ImportUpdate> Unable to load group %s", gp.Group.Name)
				}
				if err := group.InsertGroupInPipeline(db, pip.ID, g.ID, gp.Permission); err != nil {
					return sdk.WrapError(err, "ImportUpdate> Unable to insert group %s in %s", gp.Group.Name, pip.Name)
				}
				if msgChan != nil {
					msgChan <- sdk.NewMessage(sdk.MsgPipelineGroupAdded, gp.Group.Name, pip.Name)
				}
			}
		}

		//Browse all old persmission to know if we had to remove some of then
		for _, ogp := range oldPipeline.GroupPermission {
			var ogpFound bool
			for _, gp := range pip.GroupPermission {
				if gp.Group.Name == ogp.Group.Name {
					ogpFound = true
					break
				}
			}
			if !ogpFound {
				//Delete group permission
				if err := group.DeleteGroupFromPipeline(db, pip.ID, ogp.Group.ID); err != nil {
					return sdk.WrapError(err, "ImportUpdate> Unable to delete group %s in %s", ogp.Group.Name, pip.Name)
				}
				if msgChan != nil {
					msgChan <- sdk.NewMessage(sdk.MsgPipelineGroupDeleted, ogp.Group.Name, pip.Name)
				}
			}
		}
	}

	for i := range pip.Stages {
		s := &pip.Stages[i]
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
				return sdk.WrapError(err, "ImportUpdate> Unable to insert stage %s in %s", s.Name, pip.Name)
			}
			//Insert stage's Jobs
			for x := range s.Jobs {
				jobAction := &s.Jobs[x]
				if errs := CheckJob(db, jobAction); errs != nil {
					log.Debug("CheckJob > %s", errs)
					return errs
				}
				jobAction.PipelineStageID = s.ID
				jobAction.Action.Type = sdk.JoinedAction
				log.Debug("Creating job %s on stage %s on pipeline %s", jobAction.Action.Name, s.Name, pip.Name)
				if err := InsertJob(db, jobAction, s.ID, pip); err != nil {
					return sdk.WrapError(err, "ImportUpdate> Unable to insert job %s in %s", jobAction.Action.Name, pip.Name)
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
				if errs := CheckJob(db, jobAction); errs != nil {
					log.Debug(">> CheckJob > %s", errs)
					return errs
				}
			}
			// Delete all existing jobs in existing stage
			for _, oj := range oldStage.Jobs {
				if err := DeleteJob(db, oj, u.ID); err != nil {
					return sdk.WrapError(err, "ImportUpdate> Unable to delete job %s in %s", oj.Action.Name, pip.Name)
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
					return sdk.WrapError(err, "ImportUpdate> Unable to insert job %s in %s", j.Action.Name, pip.Name)
				}
				if msgChan != nil {
					msgChan <- sdk.NewMessage(sdk.MsgPipelineJobAdded, j.Action.Name, s.Name)
				}
			}

			if oldStage.BuildOrder != s.BuildOrder {
				s.ID = oldStage.ID
				if err := updateStageOrder(db, s.ID, s.BuildOrder); err != nil {
					return sdk.WrapError(err, "ImportUpdate> Unable to update stage %s", s.Name)
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
		for _, s := range pip.Stages {
			if s.Name == os.Name {
				stageFound = true
				break
			}
		}
		if !stageFound {
			for x := range os.Jobs {
				j := os.Jobs[x]
				if err := DeleteJob(db, j, u.ID); err != nil {
					return sdk.WrapError(err, "ImportUpdate> Unable to delete job %s in %s", j.Action.Name, pip.Name)
				}
				if msgChan != nil {
					msgChan <- sdk.NewMessage(sdk.MsgPipelineJobDeleted, j.Action.Name, os.Name)
				}
			}
			if err := DeleteStageByID(db, &os, u.ID); err != nil {
				return sdk.WrapError(err, "ImportUpdate> Unable to delete stage %d", os.ID)
			}
			if msgChan != nil {
				msgChan <- sdk.NewMessage(sdk.MsgPipelineStageDeleted, os.Name)
			}
		}
	}

	for _, param := range pip.Parameter {
		found := false
		for _, oldParam := range oldPipeline.Parameter {
			if param.Name == oldParam.Name {
				found = true
				if err := UpdateParameterInPipeline(db, pip.ID, oldParam.Name, param); err != nil {
					return sdk.WrapError(err, "ImportUpdate> cannot update parameter %s", param.Name)
				}
				break
			}
		}
		if !found {
			if err := InsertParameterInPipeline(db, pip.ID, &param); err != nil {
				return sdk.WrapError(err, "ImportUpdate> cannot insert parameter %s", param.Name)
			}
		}
	}

	pip.Type = oldPipeline.Type
	errU := UpdatePipeline(db, pip)

	return sdk.WrapError(errU, "ImportUpdate> cannot update pipeline")
}

//Import insert the pipeline in the project
func Import(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, pip *sdk.Pipeline, msgChan chan<- sdk.Message, u *sdk.User) error {
	//Set projectID and Key in pipeline
	pip.ProjectID = proj.ID
	pip.ProjectKey = proj.Key

	//Check if pipeline exists
	ok, errExist := ExistPipeline(db, proj.ID, pip.Name)
	if errExist != nil {
		return sdk.WrapError(errExist, "Import> Unable to check if pipeline %s %s exists", proj.Name, pip.Name)
	}
	if !ok {
		if err := importNew(db, store, proj, pip, u); err != nil {
			log.Error("pipeline.Import> %s", err)
			switch err.(type) {
			case *sdk.Errors:
				if msgChan != nil {
					msgChan <- sdk.NewMessage(sdk.MsgPipelineCreationAborted, pip.Name)
				}
				for _, m := range *err.(*sdk.Errors) {
					msgChan <- m
				}
				return sdk.ErrInvalidPipeline
			default:
				return sdk.WrapError(err, "pipeline.Import")
			}
		}
		if msgChan != nil {
			msgChan <- sdk.NewMessage(sdk.MsgPipelineCreated, pip.Name)
		}
	}
	//Reload the pipeline
	pip2, err := LoadPipeline(db, proj.Key, pip.Name, false)
	if err != nil {
		return sdk.WrapError(err, "Import> Unable to load imported pipeline", proj.Name, pip.Name)
	}
	//Be confident: use the pipeline
	*pip = *pip2
	if ok {
		msgChan <- sdk.NewMessage(sdk.MsgPipelineExists, pip.Name)
	}
	return nil
}

func importNew(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, pip *sdk.Pipeline, u *sdk.User) error {
	log.Debug("pipeline.importNew> Creating pipeline %s", pip.Name)
	//Insert pipeline
	if err := InsertPipeline(db, store, proj, pip, u); err != nil {
		return err
	}

	//If no GroupPermission provided, inherit from project
	if pip.GroupPermission == nil {
		pip.GroupPermission = proj.ProjectGroups
	}

	//Insert group permission
	if err := group.InsertGroupsInPipeline(db, pip.GroupPermission, pip.ID); err != nil {
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
			if errs := CheckJob(db, jobAction); errs != nil {
				log.Warning("pipeline.importNew.CheckJob > %s", errs)
				return errs
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
