package migrate

import (
	"fmt"

	"github.com/ovh/cds/engine/api/action"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type requirement struct {
	ID    int64
	Value string
}

// ActionModelRequirements adds group name for worker model not shared.infra on existing action's and job's requirements.
func ActionModelRequirements(store cache.Store, DBFunc func() *gorp.DbMap) error {
	db := DBFunc()

	log.Info("migrate>ActionModelRequirements> Start migration")

	// get all existing model from database
	wms, err := worker.LoadWorkerModelsNotSharedInfra(db)
	if err != nil {
		return err
	}

	log.Info("migrate>ActionModelRequirements> Found %d worker model", len(wms))

	// for each worker model try to migrate existing requirements
	for i := range wms {
		log.Info("migrate>ActionModelRequirements> Migrate requirements for model %s/%s (%d/%d)", wms[i].Group.Name, wms[i].Name, i+1, len(wms))
		if err := migrateActionRequirementForModel(db, wms[i]); err != nil {
			return err
		}
	}

	log.Info("migrate>ActionModelRequirements> End migration")
	return nil
}

func migrateActionRequirementForModel(db *gorp.DbMap, wm sdk.Model) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	// select and lock requirements to migrate for given model
	rs, err := action.GetRequirementsTypeModelAndValueStartBy(tx, wm.Name)
	if err != nil {
		return err
	}

	// if no requirements to migrate, stop migration
	if len(rs) == 0 {
		log.Info("migrate>ActionModelRequirements> Migrate requirements for model %s/%s - No action requirements to migrate", wm.Group.Name, wm.Name)
		return nil
	}
	log.Info("migrate>ActionModelRequirements> Migrate requirements for model %s/%s - Found %d action requirements to migrate", wm.Group.Name, wm.Name, len(rs))

	// try to migrate each requirement
	for i := range rs {
		newValue := fmt.Sprintf("%s/%s", wm.Group.Name, rs[i].Value)
		log.Info("migrate>ActionModelRequirements> Migrate requirements for model %s/%s - Update action requirement #%d (%d/%d) - %s -> %s", wm.Group.Name, wm.Name, rs[i].ID, i+1, len(rs), rs[i].Value, newValue)
		rs[i].Name = newValue
		rs[i].Value = newValue
		if err := action.UpdateRequirement(tx, &rs[i]); err != nil {
			return err
		}
	}

	return sdk.WithStack(tx.Commit())
}
