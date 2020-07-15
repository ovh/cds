package workermodel

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gorpmapping"
)

// Create returns a new worker model for given data.
func Create(ctx context.Context, db gorpmapping.SqlExecutorWithTx, data sdk.Model, ident sdk.Identifiable) (*sdk.Model, error) {
	// the default group cannot own worker model
	if group.IsDefaultGroupID(data.GroupID) {
		return nil, sdk.WrapError(sdk.ErrWrongRequest, "this group can't be owner of a worker model")
	}

	// check if worker model already exists
	existingModel, err := LoadByNameAndGroupID(ctx, db, data.Name, data.GroupID)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return nil, err
	}
	if existingModel != nil {
		return nil, sdk.NewErrorFrom(sdk.ErrModelNameExist, "worker model already exists with name %s for given group", data.Name)
	}

	// if a model pattern is given try to get it from database
	if data.PatternName != "" {
		modelPattern, err := LoadPatternByNameAndType(ctx, db, data.Type, data.PatternName)
		if err != nil {
			return nil, sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given worker model name"))
		}

		// set pattern data on given model
		switch data.Type {
		case sdk.Docker:
			data.ModelDocker.Cmd = modelPattern.Model.Cmd
			data.ModelDocker.Shell = modelPattern.Model.Shell
			data.ModelDocker.Envs = modelPattern.Model.Envs
		default:
			data.ModelVirtualMachine.PreCmd = modelPattern.Model.PreCmd
			data.ModelVirtualMachine.Cmd = modelPattern.Model.Cmd
			data.ModelVirtualMachine.PostCmd = modelPattern.Model.PostCmd
		}
	}

	// init new model from given data
	var model sdk.Model
	model.Update(data)
	model.Author.Username = ident.GetUsername()
	model.Author.Fullname = ident.GetFullname()
	model.Author.Email = ident.GetEmail()

	if err := Insert(ctx, db, &model); err != nil {
		return nil, sdk.WrapError(err, "cannot add worker model")
	}

	return &model, nil
}

// Update from given data.
func Update(ctx context.Context, db gorpmapping.SqlExecutorWithTx, old *sdk.Model, data sdk.Model) (*sdk.Model, error) {
	// the default group cannot own worker model
	if group.IsDefaultGroupID(data.GroupID) {
		return nil, sdk.WrapError(sdk.ErrWrongRequest, "this group can't be owner of a worker model")
	}

	if old.GroupID != data.GroupID || old.Name != data.Name {
		// check if worker model already exists
		existingModel, err := LoadByNameAndGroupID(ctx, db, data.Name, data.GroupID)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil, err
		}
		if existingModel != nil {
			return nil, sdk.NewErrorFrom(sdk.ErrModelNameExist, "worker model already exists with name %s for given group", data.Name)
		}
	}

	// if a model pattern is given try to get it from database
	if data.PatternName != "" {
		modelPattern, err := LoadPatternByNameAndType(ctx, db, data.Type, data.PatternName)
		if err != nil {
			return nil, sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given worker model name"))
		}

		// set pattern data on given model
		switch data.Type {
		case sdk.Docker:
			data.ModelDocker.Cmd = modelPattern.Model.Cmd
			data.ModelDocker.Shell = modelPattern.Model.Shell
			data.ModelDocker.Envs = modelPattern.Model.Envs
		default:
			data.ModelVirtualMachine.PreCmd = modelPattern.Model.PreCmd
			data.ModelVirtualMachine.Cmd = modelPattern.Model.Cmd
			data.ModelVirtualMachine.PostCmd = modelPattern.Model.PostCmd
		}
	}

	// update fields from request data
	model := sdk.Model(*old)
	model.Update(data)

	// model need to be register when modified
	model.UserLastModified = time.Now()
	model.NeedRegistration = true
	model.NbSpawnErr = 0
	model.LastSpawnErr = nil
	model.LastSpawnErrLogs = nil

	// update model in db
	if err := UpdateDB(ctx, db, &model); err != nil {
		return nil, sdk.WrapError(err, "cannot update worker model")
	}

	oldGrp, err := group.LoadByID(ctx, db, old.GroupID)
	if err != nil {
		return nil, err
	}
	grp, err := group.LoadByID(ctx, db, model.GroupID)
	if err != nil {
		return nil, err
	}

	oldPath, newPath := sdk.ComputeWorkerModelPath(oldGrp.Name, old.Name), sdk.ComputeWorkerModelPath(grp.Name, model.Name)
	// if the model has been renamed, we will have to update requirements
	if oldPath != newPath {
		// select requirements to update
		rs, err := action.GetRequirementsTypeModelAndValueStartBy(ctx, db, oldPath)
		if err != nil {
			return nil, err
		}

		// try to migrate each requirement
		for i := range rs {
			newValue := fmt.Sprintf("%s%s", newPath, strings.TrimPrefix(rs[i].Value, oldPath))
			rs[i].Name = newValue
			rs[i].Value = newValue
			if err := action.UpdateRequirement(db, &rs[i]); err != nil {
				return nil, err
			}
		}
	}

	return &model, nil
}

// CopyModelTypeData try to set missing type info for given model data.
func CopyModelTypeData(old, data *sdk.Model) error {
	if old.Restricted && !data.Restricted && data.PatternName == "" {
		return sdk.NewErrorFrom(sdk.ErrWorkerModelNoPattern, "a model script pattern should be given to set the model to not restricted")
	}

	// if model is not restricted and a pattern is not given, reuse old model info
	if !data.Restricted && data.PatternName == "" {
		if old.Type != data.Type {
			return sdk.NewErrorFrom(sdk.ErrWorkerModelNoPattern, "we can't fetch previous user data because type or restricted is different")
		}
		// set pattern data on given model
		switch data.Type {
		case sdk.Docker:
			data.ModelDocker.Cmd = old.ModelDocker.Cmd
			data.ModelDocker.Shell = old.ModelDocker.Shell
			data.ModelDocker.Envs = old.ModelDocker.Envs
		default:
			data.ModelVirtualMachine.PreCmd = old.ModelVirtualMachine.PreCmd
			data.ModelVirtualMachine.Cmd = old.ModelVirtualMachine.Cmd
			data.ModelVirtualMachine.PostCmd = old.ModelVirtualMachine.PostCmd
		}
	}

	return nil
}
