package platform

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var (
	// BuiltinModels list available platform models
	BuiltinModels = []sdk.PlatformModel{
		sdk.KafkaPlatform,
	}
)

// CreateBuiltinModels creates platforms models
func CreateBuiltinModels(db *gorp.DbMap) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "CreateBuiltinModels> Unable to start transaction")
	}
	defer tx.Rollback()

	if _, err := tx.Exec("LOCK TABLE platform_model IN ACCESS EXCLUSIVE MODE"); err != nil {
		return sdk.WrapError(err, "CreateBuiltinModels> Unable to lock table")
	}

	for i := range BuiltinModels {
		p := &BuiltinModels[i]
		ok, err := ModelExists(tx, p.Name)
		if err != nil {
			return sdk.WrapError(err, "CreateModels")
		}

		if !ok {
			log.Debug("CreateBuiltinModels> inserting platform config: %s", p.Name)
			if err := InsertModel(tx, p); err != nil {
				return sdk.WrapError(err, "CreateBuiltinModels> error on insert")
			}
		} else {
			log.Debug("CreateBuiltinModels> updating platform config: %s", p.Name)
			oldM, err := LoadModelByName(tx, p.Name)
			if err != nil {
				return sdk.WrapError(err, "CreateBuiltinModels>  error on load")
			}
			p.ID = oldM.ID
			if err := UpdateModel(tx, p); err != nil {
				return sdk.WrapError(err, "CreateBuiltinModels>  error on update")
			}
		}
	}
	return tx.Commit()
}
