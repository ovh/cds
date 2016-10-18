package worker

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// ActionBuildInfo is returned to worker in answer to takeActionBuildHandler
type ActionBuildInfo struct {
	ActionBuild sdk.ActionBuild
	Action      sdk.Action
	Secrets     []sdk.Variable
}

// ErrNoWorker means the given worker ID is not found
var ErrNoWorker = fmt.Errorf("cds: no worker found")

// DeleteWorker remove worker from database
func DeleteWorker(db *sql.DB, id string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("DeleteWorker> Cannot start tx: %s\n", err)
	}
	defer tx.Rollback()

	query := `SELECT name, status, action_build_id FROM worker WHERE id = $1 FOR UPDATE`
	var st, name string
	var actionBuildID sql.NullInt64
	err = tx.QueryRow(query, id).Scan(&name, &st, &actionBuildID)
	if err != nil {
		log.Info("DeleteWorker> Cannot lock worker: %s\n", err)
		return nil
	}

	if st == sdk.StatusBuilding.String() {
		// AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHH
		// Worker is awol while building !
		// We need to restart this action before anyone notice
		if actionBuildID.Valid == false {
			return fmt.Errorf("DeleteWorker> Meh, worker %s crashed while building but action_build_id is NULL!\n", name)
		}

		log.Notice("Worker %s crashed while building %d !\n", name, actionBuildID.Int64)
		err = pipeline.RestartActionBuild(tx, actionBuildID.Int64)
		if err != nil {
			log.Critical("DeleteWorker> Cannot restart action build: %s\n", err)
		} else {
			log.Notice("DeleteWorker> ActionBuild %d restarted after crash\n", actionBuildID.Int64)
		}
	}

	// Well then, let's remove this loser
	query = `DELETE FROM worker WHERE id = $1`
	_, err = tx.Exec(query, id)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

// InsertWorker inserts worker representation into database
func InsertWorker(db database.Executer, w *sdk.Worker, userID int64, groupID int64) error {
	query := `INSERT INTO worker (id, name, last_beat, owner_id, model, status, hatchery_id, group_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := db.Exec(query, w.ID, w.Name, time.Now(), userID, w.Model, w.Status.String(), w.HatcheryID, groupID)
	return err
}

// LoadWorker retrieves worker in database
func LoadWorker(db database.Querier, id string) (*sdk.Worker, error) {
	w := &sdk.Worker{}
	var statusS string
	query := `SELECT id, name, last_beat, owner_id, model, status, hatchery_id, group_id FROM worker WHERE worker.id = $1 FOR UPDATE`

	err := db.QueryRow(query, id).Scan(&w.ID, &w.Name, &w.LastBeat, &w.OwnerID, &w.Model, &statusS, &w.HatcheryID, &w.GroupID)
	if err != nil {
		return nil, err
	}
	w.Status = sdk.StatusFromString(statusS)

	return w, nil
}

// LoadWorkersByModel load workers by model
func LoadWorkersByModel(db database.Querier, modelID int64) ([]sdk.Worker, error) {
	w := []sdk.Worker{}
	var statusS string
	query := `SELECT worker.id, worker.name, worker.last_beat, worker.owner_id, worker.model, worker.status, worker.hatchery_id,
						"user".username
	          FROM worker
	          JOIN "user" ON "user".id = worker.owner_id
	          WHERE worker.model = $1
	          ORDER BY worker.name ASC`

	rows, err := db.Query(query, modelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var worker sdk.Worker
		var user sdk.User

		err = rows.Scan(&worker.ID, &worker.Name, &worker.LastBeat, &worker.OwnerID, &worker.Model, &statusS, &worker.HatcheryID, &user.Username)
		if err != nil {
			return nil, err
		}
		worker.Status = sdk.StatusFromString(statusS)
		worker.Owner = user
		w = append(w, worker)
	}

	return w, nil
}

// LoadWorkers load all workers in db
func LoadWorkers(db *sql.DB) ([]sdk.Worker, error) {
	w := []sdk.Worker{}
	var statusS string
	query := `SELECT id, name, last_beat, owner_id, model, status, hatchery_id FROM worker WHERE 1 = 1 ORDER BY name ASC`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var worker sdk.Worker
		err = rows.Scan(&worker.ID, &worker.Name, &worker.LastBeat, &worker.OwnerID, &worker.Model, &statusS, &worker.HatcheryID)
		if err != nil {
			return nil, err
		}
		worker.Status = sdk.StatusFromString(statusS)
		w = append(w, worker)
	}

	return w, nil
}

// LoadDeadWorkers load worker with refresh last beat > timeout
func LoadDeadWorkers(db *sql.DB, timeout float64) ([]sdk.Worker, error) {
	var w []sdk.Worker
	var statusS string
	query := `	SELECT id, name, last_beat, owner_id, model, status, hatchery_id
				FROM worker 
				WHERE 1 = 1
				AND now() - last_beat > $1 * INTERVAL '1' SECOND
				ORDER BY name ASC
				LIMIT 10000`
	rows, err := db.Query(query, int64(math.Floor(timeout)))
	if err != nil {
		log.Warning("LoadDeadWorkers> Error querying workers")
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var worker sdk.Worker
		err = rows.Scan(&worker.ID, &worker.Name, &worker.LastBeat, &worker.OwnerID, &worker.Model, &statusS, &worker.HatcheryID)
		if err != nil {
			log.Warning("LoadDeadWorkers> Error scanning workers")
			return nil, err
		}
		worker.Status = sdk.StatusFromString(statusS)
		w = append(w, worker)
	}

	return w, nil
}

// RefreshWorker Update worker last_beat
func RefreshWorker(db *sql.DB, workerID string) error {
	query := `UPDATE worker SET last_beat = $1 WHERE id = $2`
	res, err := db.Exec(query, time.Now(), workerID)
	if err != nil {
		return err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if n != 1 {
		return fmt.Errorf("cds: cannot refresh worker '%s', not found", workerID)
	}

	return nil
}

func generateID() (string, error) {
	size := 64
	bs := make([]byte, size)
	_, err := rand.Read(bs)
	if err != nil {
		log.Critical("generateID: rand.Read failed: %s\n", err)
		return "", err
	}
	str := hex.EncodeToString(bs)
	token := []byte(str)[0:size]

	log.Debug("generateID: new generated id: %s\n", token)
	return string(token), nil
}

// RegistrationForm represents the arguments needed to register a worker
type RegistrationForm struct {
	Name               string
	UserKey            string
	Model              int64
	Hatchery           int64
	BinaryCapabilities []string
}

// RegisterWorker  Register new worker
func RegisterWorker(db *sql.DB, name string, uk string, modelID int64, hatcheryID int64, binaryCapabilities []string) (*sdk.Worker, error) {

	if name == "" {
		return nil, fmt.Errorf("cannot register worker with empty name")
	}

	if uk == "" {
		return nil, fmt.Errorf("cannot register worker with empty worker key")
	}

	/// Load token
	var userID, groupID int64
	var e sdk.Expiration
	t, err := LoadToken(db, uk)
	// /!\ LEGACY:  Load user key
	if err != nil && err == sql.ErrNoRows {
		log.Warning("RegisterWorker> cannot load token> %s\n", err)
		userID, e, err = LoadUserKey(db, uk)
		if err != nil && err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid worker key")
		}
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		groupID = t.GroupID
		e = t.Expiration
	}

	id, err := generateID()
	if err != nil {
		log.Warning("registerWorker: Cannot generate ID: %s\n", err)
		return nil, err
	}

	w := &sdk.Worker{
		ID:         id,
		Name:       name,
		Model:      modelID,
		HatcheryID: hatcheryID,
		Status:     sdk.StatusWaiting,
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	err = InsertWorker(tx, w, userID, groupID)
	if err != nil {
		log.Warning("registerWorker: Cannot insert worker in database: %s\n", err)
		return nil, err
	}

	if e == sdk.Session {
		err = DeleteUserKey(tx, uk)
		if err != nil {
			log.Warning("registerWorker> Cannot remove single use key: %s\n", err)
			return nil, err
		}
	}

	//If the worker is registered for a model and it gave us BinaryCapabilities...
	if len(binaryCapabilities) > 0 && modelID != 0 {
		go func() {
			//Start a new tx for this goroutine
			ntx, err := db.Begin()
			if err != nil {
				log.Warning("RegisterWorker> Unable to start a transaction : %s", err)
				return
			}
			defer ntx.Rollback()

			existingCapas, err := LoadWorkerModelCapabilities(ntx, modelID)
			if err != nil {
				log.Warning("RegisterWorker> Unable to load worker model capabilities : %s", err)
				return
			}

			var newCapas []string
			for _, b := range binaryCapabilities {
				var found bool
				for _, c := range existingCapas {
					if b == c.Value {
						found = true
						break
					}
				}
				if !found {
					newCapas = append(newCapas, b)
				}
			}
			if len(newCapas) > 0 {
				log.Notice("Updating model %d binary capabilities with %d capabilities", modelID, len(newCapas))
				for _, b := range newCapas {
					query := `insert into worker_capability (worker_model_id, name, argument, type) values ($1, $2, $3, $4)`
					if _, err := ntx.Exec(query, modelID, b, b, string(sdk.BinaryRequirement)); err != nil {
						//Ignore errors because we let the database to check constraints...
						log.Info("registerWorker> Cannot insert into worker_capability: %s\n", err)
						return
					}
				}
			}
			if err := ntx.Commit(); err != nil {
				log.Warning("RegisterWorker> Unable to commit transaction : %s", err)
			}
		}()
	}
	return w, tx.Commit()
}

// SetToBuilding sets action_build_id and status to building on given worker
func SetToBuilding(db database.Executer, workerID string, actionBuildID int64) error {
	query := `UPDATE worker SET status = $1, action_build_id = $2 WHERE id = $3`

	res, err := db.Exec(query, sdk.StatusBuilding.String(), actionBuildID, workerID)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected != 1 {
		return fmt.Errorf("SetActionBuild: Multiple (%d) rows affected ! (id=%s)\n", rowsAffected, workerID)
	}

	return nil
}

// UpdateWorkerStatus changes worker status to Disabled
func UpdateWorkerStatus(db database.Executer, workerID string, status sdk.Status) error {
	query := `UPDATE worker SET status = $1, action_build_id = NULL WHERE id = $2`

	res, err := db.Exec(query, status.String(), workerID)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected > 1 {
		log.Critical("UpdateWorkerStatus: Multiple (%d) rows affected ! (id=%s)\n", rowsAffected, workerID)
	}

	if rowsAffected == 0 {
		return ErrNoWorker
	}

	return nil
}

// FindBuildingWorker retrieves in database the worker building given actionBuildID
func FindBuildingWorker(db database.Querier, actionBuildID string) (string, error) {
	query := `SELECT id FROM worker WHERE action_build_id = $1`

	var id string
	err := db.QueryRow(query, actionBuildID).Scan(&id)
	return id, err
}
