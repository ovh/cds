package worker

import (
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
)

// CreateModel returns a new worker model for given data.
func CreateModel(db gorp.SqlExecutor, u *sdk.User, data sdk.Model) (*sdk.Model, error) {
	// the default group cannot own worker model
	if group.IsDefaultGroupID(data.GroupID) {
		return nil, sdk.WrapError(sdk.ErrWrongRequest, "this group can't be owner of a worker model")
	}

	// check that the group exists and user is admin for group id
	grp, err := group.LoadGroupByID(db, data.GroupID)
	if err != nil {
		return nil, err
	}
	if err := group.CheckUserIsGroupAdmin(grp, u); err != nil {
		return nil, err
	}

	// check if worker model already exists
	if _, err := LoadWorkerModelByNameAndGroupID(db, data.Name, grp.ID); err == nil {
		return nil, sdk.NewErrorFrom(sdk.ErrModelNameExist, "worker model already exists with name %s for group %s", data.Name, grp.Name)
	}

	// provision is allowed only for CDS Admin or by user with a restricted model
	if !u.Admin && !data.Restricted {
		data.Provision = 0
	}

	// if current user is not admin and model is not restricted, a pattern should be given
	if !u.Admin && !data.Restricted && data.PatternName == "" {
		return nil, sdk.NewErrorFrom(sdk.ErrWorkerModelNoPattern, "missing model pattern name")
	}

	// if a model pattern is given try to get it from database
	if data.PatternName != "" {
		modelPattern, err := LoadWorkerModelPatternByName(db, data.Type, data.PatternName)
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

	model.CreatedBy = sdk.User{
		Email:    u.Email,
		Username: u.Username,
		Admin:    u.Admin,
		Fullname: u.Fullname,
		ID:       u.ID,
		Origin:   u.Origin,
	}

	if err := InsertWorkerModel(db, &model); err != nil {
		return nil, sdk.WrapError(err, "cannot add worker model")
	}

	return &model, nil
}

// UpdateModel from given data.
func UpdateModel(db gorp.SqlExecutor, u *sdk.User, old *sdk.Model, data sdk.Model) (*sdk.Model, error) {
	// the default group cannot own worker model
	if group.IsDefaultGroupID(data.GroupID) {
		return nil, sdk.WrapError(sdk.ErrWrongRequest, "this group can't be owner of a worker model")
	}

	grp, err := group.LoadGroupByID(db, data.GroupID)
	if err != nil {
		return nil, err
	}

	if old.GroupID != data.GroupID || old.Name != data.Name {
		// check that the group exists and user is admin for group id
		if err := group.CheckUserIsGroupAdmin(grp, u); err != nil {
			return nil, err
		}

		// check that no worker model already exists for same group/name
		if _, err := LoadWorkerModelByNameAndGroupID(db, data.Name, grp.ID); err == nil {
			return nil, sdk.NewErrorFrom(sdk.ErrAlreadyExist, "an action already exists for given name on this group")
		}
	}

	// provision is allowed only for CDS Admin or by user with a restricted model
	if !u.Admin && !data.Restricted {
		data.Provision = 0
	}

	// if a model pattern is given try to get it from database
	if data.PatternName != "" {
		modelPattern, err := LoadWorkerModelPatternByName(db, data.Type, data.PatternName)
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

	// update model in db
	if err := UpdateWorkerModel(db, &model); err != nil {
		return nil, sdk.WrapError(err, "cannot update worker model")
	}

	oldPath, newPath := old.GetPath(old.Group.Name), model.GetPath(grp.Name)
	// if the model has been renamed, we will have to update requirements
	if oldPath != newPath {
		// select requirements to update
		rs, err := action.GetRequirementsTypeModelAndValueStartBy(db, oldPath)
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
func CopyModelTypeData(u *sdk.User, old, data *sdk.Model) error {
	// if current user is not admin and model is not restricted and a pattern is not given, reuse old model info
	if !u.Admin && !data.Restricted && data.PatternName == "" {
		if old.Type != data.Type {
			return sdk.WrapError(sdk.ErrWorkerModelNoPattern, "we can't fetch previous user data because type or restricted is different")
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
