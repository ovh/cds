package worker

import (
	"database/sql"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// InsertWorkerModel insert a new worker model in database
func InsertWorkerModel(db gorp.SqlExecutor, model *sdk.Model) error {
	dbmodel := WorkerModel(*model)
	if err := db.Insert(&dbmodel); err != nil {
		return err
	}
	*model = sdk.Model(dbmodel)
	return nil
}

// UpdateWorkerModel update a worker model
func UpdateWorkerModel(db gorp.SqlExecutor, model sdk.Model) error {
	dbmodel := WorkerModel(model)
	if _, err := db.Update(&dbmodel); err != nil {
		return err
	}
	return nil
}

// LoadWorkerModels retrieves models from database
func LoadWorkerModels(db gorp.SqlExecutor) ([]sdk.Model, error) {
	ms := []WorkerModel{}
	if _, err := db.Select(&ms, "select * from worker_model order by name"); err != nil {
		return nil, sdk.WrapError(err, "LoadWorkerModels> Unable to load worker models: %T", err)
	}
	models := []sdk.Model{}
	for i := range ms {
		if err := ms[i].PostSelect(db); err != nil {
			return nil, sdk.WrapError(err, "LoadWorkerModels> postSelect>")
		}
		models = append(models, sdk.Model(ms[i]))
	}
	return models, nil
}

// LoadWorkerModelByName retrieves a specific worker model in database
func LoadWorkerModelByName(db gorp.SqlExecutor, name string) (*sdk.Model, error) {
	m := WorkerModel(sdk.Model{})
	if err := db.SelectOne(&m, "select * from worker_model where name = $1", name); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNoWorkerModel
		}
		return nil, err
	}
	if err := m.PostSelect(db); err != nil {
		return nil, err
	}

	model := sdk.Model(m)
	return &model, nil
}

// LoadWorkerModelByID retrieves a specific worker model in database
func LoadWorkerModelByID(db gorp.SqlExecutor, ID int64) (*sdk.Model, error) {
	m := WorkerModel(sdk.Model{})
	if err := db.SelectOne(&m, "select * from worker_model where id = $1", ID); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNoWorkerModel
		}
		return nil, err
	}
	if err := m.PostSelect(db); err != nil {
		return nil, err
	}
	model := sdk.Model(m)
	return &model, nil
}

// LoadWorkerModelsByUser returns worker models list according to user's groups
func LoadWorkerModelsByUser(db gorp.SqlExecutor, user *sdk.User) ([]sdk.Model, error) {
	ms := []WorkerModel{}
	if user.Admin {
		query := `	select * from worker_model`
		if _, err := db.Select(&ms, query); err != nil {
			return nil, err
		}
	} else {
		query := `	select *
					from worker_model
					where group_id in (select group_id from group_user where user_id = $1)
					union
					select * from worker_model
					where group_id = $2
					order by name`
		if _, err := db.Select(&ms, query, user.ID, group.SharedInfraGroup.ID); err != nil {
			return nil, err
		}
	}

	models := []sdk.Model{}
	for i := range ms {
		if err := ms[i].PostSelect(db); err != nil {
			return nil, err
		}
		models = append(models, sdk.Model(ms[i]))
	}
	return models, nil
}

// DeleteWorkerModel removes from database worker model informations and all its capabilities
func DeleteWorkerModel(db gorp.SqlExecutor, ID int64) error {
	m := WorkerModel(sdk.Model{ID: ID})
	count, err := db.Delete(&m)
	if err != nil {
		return err
	}
	if count == 0 {
		return sdk.ErrNoWorkerModel
	}
	return nil
}

// LoadWorkerModelCapabilities retrieves capabilities of given worker model
func LoadWorkerModelCapabilities(db gorp.SqlExecutor, workerID int64) ([]sdk.Requirement, error) {
	query := `SELECT name, type, argument FROM worker_capability WHERE worker_model_id = $1 ORDER BY name`

	rows, err := db.Query(query, workerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var capas []sdk.Requirement
	for rows.Next() {
		var c sdk.Requirement
		if err := rows.Scan(&c.Name, &c.Type, &c.Value); err != nil {
			return nil, err
		}
		capas = append(capas, c)
	}
	return capas, nil
}

// DeleteWorkerModelCapability removes a capability from existing worker model
func DeleteWorkerModelCapability(db gorp.SqlExecutor, workerID int64, capaName string) error {
	query := `DELETE FROM worker_capability WHERE worker_model_id = $1 AND name = $2`

	res, err := db.Exec(query, workerID, capaName)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows <= 0 {
		return sdk.ErrNoWorkerModelCapa
	}

	return nil
}

// ComputeRegistrationNeeds checks if worker models need to be register
// if requirements contains "binary" type: all workers model need to be registered again by
// setting flag need_registration to true in DB.
func ComputeRegistrationNeeds(db gorp.SqlExecutor, allBinaryReqs []sdk.Requirement, reqs []sdk.Requirement) error {
	log.Debug("ComputeRegistrationNeeds>")
	for _, r := range reqs {
		if r.Type == sdk.BinaryRequirement {
			exist := false
			for _, e := range allBinaryReqs {
				if e.Value == r.Value {
					exist = true
					break
				}
			}
			if !exist {
				return updateAllToNeedRegistration(db)
			}
		}
	}
	return nil
}

func updateAllToNeedRegistration(db gorp.SqlExecutor) error {
	query := `UPDATE worker_model SET need_registration = $1`
	res, err := db.Exec(query, true)
	if err != nil {
		return sdk.WrapError(err, "updateAllToNeedRegistration>")
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return sdk.WrapError(err, "updateAllToNeedRegistration>")
	}
	log.Debug("updateAllToNeedRegistration> %d worker model(s) need registration", rows)
	return nil
}

// updateRegistration updates need_registration to false and last_registration time
func updateRegistration(db gorp.SqlExecutor, modelID int64) error {
	query := `UPDATE worker_model SET need_registration=$1, last_registration = $2 WHERE id = $3`
	res, err := db.Exec(query, false, time.Now(), modelID)
	if err != nil {
		return sdk.WrapError(err, "updateRegistration>")
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return sdk.WrapError(err, "updateRegistration>")
	}
	log.Debug("updateRegistration> %d worker model updated", rows)
	return nil
}

// UpdateWorkerModelCapability update a worker model capability
func UpdateWorkerModelCapability(db gorp.SqlExecutor, capa sdk.Requirement, modelID int64) error {
	query := `UPDATE worker_capability SET type=$1, argument=$2 WHERE worker_model_id = $3 AND name = $4`
	res, err := db.Exec(query, string(capa.Type), capa.Value, modelID, capa.Name)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows <= 0 {
		return sdk.ErrNoWorkerModelCapa
	}
	return nil
}
