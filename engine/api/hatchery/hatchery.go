package hatchery

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"math"
	"strings"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// InsertHatchery registers in database new hatchery
func InsertHatchery(db *sql.DB, h *sdk.Hatchery) error {

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	h.UID, err = generateID()
	if err != nil {
		return err
	}

	query := `INSERT INTO hatchery (name, group_id, last_beat, uid) VALUES ($1, $2, NOW(), $3) RETURNING id`
	err = tx.QueryRow(query, h.Name, h.GroupID, h.UID).Scan(&h.ID)
	if err != nil {
		return err
	}

	// allow hatchery to not declare any model
	if h.Model.Name == "" && h.Model.Image == "" {
		return tx.Commit()
	}

	query = `INSERT INTO worker_model (type, name, image, group_id) VALUES ($1,$2, $3,$4) RETURNING id`
	err = tx.QueryRow(query, string(sdk.HostProcess), h.Model.Name, h.Model.Image, h.GroupID).Scan(&h.Model.ID)
	if err != nil && strings.Contains(err.Error(), "idx_worker_model_name") {
		return sdk.ErrModelNameExist
	}
	if err != nil {
		return err
	}

	for _, c := range h.Model.Capabilities {
		query = `INSERT INTO worker_capability (worker_model_id, type, name, argument) VALUES ($1, $2, $3, $4)`
		_, err = tx.Exec(query, h.Model.ID, string(c.Type), c.Name, c.Value)
		if err != nil {
			log.Warning("Cannot insert capability: %s\n", err)
			return err
		}
	}

	query = `INSERT INTO hatchery_model (hatchery_id, worker_model_id) VALUES ($1, $2)`
	_, err = tx.Exec(query, h.ID, h.Model.ID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteHatchery removes from database given hatchery and linked model
func DeleteHatchery(db *sql.DB, id int64, workerModelID int64) error {
	dbmap := database.DBMap(db)
	tx, err := dbmap.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `DELETE FROM hatchery_model WHERE hatchery_id = $1`
	_, err = tx.Exec(query, id)
	if err != nil {
		return err
	}

	if workerModelID > 0 {
		err = worker.DeleteWorkerModel(tx, workerModelID)
		if err != nil {
			return err
		}
	}

	query = `DELETE FROM hatchery WHERE id = $1`
	_, err = tx.Exec(query, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// Exists returns an error is hatchery with given id does not exists
func Exists(db *sql.DB, id int64) error {
	query := `SELECT id FROM hatchery WHERE id = $1`
	return db.QueryRow(query, id).Scan(&id)
}

// LoadDeadHatcheries load hatchery with refresh last beat > timeout
func LoadDeadHatcheries(db *sql.DB, timeout float64) ([]sdk.Hatchery, error) {
	var hatcheries []sdk.Hatchery
	query := `	SELECT id, name, last_beat, group_id, worker_model_id
				FROM hatchery
				LEFT JOIN hatchery_model ON hatchery_model.hatchery_id = hatchery.id
				WHERE now() - last_beat > $1 * INTERVAL '1' SECOND
				LIMIT 10000`
	rows, err := db.Query(query, int64(math.Floor(timeout)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wmID sql.NullInt64
	for rows.Next() {
		var h sdk.Hatchery
		err = rows.Scan(&h.ID, &h.Name, &h.LastBeat, &h.GroupID, &wmID)
		if err != nil {
			return nil, err
		}
		if wmID.Valid {
			h.Model.ID = wmID.Int64
		}
		hatcheries = append(hatcheries, h)
	}

	return hatcheries, nil
}

// LoadHatchery fetch hatchery info from database given UID
func LoadHatchery(db *sql.DB, uid string) (*sdk.Hatchery, error) {
	query := `SELECT id, uid, name, last_beat, group_id, worker_model_id
							FROM hatchery
							LEFT JOIN hatchery_model ON hatchery_model.hatchery_id = hatchery.id
							WHERE uid = $1`

	var h sdk.Hatchery
	var wmID sql.NullInt64
	err := db.QueryRow(query, uid).Scan(&h.ID, &h.UID, &h.Name, &h.LastBeat, &h.GroupID, &wmID)
	if err != nil {
		return nil, err
	}

	if wmID.Valid {
		h.Model.ID = wmID.Int64
	}

	return &h, nil
}

// LoadHatcheries retrieves in database all registered hatcheries
func LoadHatcheries(db *sql.DB) ([]sdk.Hatchery, error) {
	var hatcheries []sdk.Hatchery

	query := `SELECT id, uid, name, last_beat, group_id, worker_model_id
							FROM hatchery
							LEFT JOIN hatchery_model ON hatchery_model.hatchery_id = hatchery.id
							LIMIT 10000`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wmID sql.NullInt64
	for rows.Next() {
		var h sdk.Hatchery
		err = rows.Scan(&h.ID, &h.UID, &h.Name, &h.LastBeat, &h.GroupID, &wmID)
		if err != nil {
			return nil, err
		}
		if wmID.Valid {
			h.Model.ID = wmID.Int64
		}
		hatcheries = append(hatcheries, h)
	}

	return hatcheries, nil
}

// RefreshHatchery Update hatchery last_beat
func RefreshHatchery(db *sql.DB, hatchID string) error {
	query := `UPDATE hatchery SET last_beat = NOW() WHERE id = $1`
	res, err := db.Exec(query, hatchID)
	if err != nil {
		return err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if n == 0 {
		return sdk.ErrNotFound
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
