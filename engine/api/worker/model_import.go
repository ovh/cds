package worker

import (
	"database/sql"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// ParseAndImport parse and import an exportentities.WorkerModel
func ParseAndImport(db gorp.SqlExecutor, eWorkerModel *exportentities.WorkerModel, force bool, u *sdk.User) (*sdk.Model, error) {
	sdkWm, err := eWorkerModel.GetWorkerModel()
	if err != nil {
		return nil, err
	}

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
	for _, g := range u.Groups {
		if g.ID == sdkWm.GroupID {
			for _, a := range g.Admins {
				if a.ID == u.ID {
					isGroupAdmin = true
					break currentUGroup
				}
			}
		}
	}

	//User should have the right permission or be admin
	if !u.Admin && !isGroupAdmin {
		return nil, sdk.ErrWorkerModelNoAdmin
	}

	switch sdkWm.Type {
	case sdk.Docker:
		if sdkWm.ModelDocker.Image == "" {
			return nil, sdk.WrapError(sdk.ErrWrongRequest, "Invalid worker image")
		}
		if !u.Admin && !sdkWm.Restricted {
			if modelPattern == nil {
				return nil, sdk.ErrWorkerModelNoPattern
			}
		}
		if sdkWm.ModelDocker.Cmd == "" || sdkWm.ModelDocker.Shell == "" {
			return nil, sdk.WrapError(sdk.ErrWrongRequest, "Invalid worker command or invalid shell command")
		}
	default:
		if sdkWm.ModelVirtualMachine.Image == "" {
			return nil, sdk.WrapError(sdk.ErrWrongRequest, "Invalid worker command or invalid image")
		}
		if !u.Admin && !sdkWm.Restricted {
			if modelPattern == nil {
				return nil, sdk.ErrWorkerModelNoPattern
			}
			sdkWm.ModelVirtualMachine.PreCmd = modelPattern.Model.PreCmd
			sdkWm.ModelVirtualMachine.Cmd = modelPattern.Model.Cmd
			sdkWm.ModelVirtualMachine.PostCmd = modelPattern.Model.PostCmd
		}
	}

	if sdkWm.GroupID == 0 {
		return nil, sdk.WrapError(sdk.ErrWrongRequest, "groupID should be set")
	}

	if group.IsDefaultGroupID(sdkWm.GroupID) {
		return nil, sdk.WrapError(sdk.ErrWrongRequest, "this group can't be owner of a worker model")
	}

	// provision is allowed only for CDS Admin
	// or by currentUser with a restricted model
	if !u.Admin && !sdkWm.Restricted {
		sdkWm.Provision = 0
	}

	if force {
		if existingWm, err := LoadWorkerModelByName(db, sdkWm.Name); err != nil {
			if sdk.Cause(err) == sql.ErrNoRows {
				if errAdd := InsertWorkerModel(db, &sdkWm); errAdd != nil {
					return nil, sdk.WrapError(errAdd, "cannot add worker model %s", sdkWm.Name)
				}
			} else {
				return nil, sdk.WrapError(err, "cannot find worker model %s", sdkWm.Name)
			}
		} else {
			sdkWm.ID = existingWm.ID
			if errU := UpdateWorkerModel(db, &sdkWm); errU != nil {
				return nil, sdk.WrapError(errU, "cannot update worker model %s", sdkWm.Name)
			}
		}
		return &sdkWm, nil
	}

	if errAdd := InsertWorkerModel(db, &sdkWm); errAdd != nil {
		if errPG, ok := sdk.Cause(errAdd).(*pq.Error); ok && errPG.Code == gorpmapping.ViolateUniqueKeyPGCode {
			errAdd = sdk.ErrConflict
		}
		return nil, sdk.WrapError(errAdd, "cannot add worker model %s", sdkWm.Name)
	}

	return &sdkWm, nil
}
