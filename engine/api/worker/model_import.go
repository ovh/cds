package worker

import (
	"database/sql"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// ParseAndImport parse and import an exportentities.WorkerModel
func ParseAndImport(db gorp.SqlExecutor, store cache.Store, eWorkerModel *exportentities.WorkerModel, force bool, u *sdk.AuthentifiedUser) (*sdk.Model, error) {
	sdkWm, errInvalidModel := eWorkerModel.GetWorkerModel()
	gr, err := group.LoadGroupByName(db, sdkWm.Group.Name)
	if err != nil {
		return nil, sdk.WrapError(err, "Unable to get group %s", sdkWm.Group.Name)
	}
	sdkWm.Group = *gr
	sdkWm.GroupID = gr.ID

	var modelPattern *sdk.ModelPattern
	if sdkWm.PatternName != "" {
		var errP error
		modelPattern, errP = LoadWorkerModelPatternByName(db, sdkWm.Type, sdkWm.PatternName)
		if errP != nil {
			return nil, sdk.WrapError(sdk.ErrWrongRequest, "Cannot load worker model pattern %s : %v", sdkWm.PatternName, errP)
		}
	}

	//User must be admin of the group set in the model
	var isGroupAdmin bool
currentUGroup:
	for _, g := range u.OldUserStruct.Groups {
		if g.ID == sdkWm.GroupID {
			for _, a := range g.Admins {
				if a.ID == u.OldUserStruct.ID {
					isGroupAdmin = true
					break currentUGroup
				}
			}
		}
	}

	//User should have the right permission or be admin
	if !u.Admin() && !isGroupAdmin {
		return nil, sdk.ErrWorkerModelNoAdmin
	}

	var badRequestError error
	asSimpleUser := !u.Admin() && !sdkWm.Restricted
	switch sdkWm.Type {
	case sdk.Docker:
		if sdkWm.ModelDocker.Image == "" {
			return nil, sdk.NewErrorFrom(sdk.ErrWrongRequest, "Invalid worker image")
		}
		if modelPattern != nil {
			sdkWm.ModelDocker.Cmd = modelPattern.Model.Cmd
			sdkWm.ModelDocker.Shell = modelPattern.Model.Shell
			sdkWm.ModelDocker.Envs = modelPattern.Model.Envs
		}
		if sdkWm.ModelDocker.Cmd == "" || sdkWm.ModelDocker.Shell == "" {
			badRequestError = sdk.NewErrorFrom(sdk.ErrWrongRequest, "Invalid worker command or invalid shell command")
		}
	default:
		if sdkWm.ModelVirtualMachine.Image == "" {
			return nil, sdk.NewErrorFrom(sdk.ErrWrongRequest, "Invalid worker image: cannot be empty")
		}
		if modelPattern != nil {
			sdkWm.ModelVirtualMachine.PreCmd = modelPattern.Model.PreCmd
			sdkWm.ModelVirtualMachine.Cmd = modelPattern.Model.Cmd
			sdkWm.ModelVirtualMachine.PostCmd = modelPattern.Model.PostCmd
		}

		if sdkWm.ModelVirtualMachine.Cmd == "" {
			badRequestError = sdk.NewErrorFrom(sdk.ErrWrongRequest, "Invalid worker command: Cannot be empty")
		}
	}

	if sdkWm.GroupID == 0 {
		return nil, sdk.NewErrorFrom(sdk.ErrWrongRequest, "groupID should be set")
	}

	if group.IsDefaultGroupID(sdkWm.GroupID) {
		return nil, sdk.NewErrorFrom(sdk.ErrWrongRequest, "this group can't be owner of a worker model")
	}

	// provision is allowed only for CDS Admin
	// or by currentUser with a restricted model
	if asSimpleUser {
		sdkWm.Provision = 0
	}

	if force {
		if existingWm, err := LoadWorkerModelByNameWithPassword(db, sdkWm.Name); err != nil {
			if sdk.Cause(err) == sql.ErrNoRows {
				if asSimpleUser && modelPattern == nil {
					return nil, sdk.ErrWorkerModelNoPattern
				}
				if errInvalidModel != nil {
					return nil, errInvalidModel
				}
				if badRequestError != nil {
					return nil, badRequestError
				}
				if errAdd := InsertWorkerModel(db, &sdkWm); errAdd != nil {
					return nil, sdk.WrapError(errAdd, "cannot add worker model %s", sdkWm.Name)
				}
			} else {
				return nil, sdk.WrapError(err, "cannot find worker model %s", sdkWm.Name)
			}
		} else {
			sdkWm.ID = existingWm.ID
			if asSimpleUser && modelPattern == nil {
				if existingWm.Type != sdk.Docker || existingWm.Restricted != sdkWm.Restricted { // Forbidden because we can't fetch previous user data
					return nil, sdk.WrapError(sdk.ErrWorkerModelNoPattern, "we can't fetch previous user data because type or restricted is different")
				}
				switch sdkWm.Type {
				case sdk.Docker:
					img := sdkWm.ModelDocker.Image
					sdkWm.ModelDocker = existingWm.ModelDocker
					sdkWm.ModelDocker.Image = img
				default:
					img := sdkWm.ModelVirtualMachine.Image
					flavor := sdkWm.ModelVirtualMachine.Flavor
					sdkWm.ModelVirtualMachine = existingWm.ModelVirtualMachine
					sdkWm.ModelVirtualMachine.Image = img
					sdkWm.ModelVirtualMachine.Flavor = flavor
				}
			}
			if sdkWm.ModelDocker.Password == sdk.PasswordPlaceholder {
				decryptedPw, err := DecryptValue(existingWm.ModelDocker.Password)
				if err != nil {
					return nil, sdk.WrapError(err, "cannot decrypt password")
				}
				sdkWm.ModelDocker.Password = decryptedPw
			}

			if !asSimpleUser {
				if errInvalidModel != nil {
					return nil, errInvalidModel
				}
				if badRequestError != nil {
					return nil, badRequestError
				}
			}

			if errU := UpdateWorkerModel(db, &sdkWm); errU != nil {
				return nil, sdk.WrapError(errU, "cannot update worker model %s", sdkWm.Name)
			}
		}
		return &sdkWm, nil
	}

	if asSimpleUser && modelPattern == nil {
		return nil, sdk.ErrWorkerModelNoPattern
	}
	if errInvalidModel != nil {
		return nil, errInvalidModel
	}
	if badRequestError != nil {
		return nil, badRequestError
	}
	if errAdd := InsertWorkerModel(db, &sdkWm); errAdd != nil {
		if errPG, ok := sdk.Cause(errAdd).(*pq.Error); ok && errPG.Code == gorpmapping.ViolateUniqueKeyPGCode {
			errAdd = sdk.ErrConflict
		}
		return nil, sdk.WrapError(errAdd, "cannot add worker model %s", sdkWm.Name)
	}

	// delete current cache of worker model after import
	store.DeleteAll(cache.Key("api:workermodels:*"))
	return &sdkWm, nil
}
