package worker

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/token"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// PipelineBuildJobInfo is returned to worker in answer to takePipelineBuildJobHandler
type PipelineBuildJobInfo struct {
	PipelineBuildJob sdk.PipelineBuildJob
	Secrets          []sdk.Variable
	PipelineID       int64
	BuildNumber      int64
}

// WorkflowNodeJobRunInfo is returned to worker in answer to postTakeWorkflowJobHandler
type WorkflowNodeJobRunInfo struct {
	NodeJobRun sdk.WorkflowNodeJobRun
	Secrets    []sdk.Variable
	Number     int64
	SubNumber  int64
}

// ErrNoWorker means the given worker ID is not found
var ErrNoWorker = fmt.Errorf("cds: no worker found")

// DeleteWorker remove worker from database
func DeleteWorker(db *gorp.DbMap, id string) error {
	tx, errb := db.Begin()
	if errb != nil {
		return fmt.Errorf("DeleteWorker> Cannot start tx: %s", errb)
	}
	defer tx.Rollback()

	query := `SELECT name, status, action_build_id FROM worker WHERE id = $1 FOR UPDATE`
	var st, name string
	var pbJobID sql.NullInt64
	if err := tx.QueryRow(query, id).Scan(&name, &st, &pbJobID); err != nil {
		log.Debug("DeleteWorker[%d]> Cannot lock worker: %s", id, err)
		return nil
	}

	if st == sdk.StatusBuilding.String() {
		// Worker is awol while building !
		// We need to restart this action
		if pbJobID.Valid == false {
			return fmt.Errorf("DeleteWorker> Meh, worker %s crashed while building but action_build_id is NULL!", name)
		}

		log.Info("Worker %s crashed while building %d !", name, pbJobID.Int64)
		if err := pipeline.RestartPipelineBuildJob(tx, pbJobID.Int64); err != nil {
			log.Error("DeleteWorker[%d]> Cannot restart pipeline build job: %s", id, err)
		} else {
			log.Info("DeleteWorker[%d]> PipelineBuildJob %d restarted after crash", id, pbJobID.Int64)
		}
	}

	// Well then, let's remove this loser
	query = `DELETE FROM worker WHERE id = $1`
	if _, err := tx.Exec(query, id); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// InsertWorker inserts worker representation into database
func InsertWorker(db gorp.SqlExecutor, w *sdk.Worker, groupID int64) error {
	query := `INSERT INTO worker (id, name, last_beat, model, status, hatchery_id, hatchery_name, group_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := db.Exec(query, w.ID, w.Name, time.Now(), w.Model, w.Status.String(), w.HatcheryID, w.HatcheryName, groupID)
	return err
}

// LoadWorker retrieves worker in database
func LoadWorker(db gorp.SqlExecutor, id string) (*sdk.Worker, error) {
	w := &sdk.Worker{}
	var statusS string
	query := `SELECT id, name, last_beat, group_id, model, status, hatchery_id, hatchery_name, group_id FROM worker WHERE worker.id = $1 FOR UPDATE`

	err := db.QueryRow(query, id).Scan(&w.ID, &w.Name, &w.LastBeat, &w.GroupID, &w.Model, &statusS, &w.HatcheryID, &w.HatcheryName, &w.GroupID)
	if err != nil {
		return nil, err
	}
	w.Status = sdk.StatusFromString(statusS)

	return w, nil
}

// LoadWorkersByPipelineJobID load all workers in db by pipeline job id
func LoadWorkersByPipelineJobID(db gorp.SqlExecutor, pipJobID int64) ([]sdk.Worker, error) {
	w := []sdk.Worker{}
	var statusS string
	query := `SELECT id, name, last_beat, group_id, model, status, hatchery_id, hatchery_name FROM worker WHERE action_build_id = $1 ORDER BY name ASC`

	rows, err := db.Query(query, pipJobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var worker sdk.Worker
		err = rows.Scan(&worker.ID, &worker.Name, &worker.LastBeat, &worker.GroupID, &worker.Model, &statusS, &worker.HatcheryID, &worker.HatcheryName)
		if err != nil {
			return nil, err
		}
		worker.Status = sdk.StatusFromString(statusS)
		w = append(w, worker)
	}

	return w, nil
}

// LoadWorkers load all workers in db
func LoadWorkers(db gorp.SqlExecutor) ([]sdk.Worker, error) {
	w := []sdk.Worker{}
	var statusS string
	query := `SELECT id, name, last_beat, group_id, model, status, hatchery_id, hatchery_name FROM worker WHERE 1 = 1 ORDER BY name ASC`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var worker sdk.Worker
		err = rows.Scan(&worker.ID, &worker.Name, &worker.LastBeat, &worker.GroupID, &worker.Model, &statusS, &worker.HatcheryID, &worker.HatcheryName)
		if err != nil {
			return nil, err
		}
		worker.Status = sdk.StatusFromString(statusS)
		w = append(w, worker)
	}

	return w, nil
}

// LoadDeadWorkers load worker with refresh last beat > timeout
func LoadDeadWorkers(db gorp.SqlExecutor, timeout float64) ([]sdk.Worker, error) {
	var w []sdk.Worker
	var statusS string
	query := `	SELECT id, name, last_beat, group_id, model, status, hatchery_id, hatchery_name
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
		err = rows.Scan(&worker.ID, &worker.Name, &worker.LastBeat, &worker.GroupID, &worker.Model, &statusS, &worker.HatcheryID, &worker.HatcheryName)
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
func RefreshWorker(db gorp.SqlExecutor, workerID string) error {
	query := `UPDATE worker SET last_beat = $1 WHERE id = $2`
	res, err := db.Exec(query, time.Now(), workerID)
	if err != nil {
		log.Warning("RefreshWorker> Unable to update worker: %s", workerID)
		return err
	}

	n, err := res.RowsAffected()
	if err != nil {
		log.Warning("RefreshWorker> Unable to refresh worker: %s", workerID)
		return err
	}

	if n != 1 {
		return sdk.NewError(sdk.ErrForbidden, fmt.Errorf("unknown worker '%s'", workerID))
	}

	return nil
}

func generateID() (string, error) {
	size := 64
	bs := make([]byte, size)
	if _, err := rand.Read(bs); err != nil {
		log.Error("generateID: rand.Read failed: %s", err)
		return "", err
	}
	str := hex.EncodeToString(bs)
	token := []byte(str)[0:size]

	log.Debug("generateID: new generated id: %s", token)
	return string(token), nil
}

// RegistrationForm represents the arguments needed to register a worker
type RegistrationForm struct {
	Name               string
	Token              string
	Model              int64
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
func RegisterWorker(db *gorp.DbMap, name string, key string, modelID int64, h *sdk.Hatchery, binaryCapabilities []string) (*sdk.Worker, error) {
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

	if h != nil {
		if h.GroupID != t.GroupID {
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
		Model:   modelID,
		Status:  sdk.StatusWaiting,
		GroupID: t.GroupID,
	}

	if h != nil {
		w.HatcheryID = h.ID
		w.HatcheryName = h.Name
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
						log.Debug("registerWorker> Cannot insert into worker_capability: %s", err)
						return
					}
				}
			}
			if err := ntx.Commit(); err != nil {
				log.Warning("RegisterWorker> Unable to commit transaction: %s", err)
			}
		}()
		if err := updateRegistration(db, modelID); err != nil {
			log.Warning("registerWorker> Unable updateRegistration: %s", err)
		}
	}
	return w, tx.Commit()
}

// SetStatus sets action_build_id and status to building on given worker
func SetStatus(db gorp.SqlExecutor, workerID string, status sdk.Status) error {
	query := `UPDATE worker SET status = $1 WHERE id = $2`

	res, errE := db.Exec(query, status.String(), workerID)
	if errE != nil {
		return errE
	}

	_, err := res.RowsAffected()
	return err
}

// SetToBuilding sets action_build_id and status to building on given worker
func SetToBuilding(db gorp.SqlExecutor, workerID string, actionBuildID int64) error {
	query := `UPDATE worker SET status = $1, action_build_id = $2 WHERE id = $3`

	res, errE := db.Exec(query, sdk.StatusBuilding.String(), actionBuildID, workerID)
	if errE != nil {
		return errE
	}

	_, err := res.RowsAffected()
	return err
}

// LoadWorkerModelsUsableOnGroup returns worker models for a group
func LoadWorkerModelsUsableOnGroup(db gorp.SqlExecutor, groupID, sharedinfraGroupID int64) ([]sdk.Model, error) {
	ms := []WorkerModel{}
	if _, err := db.Select(&ms, `
		SELECT * from worker_model
		WHERE disabled = FALSE
		AND (group_id = $1 OR group_id = $2 OR $1 = $2)
		ORDER by name
		`, groupID, sharedinfraGroupID); err != nil {
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
