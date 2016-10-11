package worker

import (
	"database/sql"
	"time"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/sdk"
)

// InsertWorkerModel insert a new worker model in database
func InsertWorkerModel(db *sql.DB, model *sdk.Model) error {
	query := `INSERT INTO worker_model (type, name, image, owner_id) VALUES ($1, $2, $3, $4) RETURNING id`

	err := db.QueryRow(query, string(model.Type), model.Name, model.Image, model.OwnerID).Scan(&model.ID)
	if err != nil {
		return err
	}

	return nil
}

// LoadWorkerModels retrieves models from database
func LoadWorkerModels(db database.Querier) ([]sdk.Model, error) {
	query := `SELECT worker_model.id, worker_model.type, worker_model.name, worker_model.image, worker_model.owner_id , "user".username
	          FROM worker_model
	          JOIN "user" ON "user".id =  worker_model.owner_id
	          ORDER BY worker_model.name
	          LIMIT 1000`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []sdk.Model
	for rows.Next() {
		var m sdk.Model
		var u sdk.User
		var typeS string
		err = rows.Scan(&m.ID, &typeS, &m.Name, &m.Image, &m.OwnerID, &u.Username)
		if err != nil {
			return nil, err
		}
		switch typeS {
		case string(sdk.Docker):
			m.Type = sdk.Docker
			break
		case string(sdk.Openstack):
			m.Type = sdk.Openstack
			break
		case string(sdk.HostProcess):
			m.Type = sdk.HostProcess
			break
		}
		m.Owner = u
		models = append(models, m)
	}
	rows.Close()

	for i, m := range models {
		models[i].Capabilities, err = LoadWorkerModelCapabilities(db, m.ID)
		if err != nil {
			return nil, err
		}

		models[i].Worker, err = LoadWorkersByModel(db, m.ID)
		if err != nil {
			return nil, err
		}

	}

	return models, nil
}

// LoadWorkerModel retrieves a specific worker model in database
func LoadWorkerModel(db *sql.DB, name string) (*sdk.Model, error) {
	query := `SELECT worker_model.id, worker_model.type, worker_model.name, worker_model.image, worker_model.owner_id , "user".username
		  FROM worker_model
		  JOIN "user" ON "user".id = worker_model.owner_id
		  WHERE name = $1`

	var m sdk.Model
	var u sdk.User
	var typeS string
	err := db.QueryRow(query, name).Scan(&m.ID, &typeS, &m.Name, &m.Image, &m.OwnerID, &u.Username)
	if err != nil && err == sql.ErrNoRows {
		return nil, sdk.ErrNoWorkerModel
	}
	if err != nil {
		return nil, err
	}
	switch typeS {
	case string(sdk.Docker):
		m.Type = sdk.Docker
		break
	case string(sdk.Openstack):
		m.Type = sdk.Openstack
		break
	case string(sdk.HostProcess):
		m.Type = sdk.HostProcess
		break
	}
	m.Owner = u

	m.Capabilities, err = LoadWorkerModelCapabilities(db, m.ID)
	if err != nil {
		return nil, err
	}

	m.Worker, err = LoadWorkersByModel(db, m.ID)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// DeleteWorkerModel removes from database worker model informations and all its capabilities
func DeleteWorkerModel(tx *sql.Tx, workerModelID int64) error {

	// first, disable all workers of this model (CD-766)
	query := `UPDATE worker SET status = $1 WHERE model = $2`
	_, err := tx.Exec(query, string(sdk.StatusDisabled), workerModelID)
	if err != nil {
		return err
	}

	// then delete all worker model related info
	query = `DELETE FROM worker_capability WHERE worker_model_id = $1`
	_, err = tx.Exec(query, workerModelID)
	if err != nil {
		return err
	}

	query = `DELETE FROM worker_model WHERE id = $1`
	_, err = tx.Exec(query, workerModelID)
	if err != nil {
		return err
	}

	return nil
}

// InsertWorkerModelCapability adds a capability to an existing worker model
func InsertWorkerModelCapability(db *sql.DB, workerModelID int64, capa sdk.Requirement) error {
	query := `INSERT INTO worker_capability (worker_model_id, type, name, argument) VALUES ($1, $2, $3, $4)`

	_, err := db.Exec(query, workerModelID, string(capa.Type), capa.Name, capa.Value)
	if err != nil {
		return err
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
		var typeS string
		err = rows.Scan(&c.Name, &typeS, &c.Value)
		if err != nil {
			return nil, err
		}
		switch typeS {
		case string(sdk.BinaryRequirement):
			c.Type = sdk.BinaryRequirement
			break
		case string(sdk.NetworkAccessRequirement):
			c.Type = sdk.NetworkAccessRequirement
			break
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
func DeleteWorkerModelCapability(db *sql.DB, workerID int64, capaName string) error {
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
func UpdateWorkerModelCapability(db *sql.DB, capa sdk.Requirement, modelID int64) error {
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

// UpdateWorkerModel update a worker model
func UpdateWorkerModel(db *sql.DB, model sdk.Model) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `UPDATE worker_model SET type=$1, name=$2, image=$3 WHERE id = $4`
	_, err = tx.Exec(query, string(model.Type), model.Name, model.Image, model.ID)
	if err != nil {
		return err
	}

	// Disable all instances of this model spawned by an hatchery
	query = `UPDATE worker SET status = $1 WHERE model = $2 AND hatchery_id > 0 AND status != $3`
	_, err = tx.Exec(query, string(sdk.StatusDisabled), model.ID, string(sdk.StatusBuilding))
	if err != nil {
		return err
	}

	if len(model.Capabilities) > 0 {
		err = deleteAllWorkerCapabilities(db, model.ID)
		if err != nil {
			return err
		}
		for _, c := range model.Capabilities {
			err = InsertWorkerModelCapability(db, model.ID, c)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}
