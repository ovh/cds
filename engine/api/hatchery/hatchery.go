package hatchery

import (
	"crypto/rand"
	"crypto/sha512"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// InsertHatchery registers in database new hatchery
func InsertHatchery(db gorp.SqlExecutor, hatchery *sdk.Hatchery) error {
	var errg error
	hatchery.UID, errg = generateID()
	if errg != nil {
		return errg
	}

	// allow hatchery to not declare any model
	if hatchery.Type == "local" && hatchery.Model.ModelVirtualMachine.Image == "" && hatchery.Model.ModelDocker.Image == "" {
		//only local hatcheries declare model on registration
		hatchery.Model.CreatedBy = sdk.User{Fullname: "Hatchery", Username: hatchery.Name}
		hatchery.Model.Type = string(sdk.HostProcess)
		hatchery.Model.GroupID = hatchery.GroupID
		hatchery.Model.UserLastModified = time.Now()
		hatchery.Model.ModelVirtualMachine = sdk.ModelVirtualMachine{
			Image: hatchery.Model.Name,
			Cmd:   "worker --api={{.API}} --token={{.Token}} --basedir={{.BaseDir}} --model={{.Model}} --name={{.Name}} --hatchery={{.Hatchery}} --hatchery-name={{.HatcheryName}} --insecure={{.HTTPInsecure}} --booked-workflow-job-id={{.WorkflowJobID}} --booked-pb-job-id={{.PipelineBuildJobID}} --single-use --force-exit",
		}

		if err := worker.InsertWorkerModel(db, &hatchery.Model); err != nil && strings.Contains(err.Error(), "idx_worker_model_name") {
			return sdk.ErrModelNameExist
		} else if err != nil {
			return err
		}
		hatchery.WorkerModelID = hatchery.Model.ID
	}

	h := Hatchery(*hatchery)
	if err := db.Insert(&h); err != nil {
		return err
	}
	*hatchery = sdk.Hatchery(h)
	return nil
}

// DeleteHatcheryByName removes from database given hatchery
func DeleteHatcheryByName(db gorp.SqlExecutor, name string) error {
	hatchery, err := LoadHatcheryByName(db, name)
	if err != nil {
		return err
	}
	if hatchery.WorkerModelID > 0 {
		if err := worker.DeleteWorkerModel(db, hatchery.WorkerModelID); err != nil {
			return err
		}
	}
	query := `DELETE FROM hatchery WHERE id = $1`
	_, err = db.Exec(query, hatchery.ID)
	return err
}

// LoadHatchery fetch hatchery info from database given UID
func LoadHatchery(db gorp.SqlExecutor, uid, name string) (*sdk.Hatchery, error) {

	//TODO yesnault

	query := `SELECT id, uid, name, group_id, worker_model_id, type, model_type, ratio_service
	FROM hatchery
	WHERE uid = $1 AND name = $2`

	var h sdk.Hatchery
	var wmID sql.NullInt64
	err := db.QueryRow(query, uid, name).Scan(&h.ID, &h.UID, &h.Name, &h.GroupID, &wmID, &h.Type, &h.ModelType, &h.RatioService)
	if err != nil {
		return nil, err
	}

	if wmID.Valid {
		h.Model.ID = wmID.Int64
		h.WorkerModelID = wmID.Int64
	}

	return &h, nil
}

// LoadHatcheryByName fetch hatchery info from database given name
func LoadHatcheryByName(db gorp.SqlExecutor, name string) (*sdk.Hatchery, error) {

	//TODO yesnault

	query := `SELECT id, uid, name, group_id, worker_model_id, type, model_type, ratio_service
			FROM hatchery
			WHERE hatchery.name = $1`

	var h sdk.Hatchery
	var wmID sql.NullInt64
	err := db.QueryRow(query, name).Scan(&h.ID, &h.UID, &h.Name, &h.GroupID, &wmID, &h.Type, &h.ModelType, &h.RatioService)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNoHatchery
		}
		return nil, err
	}

	if wmID.Valid {
		h.Model.ID = wmID.Int64
		h.WorkerModelID = wmID.Int64
	}

	return &h, nil
}

// LoadHatcheryByNameAndToken fetch hatchery info from database given name and hashed token
func LoadHatcheryByNameAndToken(db gorp.SqlExecutor, name, token string) (*sdk.Hatchery, error) {
	query := `SELECT hatchery.id, hatchery.uid, hatchery.name, hatchery.group_id, hatchery.worker_model_id, hatchery.type, hatchery.model_type, hatchery.ratio_service
			FROM hatchery
			LEFT JOIN token ON hatchery.group_id = token.group_id
			WHERE hatchery.name = $1 AND token.token = $2`

	var h sdk.Hatchery
	var wmID sql.NullInt64
	hasher := sha512.New()
	hashed := base64.StdEncoding.EncodeToString(hasher.Sum([]byte(token)))
	err := db.QueryRow(query, name, hashed).Scan(&h.ID, &h.UID, &h.Name, &h.GroupID, &wmID, &h.Type, &h.ModelType, &h.RatioService)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNoHatchery
		}
		return nil, err
	}

	if wmID.Valid {
		h.Model.ID = wmID.Int64
		h.WorkerModelID = wmID.Int64
	}

	return &h, nil
}

// CountHatcheries retrieves in database the number of hatcheries
func CountHatcheries(db gorp.SqlExecutor, wfNodeRunID int64) (int64, error) {
	query := `
	SELECT COUNT(1)
		FROM hatchery
		WHERE (
			hatchery.group_id = ANY(
				SELECT DISTINCT(project_group.group_id)
					FROM workflow_node_run
						JOIN workflow_run ON workflow_run.id = workflow_node_run.workflow_run_id
						JOIN workflow ON workflow.id = workflow_run.workflow_id
						JOIN project ON project.id = workflow.project_id
						JOIN project_group ON project_group.project_id = project.id
				WHERE workflow_node_run.id = $1
				AND project_group.role >= 5
			)
			OR
			hatchery.group_id = $2
		)
	`
	return db.SelectInt(query, wfNodeRunID, group.SharedInfraGroup.ID)
}

// LoadHatcheriesCountByNodeJobRunID retrieves in database the number of hatcheries given the node job run id
func LoadHatcheriesCountByNodeJobRunID(db gorp.SqlExecutor, wfNodeJobRunID int64) (int64, error) {
	query := `
	SELECT COUNT(1)
		FROM hatchery
		WHERE (
			hatchery.group_id = ANY(
				SELECT DISTINCT(project_group.group_id)
					FROM workflow_node_run_job
						JOIN workflow_node_run ON workflow_node_run.id = workflow_node_run_job.workflow_node_run_id
						JOIN workflow_run ON workflow_run.id = workflow_node_run.workflow_run_id
						JOIN workflow ON workflow.id = workflow_run.workflow_id
						JOIN project ON project.id = workflow.project_id
						JOIN project_group ON project_group.project_id = project.id
				WHERE workflow_node_run.id = $1
				AND project_group.role >= 5
			)
			OR
			hatchery.group_id = $2
		)
	`
	return db.SelectInt(query, wfNodeJobRunID, group.SharedInfraGroup.ID)
}

// Update update hatchery
func Update(db gorp.SqlExecutor, hatch sdk.Hatchery) error {

	//TODO yesnault

	query := `UPDATE hatchery SET name = $1, group_id = $2, uid = $3, type = $4, model_type = $5, ratio_service = $6 WHERE id = $7`
	res, err := db.Exec(query, hatch.Name, hatch.GroupID, hatch.UID, hatch.Type, hatch.ModelType, hatch.RatioService, hatch.ID)
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

	log.Debug("hatchery> generateID> new generated id: %s", token)
	return string(token), nil
}
