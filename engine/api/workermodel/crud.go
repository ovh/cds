package workermodel

import (
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
)

// Create returns a new worker model for given data.
func Create(db gorp.SqlExecutor, data sdk.Model, ident sdk.Identifiable) (*sdk.Model, error) {
	// the default group cannot own worker model
	if group.IsDefaultGroupID(data.GroupID) {
		return nil, sdk.WrapError(sdk.ErrWrongRequest, "this group can't be owner of a worker model")
	}

	// check that the group exists and user is admin for group id
	grp, err := group.LoadGroupByID(db, data.GroupID)
	if err != nil {
		return nil, err
	}

	// check if worker model already exists
	if _, err := LoadByNameAndGroupID(db, data.Name, grp.ID); err == nil {
		return nil, sdk.NewErrorFrom(sdk.ErrModelNameExist, "worker model already exists with name %s for group %s", data.Name, grp.Name)
	}

	// if a model pattern is given try to get it from database
	if data.PatternName != "" {
		modelPattern, err := LoadPatternByName(db, data.Type, data.PatternName)
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

	// TODO refactor using audit
	model.CreatedBy = sdk.User{
		Email:    ident.GetEmail(),
		Username: ident.GetUsername(),
		Fullname: ident.GetFullname(),
	}

	if err := Insert(db, &model); err != nil {
		return nil, sdk.WrapError(err, "cannot add worker model")
	}

	return &model, nil
}

// Update from given data.
func Update(db gorp.SqlExecutor, old *sdk.Model, data sdk.Model) (*sdk.Model, error) {
	// the default group cannot own worker model
	if group.IsDefaultGroupID(data.GroupID) {
		return nil, sdk.WrapError(sdk.ErrWrongRequest, "this group can't be owner of a worker model")
	}

	grp, err := group.LoadGroupByID(db, data.GroupID)
	if err != nil {
		return nil, err
	}

	if old.GroupID != data.GroupID || old.Name != data.Name {
		// check that no worker model already exists for same group/name
		if _, err := LoadByNameAndGroupID(db, data.Name, grp.ID); err == nil {
			return nil, sdk.NewErrorFrom(sdk.ErrAlreadyExist, "an action already exists for given name on this group")
		}
	}

	// if a model pattern is given try to get it from database
	if data.PatternName != "" {
		modelPattern, err := LoadPatternByName(db, data.Type, data.PatternName)
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

	// if model type is docker and given password equals the place holder value, we will reuse the old password value
	if data.Type == sdk.Docker && data.ModelDocker.Password == sdk.PasswordPlaceholder {
		decryptedPw, err := secret.DecryptValue(old.ModelDocker.Password)
		if err != nil {
			return nil, sdk.WrapError(err, "cannot decrypt password old model password")
		}
		data.ModelDocker.Password = decryptedPw
	}

	// update fields from request data
	model := sdk.Model(*old)
	model.Update(data)

	// update model in db
	if err := UpdateDB(db, &model); err != nil {
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
func CopyModelTypeData(old, data *sdk.Model) error {
	// if model is not restricted and a pattern is not given, reuse old model info
	if !data.Restricted && data.PatternName == "" {
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
