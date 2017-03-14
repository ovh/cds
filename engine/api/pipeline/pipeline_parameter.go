package pipeline

import (
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/sdk"
)

// FuncArg defines the base type for functional argument of pipeline funcs
type FuncArg func(args *structarg)

// CheckParameterInPipeline check if the parameter is already in the pipeline or not
func CheckParameterInPipeline(db gorp.SqlExecutor, pipelineID int64, paramName string) (bool, error) {
	query := `SELECT COUNT(id) FROM pipeline_parameter WHERE pipeline_id = $1 AND name = $2`

	var nb int64
	err := db.QueryRow(query, pipelineID, paramName).Scan(&nb)
	if err != nil {
		return false, err
	}
	if nb != 0 {
		return true, nil
	}
	return false, nil
}

// GetAllParametersInPipeline Get all parameters for the given pipeline
func GetAllParametersInPipeline(db gorp.SqlExecutor, pipelineID int64 /*, args ...FuncArg*/) ([]sdk.Parameter, error) {
	/*
			c := structarg{}
		for _, f := range args {
			f(&c)
		}
	*/
	parameters := []sdk.Parameter{}
	query := `SELECT id, name, value, type, description
	          FROM pipeline_parameter
	          WHERE pipeline_id=$1
	          ORDER BY name`
	rows, err := db.Query(query, pipelineID)
	if err != nil {
		return parameters, err
	}
	defer rows.Close()
	for rows.Next() {
		var p sdk.Parameter
		var typeParam, val string
		err = rows.Scan(&p.ID, &p.Name, &val, &typeParam, &p.Description)
		if err != nil {
			return nil, err
		}
		p.Type = typeParam
		p.Value = val
		parameters = append(parameters, p)
	}
	return parameters, err
}

// InsertParameterInPipeline Insert a new parameter in the given pipeline
func InsertParameterInPipeline(db gorp.SqlExecutor, pipelineID int64, param *sdk.Parameter) error {

	if string(param.Type) == string(sdk.SecretVariable) {
		return sdk.ErrNoDirectSecretUse
	}

	/* DEPRECATED: no more password in parameter type
	clear, cipher, err := secret.EncryptS(param.Type, param.Value)
	if err != nil {
		return err
	}

	query := `INSERT INTO pipeline_parameter(pipeline_id, name, value, cipher_value, type, description)
		  VALUES($1, $2, $3, $4, $5, $6) RETURNING id`
	err = db.QueryRow(query, pipelineID, param.Name, clear, cipher, string(param.Type), param.Description).Scan(&param.ID)
	*/
	query := `INSERT INTO pipeline_parameter(pipeline_id, name, value, type, description)
		  VALUES($1, $2, $3, $4, $5) RETURNING id`
	err := db.QueryRow(query, pipelineID, param.Name, param.Value, string(param.Type), param.Description).Scan(&param.ID)
	if err != nil {
		return fmt.Errorf("cannot insert in pipeline_parameter (pID:%d): %s", pipelineID, err)
	}

	query = `SELECT id FROM pipeline_trigger WHERE dest_pipeline_id = $1`
	rows, err := db.Query(query, pipelineID)
	if err != nil {
		return err
	}
	defer rows.Close()
	var ids []int64
	var id int64
	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			return fmt.Errorf("cannot scan pipeline_trigger (pID:%d): %s", pipelineID, err)
		}
		ids = append(ids, id)
	}
	for _, id := range ids {
		err = trigger.InsertTriggerParameter(db, id, *param)
		if err != nil {
			return fmt.Errorf("cannot InsertTriggerParameter (tID:%d): %s", id, err)
		}
	}

	return nil
}

// UpdateParameterInPipeline Update a parameter in the given pipeline
func UpdateParameterInPipeline(db gorp.SqlExecutor, pipelineID int64, param sdk.Parameter) error {
	/* DEPRECATED: no more password in parameter
		clear, cipher, err := secret.EncryptS(param.Type, param.Value)
	if err != nil {
		return err
	}

	// update parameter
	query := `UPDATE pipeline_parameter SET value=$1, cipher_value=$2, type=$3, description=$4, name=$6 WHERE pipeline_id=$5 AND id=$7`
	_, err = db.Exec(query, clear, cipher, string(param.Type), param.Description, pipelineID, param.Name, param.ID)
	if err != nil {
		return err
	}
	*/
	// update parameter
	query := `UPDATE pipeline_parameter SET value=$1, type=$2, description=$3, name=$5 WHERE pipeline_id=$4 AND id=$6`
	_, err := db.Exec(query, param.Value, string(param.Type), param.Description, pipelineID, param.Name, param.ID)
	if err != nil {
		return err
	}

	// Update this parameter in triggers as well
	query = `UPDATE pipeline_trigger_parameter SET type=$1, description=$2 WHERE id IN (
		SELECT pipeline_trigger_parameter.id FROM pipeline_trigger_parameter
		JOIN pipeline_trigger ON pipeline_trigger.id = pipeline_trigger_parameter.pipeline_trigger_id
		WHERE pipeline_trigger.dest_pipeline_id = $3
		AND pipeline_trigger_parameter.name = $4
	)`
	_, err = db.Exec(query, string(param.Type), param.Description, pipelineID, param.Name)
	return err
}

// DeleteParameterFromPipeline Delete a parameter from the given pipeline
func DeleteParameterFromPipeline(db gorp.SqlExecutor, pipelineID int64, paramName string) error {
	query := `DELETE FROM pipeline_parameter WHERE pipeline_id=$1 AND name=$2`
	_, err := db.Exec(query, pipelineID, paramName)
	if err != nil {
		return err
	}

	// Delete this parameter in triggers as well
	query = `DELETE FROM pipeline_trigger_parameter WHERE id IN (
		SELECT pipeline_trigger_parameter.id FROM pipeline_trigger_parameter
		JOIN pipeline_trigger ON pipeline_trigger.id = pipeline_trigger_parameter.pipeline_trigger_id
		WHERE pipeline_trigger.dest_pipeline_id = $1
		AND pipeline_trigger_parameter.name = $2
	)`
	_, err = db.Exec(query, pipelineID, paramName)
	if err != nil {
		return err
	}

	return nil
}

// DeleteAllParameterFromPipeline Delete all parameters from the given pipeline
func DeleteAllParameterFromPipeline(db gorp.SqlExecutor, pipelineID int64) error {
	query := `DELETE FROM pipeline_parameter WHERE pipeline_id=$1`
	_, err := db.Exec(query, pipelineID)
	return err
}
