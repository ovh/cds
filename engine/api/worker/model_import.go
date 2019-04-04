package worker

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// ParseAndImport parse and import an exportentities.WorkerModel
func ParseAndImport(db gorp.SqlExecutor, store cache.Store, eWorkerModel *exportentities.WorkerModel, force bool, u *sdk.User) (*sdk.Model, error) {
	data := eWorkerModel.GetWorkerModel()

	// group name should be set
	if data.Group == nil {
		return nil, sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing group name")
	}

	// check that the user is admin on the given template's group
	grp, err := group.LoadGroup(db, data.Group.Name)
	if err != nil {
		return nil, sdk.NewError(sdk.ErrWrongRequest, err)
	}
	data.GroupID = grp.ID

	// validate worker model fields
	if err := data.IsValid(); err != nil {
		return nil, err
	}

	// check if a model already exists for given info, if exists but not force update returns an error
	old, err := LoadWorkerModelByNameAndGroupID(db, data.Name, grp.ID)
	if err != nil {
		// validate worker model type fields
		if err := data.IsValidType(); err != nil {
			return nil, err
		}

		return CreateModel(db, u, data)
	} else if force {
		if err := CopyModelTypeData(u, old, &data); err != nil {
			return nil, err
		}

		// validate worker model type fields
		if err := data.IsValidType(); err != nil {
			return nil, err
		}

		return UpdateModel(db, u, old, data)
	}

	return nil, sdk.NewErrorFrom(sdk.ErrModelNameExist, "worker model already exists with name %s for group %s", data.Name, grp.Name)
}
