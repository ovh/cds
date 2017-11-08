package hatchery

import (
	"crypto/rand"
	"crypto/sha512"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"math"
	"strings"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// InsertHatchery registers in database new hatchery
func InsertHatchery(dbmap *gorp.DbMap, h *sdk.Hatchery) error {
	tx, errb := dbmap.Begin()
	if errb != nil {
		return errb
	}
	defer tx.Rollback()

	var errg error
	h.UID, errg = generateID()
	if errg != nil {
		return errg
	}

	query := `INSERT INTO hatchery (name, group_id, last_beat, uid) VALUES ($1, $2, NOW(), $3) RETURNING id`
	if err := tx.QueryRow(query, h.Name, h.GroupID, h.UID).Scan(&h.ID); err != nil {
		return err
	}

	// allow hatchery to not declare any model
	if h.Model.Name == "" && h.Model.Image == "" {
		return tx.Commit()
	}

	//only local hatcheries declare model on registration
	h.Model.CreatedBy = sdk.User{Fullname: "Hatchery", Username: h.Name}
	h.Model.Type = string(sdk.HostProcess)
	h.Model.GroupID = h.GroupID
	h.Model.UserLastModified = time.Now()

	if err := worker.InsertWorkerModel(tx, &h.Model); err != nil && strings.Contains(err.Error(), "idx_worker_model_name") {
		return sdk.ErrModelNameExist
	} else if err != nil {
		return err
	}

	query = `INSERT INTO hatchery_model (hatchery_id, worker_model_id) VALUES ($1, $2)`
	if _, err := tx.Exec(query, h.ID, h.Model.ID); err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteHatchery removes from database given hatchery and linked model
func DeleteHatchery(dbmap *gorp.DbMap, id int64, workerModelID int64) error {
	tx, errb := dbmap.Begin()
	if errb != nil {
		return errb
	}
	defer tx.Rollback()

	query := `DELETE FROM hatchery_model WHERE hatchery_id = $1`
	if _, err := tx.Exec(query, id); err != nil {
		return err
	}

	if workerModelID > 0 {
		if err := worker.DeleteWorkerModel(tx, workerModelID); err != nil {
			return err
		}
	}

	query = `DELETE FROM hatchery WHERE id = $1`
	if _, err := tx.Exec(query, id); err != nil {
		return err
	}

	return tx.Commit()
}

// LoadDeadHatcheries load hatchery with refresh last beat > timeout
func LoadDeadHatcheries(db gorp.SqlExecutor, timeout float64) ([]sdk.Hatchery, error) {
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
func LoadHatchery(db gorp.SqlExecutor, uid string) (*sdk.Hatchery, error) {
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

// LoadHatcheryByName fetch hatchery info from database given name
func LoadHatcheryByName(db gorp.SqlExecutor, name string) (*sdk.Hatchery, error) {
	query := `SELECT id, uid, name, last_beat, group_id, worker_model_id
			FROM hatchery
			LEFT JOIN hatchery_model ON hatchery_model.hatchery_id = hatchery.id
			WHERE hatchery.name = $1`

	var h sdk.Hatchery
	var wmID sql.NullInt64
	err := db.QueryRow(query, name).Scan(&h.ID, &h.UID, &h.Name, &h.LastBeat, &h.GroupID, &wmID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNoHatchery
		}
		return nil, err
	}

	if wmID.Valid {
		h.Model.ID = wmID.Int64
	}

	return &h, nil
}

// LoadHatcheryByNameAndToken fetch hatchery info from database given name and hashed token
func LoadHatcheryByNameAndToken(db gorp.SqlExecutor, name, token string) (*sdk.Hatchery, error) {
	query := `SELECT hatchery.id, hatchery.uid, hatchery.name, hatchery.last_beat, hatchery.group_id, hatchery_model.worker_model_id
			FROM hatchery
			LEFT JOIN hatchery_model ON hatchery_model.hatchery_id = hatchery.id
			LEFT JOIN token ON hatchery.group_id = token.group_id
			WHERE hatchery.name = $1 AND token.token = $2`

	var h sdk.Hatchery
	var wmID sql.NullInt64
	hasher := sha512.New()
	hashed := base64.StdEncoding.EncodeToString(hasher.Sum([]byte(token)))
	err := db.QueryRow(query, name, hashed).Scan(&h.ID, &h.UID, &h.Name, &h.LastBeat, &h.GroupID, &wmID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNoHatchery
		}
		return nil, err
	}

	if wmID.Valid {
		h.Model.ID = wmID.Int64
	}

	return &h, nil
}

// LoadHatcheries retrieves in database all registered hatcheries
func LoadHatcheries(db gorp.SqlExecutor) ([]sdk.Hatchery, error) {
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

// Update update hatchery
func Update(db gorp.SqlExecutor, hatch sdk.Hatchery) error {
	query := `UPDATE hatchery SET name = $1, group_id = $2, last_beat = NOW(), uid = $3  WHERE id = $4`
	res, err := db.Exec(query, hatch.Name, hatch.GroupID, hatch.UID, hatch.ID)
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

// RefreshHatchery Update hatchery last_beat
func RefreshHatchery(db gorp.SqlExecutor, hatchID string) error {
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
	if _, err := rand.Read(bs); err != nil {
		return "", sdk.WrapError(err, "generateID> rand.Read failed")
	}
	str := hex.EncodeToString(bs)
	token := []byte(str)[0:size]

	log.Debug("generateID> new generated id: %s", token)
	return string(token), nil
}
