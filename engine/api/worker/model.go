package worker

import (
	"database/sql"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// InsertWorkerModel insert a new worker model in database
func InsertWorkerModel(db gorp.SqlExecutor, model *sdk.Model) error {
	dbmodel := database.WorkerModel(*model)
	if err := db.Insert(&dbmodel); err != nil {
		return err
	}
	*model = sdk.Model(dbmodel)
	return nil
}

// UpdateWorkerModel update a worker model
func UpdateWorkerModel(db gorp.SqlExecutor, model sdk.Model) error {
	dbmodel := database.WorkerModel(model)
	if _, err := db.Update(&dbmodel); err != nil {
		return err
	}
	return nil
}

// LoadWorkerModels retrieves models from database
func LoadWorkerModels(db gorp.SqlExecutor) ([]sdk.Model, error) {
	ms := []database.WorkerModel{}
	if _, err := db.Select(&ms, "select * from worker_model order by name"); err != nil {
		log.Warning("Err>%T %s", err)
		return nil, err
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

// LoadWorkerModelByName retrieves a specific worker model in database
func LoadWorkerModelByName(db gorp.SqlExecutor, name string) (*sdk.Model, error) {
	m := database.WorkerModel(sdk.Model{})
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
	m := database.WorkerModel(sdk.Model{})
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

// LoadWorkerModelByGroup returns worker models for a group
func LoadWorkerModelByGroup(db gorp.SqlExecutor, groupID int64) ([]sdk.Model, error) {
	ms := []database.WorkerModel{}
	if _, err := db.Select(&ms, "select * from worker_model where group_id = $1 order by name", groupID); err != nil {
		return nil, err
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

// LoadWorkerModelByUser returns worker models list according to user's groups
func LoadWorkerModelByUser(db gorp.SqlExecutor, userID int64) ([]sdk.Model, error) {
	ms := []database.WorkerModel{}
	query := `	select * 
				from worker_model 
				where group_id in (select group_id from group_user where user_id = $1)
				union 
				select * from worker_model 
				where group_id in (select group_id from "group" where name = $2)
				order by name`
	if _, err := db.Select(&ms, query, userID, group.SharedInfraGroup); err != nil {
		return nil, err
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

//LoadSharedWorkerModel returns worker models with group shared.infra
func LoadSharedWorkerModel(db gorp.SqlExecutor) ([]sdk.Model, error) {
	ms := []database.WorkerModel{}
	if _, err := db.Select(&ms, `select * from worker_model where group_id in (select id from "group" where name = $1)`, group.SharedInfraGroup); err != nil {
		return nil, err
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
	m := database.WorkerModel(sdk.Model{ID: ID})
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
func LoadWorkerModelCapabilities(db database.Querier, workerID int64) ([]sdk.Requirement, error) {
	defer logTime("LoadWorkerModelCapabilities", time.Now())
	query := `SELECT name, type, argument FROM worker_capability WHERE worker_model_id = $1 ORDER BY name`

	rows, err := db.Query(query, workerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var capas []sdk.Requirement
	for rows.Next() {
		var c sdk.Requirement
		err = rows.Scan(&c.Name, &c.Type, &c.Value)
		if err != nil {
			return nil, err
		}
		capas = append(capas, c)
	}

	return capas, nil
}

func deleteAllWorkerCapabilities(db database.Executer, workerModelID int64) error {
	query := `DELETE FROM worker_capability WHERE worker_model_id = $1`
	_, err := db.Exec(query, workerModelID)
	return err

}

// DeleteWorkerModelCapability removes a capability from existing worker model
func DeleteWorkerModelCapability(db database.Executer, workerID int64, capaName string) error {
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

// UpdateWorkerModelCapability update a worker model capability
func UpdateWorkerModelCapability(db database.Executer, capa sdk.Requirement, modelID int64) error {
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
