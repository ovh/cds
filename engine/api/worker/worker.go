package worker

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/token"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// ErrNoWorker means the given worker ID is not found
var ErrNoWorker = fmt.Errorf("cds: no worker found")

// DeleteWorker remove worker from database
func DeleteWorker(db gorp.SqlExecutor, id string) error {
	query := `DELETE FROM worker WHERE id = $1`
	if _, err := db.Exec(query, id); err != nil {
		return sdk.WrapError(err, "DeleteWorker")
	}

	return nil
}

// InsertWorker inserts worker representation into database
func InsertWorker(db gorp.SqlExecutor, w *sdk.Worker, groupID int64) error {
	query := `INSERT INTO worker (id, name, last_beat, model, status, hatchery_name, group_id) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := db.Exec(query, w.ID, w.Name, time.Now(), w.ModelID, w.Status.String(), w.HatcheryName, groupID)
	return err
}

// LoadWorker retrieves worker in database
func LoadWorker(db gorp.SqlExecutor, id string) (*sdk.Worker, error) {
	w := &sdk.Worker{}
	var statusS string
	var pbJobID sql.NullInt64
	var jobType sql.NullString
	query := `SELECT id, action_build_id, job_type, name, last_beat, group_id, model, status, hatchery_name, group_id FROM worker WHERE worker.id = $1 FOR UPDATE`

	if err := db.QueryRow(query, id).Scan(&w.ID, &pbJobID, &jobType, &w.Name, &w.LastBeat, &w.GroupID, &w.ModelID, &statusS, &w.HatcheryName, &w.GroupID); err != nil {
		return nil, err
	}
	w.Status = sdk.StatusFromString(statusS)

	if jobType.Valid {
		w.JobType = jobType.String
	}

	if pbJobID.Valid {
		w.ActionBuildID = pbJobID.Int64
	}

	return w, nil
}

// LoadWorkers load all workers in db
func LoadWorkers(db gorp.SqlExecutor, hatcheryName string) ([]sdk.Worker, error) {
	w := []sdk.Worker{}
	var statusS string
	query := `SELECT id, name, last_beat, group_id, model, status, hatchery_name FROM worker ORDER BY name ASC`
	args := []interface{}{}

	if hatcheryName != "" {
		// TODO: remove the hatchery name from worker worker table !
		query = `SELECT id, name, last_beat, group_id, model, status, hatchery_name FROM worker WHERE hatchery_name = $1 ORDER BY name ASC`
		args = []interface{}{hatcheryName}
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var worker sdk.Worker
		err = rows.Scan(&worker.ID, &worker.Name, &worker.LastBeat, &worker.GroupID, &worker.ModelID, &statusS, &worker.HatcheryName)
		if err != nil {
			return nil, err
		}
		worker.Status = sdk.StatusFromString(statusS)
		w = append(w, worker)
	}

	return w, nil
}

// LoadDeadWorkers load worker with refresh last beat > timeout (seconds)
func LoadDeadWorkers(db gorp.SqlExecutor, timeout float64, status []string) ([]sdk.Worker, error) {
	var w []sdk.Worker
	var statusS string
	query := `SELECT id, action_build_id, job_type, name, last_beat, group_id, model, status, hatchery_name
				FROM worker
				WHERE status = ANY(string_to_array($1, ',')::text[])
				AND now() - last_beat > $2 * INTERVAL '1' SECOND
				ORDER BY name ASC
				LIMIT 10000`
	rows, err := db.Query(query, strings.Join(status, ","), int64(math.Floor(timeout)))
	if err != nil {
		log.Warning("LoadDeadWorkers> Error querying workers")
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var worker sdk.Worker
		var pbJobID sql.NullInt64
		var jobType sql.NullString
		err = rows.Scan(&worker.ID, &pbJobID, &jobType, &worker.Name, &worker.LastBeat, &worker.GroupID, &worker.ModelID, &statusS, &worker.HatcheryName)
		if err != nil {
			log.Warning("LoadDeadWorkers> Error scanning workers")
			return nil, err
		}
		if jobType.Valid {
			worker.JobType = jobType.String
		}
		if pbJobID.Valid {
			worker.ActionBuildID = pbJobID.Int64
		}
		worker.Status = sdk.StatusFromString(statusS)
		w = append(w, worker)
	}

	return w, nil
}

// RefreshWorker Update worker last_beat
func RefreshWorker(db gorp.SqlExecutor, w *sdk.Worker) error {
	if w == nil {
		return sdk.WrapError(sdk.ErrUnknownError, "RefreshWorker> Invalid worker")
	}
	query := `UPDATE worker SET last_beat = now() WHERE id = $1`
	res, err := db.Exec(query, w.ID)
	if err != nil {
		return sdk.WrapError(err, "Unable to update worker: %s", w.ID)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return sdk.WrapError(err, "Unable to refresh worker: %s", w.ID)
	}

	var mname string
	if w.Model != nil {
		mname = w.Model.Name
	}
	if n != 1 {
		return sdk.NewError(sdk.ErrForbidden, fmt.Errorf("unknown worker '%s' Name:%s GroupID:%d ModelID:%d Model:%s HatcheryName:%s",
			w.ID, w.Name, w.GroupID, w.ModelID, mname, w.HatcheryName))
	}

	return nil
}

func generateID() (string, error) {
	size := 64
	bs := make([]byte, size)
	if _, err := rand.Read(bs); err != nil {
		log.Error("api> worker> generateID: rand.Read failed: %s", err)
		return "", err
	}
	str := hex.EncodeToString(bs)
	token := []byte(str)[0:size]

	log.Debug("api> worker> generateID: new generated id: %s", token)
	return string(token), nil
}

// RegistrationForm represents the arguments needed to register a worker
type RegistrationForm struct {
	Name               string
	Token              string
	ModelID            int64
	Hatchery           int64
	HatcheryName       string
	BinaryCapabilities []string
	Version            string
	OS                 string
	Arch               string
}

// TakeForm contains booked JobID if exists
type TakeForm struct {
	BookedJobID int64
	Time        time.Time
}

// RegisterWorker  Register new worker
func RegisterWorker(db *gorp.DbMap, store cache.Store, name string, key string, modelID int64, hatchery *sdk.Service, binaryCapabilities []string, OS, arch string) (*sdk.Worker, error) {
	if name == "" {
		return nil, fmt.Errorf("cannot register worker with empty name")
	}
	if key == "" {
		return nil, fmt.Errorf("cannot register worker with empty worker key")
	}

	// Load token
	t, errL := token.LoadToken(db, key)
	if errL != nil {
		log.Warning("RegisterWorker> Cannot register worker. Caused by: %s", errL)
		return nil, errL
	}

	if hatchery != nil && hatchery.GroupID != nil {
		if *hatchery.GroupID != t.GroupID {
			return nil, sdk.ErrForbidden
		}
	}

	//Load Model
	var m *sdk.Model
	if modelID != 0 {
		var errM error
		m, errM = LoadWorkerModelByID(db, modelID)
		if errM != nil {
			log.Warning("RegisterWorker> Cannot load model: %s", errM)
			return nil, errM
		}
	}

	//If worker model is public (sharedInfraGroup) it can be ran by every one
	//If worker is public it can run every model
	//Private worker for a group cannot run a private model for another group
	if m != nil {
		if t.GroupID != group.SharedInfraGroup.ID && t.GroupID != m.GroupID && m.GroupID != group.SharedInfraGroup.ID {
			log.Warning("RegisterWorker> worker %s (%d) cannot be spawned as %s (%d)", name, t.GroupID, m.Name, m.GroupID)
			return nil, sdk.ErrForbidden
		}
	}

	//generate an ID
	id, errG := generateID()
	if errG != nil {
		log.Warning("registerWorker: Cannot generate ID: %s", errG)
		return nil, errG
	}

	//Instanciate a new worker
	w := &sdk.Worker{
		ID:      id,
		Name:    name,
		ModelID: modelID,
		Model:   m,
		Status:  sdk.StatusWaiting,
		GroupID: t.GroupID,
	}

	if hatchery != nil {
		w.HatcheryName = hatchery.Name
	}

	tx, errTx := db.Begin()
	if errTx != nil {
		return nil, errTx
	}
	defer tx.Rollback()

	if err := InsertWorker(tx, w, t.GroupID); err != nil {
		log.Warning("registerWorker: Cannot insert worker in database: %s", err)
		return nil, err
	}

	//If the worker is registered for a model and it gave us BinaryCapabilities...
	if len(binaryCapabilities) > 0 && modelID != 0 {
		go func() {
			//Start a new tx for this goroutine
			ntx, err := db.Begin()
			if err != nil {
				log.Warning("RegisterWorker> Unable to start a transaction: %s", err)
				return
			}
			defer ntx.Rollback()

			existingCapas, err := LoadWorkerModelCapabilities(ntx, modelID)
			if err != nil {
				log.Warning("RegisterWorker> Unable to load worker model capabilities: %s", err)
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
				log.Debug("Updating model %d binary capabilities with %d capabilities", modelID, len(newCapas))
				for _, b := range newCapas {
					query := `insert into worker_capability (worker_model_id, name, argument, type) values ($1, $2, $3, $4)`
					if _, err := ntx.Exec(query, modelID, b, b, string(sdk.BinaryRequirement)); err != nil {
						//Ignore errors because we let the database to check constraints...
						log.Debug("registerWorker> Cannot insert into worker_capability: %v", err)
						return
					}
				}
			}

			var capaToDelete []string
			for _, existingCapa := range existingCapas {
				var found bool
				for _, currentCapa := range binaryCapabilities {
					if existingCapa.Value == currentCapa {
						found = true
						break
					}
				}
				if !found {
					capaToDelete = append(capaToDelete, existingCapa.Value)
				}
			}

			if len(capaToDelete) > 0 {
				log.Debug("Updating model %d binary capabilities with %d capabilities to delete", modelID, len(capaToDelete))
				query := `DELETE FROM worker_capability WHERE worker_model_id=$1 AND name=ANY(string_to_array($2, ',')::text[]) AND type=$3`
				if _, err := db.Exec(query, modelID, strings.Join(capaToDelete, ","), string(sdk.BinaryRequirement)); err != nil {
					//Ignore errors because we let the database to check constraints...
					log.Warning("registerWorker> Cannot delete from worker_capability: %v", err)
					return
				}
			}

			if OS != "" && arch != "" {
				if err := updateOSAndArch(db, modelID, OS, arch); err != nil {
					log.Warning("registerWorker> Cannot update os and arch for worker model %d : %s", modelID, err)
					return
				}
			}

			if err := ntx.Commit(); err != nil {
				log.Warning("RegisterWorker> Unable to commit transaction: %s", err)
			}
		}()
		if err := updateRegistration(tx, modelID); err != nil {
			log.Warning("registerWorker> Unable updateRegistration: %s", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return w, err
	}

	// Useful to let models cache in hatchery refresh
	keyWorkerModel := keyBookWorkerModel(modelID)
	store.UpdateTTL(keyWorkerModel, modelsCacheTTLInSeconds+10)
	// delete from cache by group
	store.Delete(cache.Key("api:workermodels:bygroup", fmt.Sprintf("%d", t.GroupID)))

	return w, nil
}

// SetStatus sets action_build_id and status to building on given worker
func SetStatus(db gorp.SqlExecutor, workerID string, status sdk.Status) error {
	query := `UPDATE worker SET status = $1 WHERE id = $2`
	if status == sdk.StatusDisabled {
		query = `UPDATE worker SET status = $1, action_build_id = NULL WHERE id = $2`
	}

	if _, errE := db.Exec(query, status.String(), workerID); errE != nil {
		return sdk.WrapError(errE, "SetStatus")
	}
	return nil
}

// SetToBuilding sets action_build_id and status to building on given worker
func SetToBuilding(db gorp.SqlExecutor, workerID string, actionBuildID int64, jobType string) error {
	query := `UPDATE worker SET status = $1, action_build_id = $2, job_type = $3 WHERE id = $4`

	res, errE := db.Exec(query, sdk.StatusBuilding.String(), actionBuildID, jobType, workerID)
	if errE != nil {
		return errE
	}

	_, err := res.RowsAffected()
	return err
}

// UpdateWorkerStatus changes worker status to Disabled
func UpdateWorkerStatus(db gorp.SqlExecutor, workerID string, status sdk.Status) error {
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
		log.Error("UpdateWorkerStatus: Multiple (%d) rows affected ! (id=%s)", rowsAffected, workerID)
	}

	if rowsAffected == 0 {
		return ErrNoWorker
	}

	return nil
}
