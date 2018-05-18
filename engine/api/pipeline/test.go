package pipeline

import (
	"database/sql"
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/venom"
)

// LoadTestResults retrieves tests on a specific build in database
func LoadTestResults(db gorp.SqlExecutor, pbID int64) (venom.Tests, error) {
	query := `SELECT tests FROM pipeline_build_test WHERE pipeline_build_id = $1`
	t := venom.Tests{}
	var data string

	err := db.QueryRow(query, pbID).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return t, nil
		}
		return t, err
	}

	err = json.Unmarshal([]byte(data), &t)
	if err != nil {
		return t, err
	}

	return t, nil
}

// InsertTestResults inserts test results of a specific pipeline build in database
func InsertTestResults(db gorp.SqlExecutor, pbID int64, tests venom.Tests) error {
	query := `INSERT INTO pipeline_build_test (pipeline_build_id, tests) VALUES ($1, $2)`

	data, err := json.Marshal(tests)
	if err != nil {
		return err
	}

	_, err = db.Exec(query, pbID, string(data))
	if err != nil {
		return err
	}

	return nil
}

// UpdateTestResults update test results of a specific pipeline build in database
func UpdateTestResults(db *gorp.DbMap, pbID int64, tests venom.Tests) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `DELETE FROM pipeline_build_test WHERE pipeline_build_id = $1`
	_, err = tx.Exec(query, pbID)
	if err != nil {
		return err
	}

	err = InsertTestResults(tx, pbID, tests)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

// DeletePipelineTestResults removes from database test results for a specific pipeline
func DeletePipelineTestResults(db gorp.SqlExecutor, pipID int64) error {
	query := `DELETE FROM pipeline_build_test WHERE pipeline_build_id IN
		(SELECT id FROM pipeline_build WHERE pipeline_id = $1)`

	_, err := db.Exec(query, pipID)
	if err != nil {
		return err
	}

	return nil
}

/*
// DeleteApplicationPipelineTestResults removes from database test results for a specific pipeline linked to a specific application
func DeleteApplicationPipelineTestResults(db gorp.SqlExecutor, appID int64, pipID int64) error {
	query := `DELETE FROM pipeline_build_test WHERE pipeline_build_id IN
		(SELECT id FROM pipeline_build WHERE application_id = $1 AND pipeline_id = $2)`

	_, err := db.Exec(query, appID, pipID)
	if err != nil {
		return err
	}

	return nil
}
*/
