package pipeline

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

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
func GetAllParametersInPipeline(ctx context.Context, db gorp.SqlExecutor, pipelineID int64) ([]sdk.Parameter, error) {
	_, end := telemetry.Span(ctx, "pipeline.GetAllParametersInPipeline")
	defer end()

	parameters := []sdk.Parameter{}
	query := `SELECT id, name, value, type, description
	          FROM pipeline_parameter
	          WHERE pipeline_id=$1
	          ORDER BY name`
	rows, err := db.Query(query, pipelineID)
	if err != nil {
		return parameters, sdk.WithStack(err)
	}
	defer rows.Close()
	for rows.Next() {
		var p sdk.Parameter
		var typeParam, val string
		if err := rows.Scan(&p.ID, &p.Name, &val, &typeParam, &p.Description); err != nil {
			return nil, sdk.WithStack(err)
		}
		p.Type = typeParam
		p.Value = val
		parameters = append(parameters, p)
	}
	return parameters, nil
}

// InsertParameterInPipeline Insert a new parameter in the given pipeline
func InsertParameterInPipeline(db gorp.SqlExecutor, pipelineID int64, param *sdk.Parameter) error {
	if param.Type == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid parameter, wrong type")
	}

	rx := sdk.NamePatternRegex
	if !rx.MatchString(param.Name) {
		return sdk.NewErrorFrom(sdk.ErrInvalidName, "parameter name should match %s", sdk.NamePattern)
	}

	if string(param.Type) == string(sdk.SecretVariable) {
		return sdk.WrapError(sdk.ErrNoDirectSecretUse, "InsertParameterInPipeline>")
	}

	query := `INSERT INTO pipeline_parameter(pipeline_id, name, value, type, description)
		  VALUES($1, $2, $3, $4, $5) RETURNING id`
	err := db.QueryRow(query, pipelineID, param.Name, param.Value, string(param.Type), param.Description).Scan(&param.ID)
	if err != nil {
		return sdk.WrapError(err, "cannot insert in pipeline_parameter (pID:%d)", pipelineID)
	}

	return nil
}

// UpdateParameterInPipeline Update a parameter in the given pipeline
func UpdateParameterInPipeline(db gorp.SqlExecutor, pipelineID int64, oldParamName string, param sdk.Parameter) error {
	if param.Type == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid parameter, wrong type")
	}

	rx := sdk.NamePatternRegex
	if !rx.MatchString(param.Name) {
		return sdk.NewErrorFrom(sdk.ErrInvalidName, "parameter name should match pattern %s", sdk.NamePattern)
	}

	// update parameter
	query := `UPDATE pipeline_parameter SET value=$1, type=$2, description=$3, name=$4 WHERE pipeline_id=$5 AND name=$6`
	if _, err := db.Exec(query, param.Value, string(param.Type), param.Description, param.Name, pipelineID, oldParamName); err != nil {
		if errPG, ok := err.(*pq.Error); ok && errPG.Code == gorpmapper.ViolateUniqueKeyPGCode {
			return sdk.NewErrorWithStack(err, sdk.ErrParameterExists)
		}
		return sdk.WithStack(err)
	}

	return nil
}

// DeleteParameterFromPipeline Delete a parameter from the given pipeline
func DeleteParameterFromPipeline(db gorp.SqlExecutor, pipelineID int64, paramName string) error {
	query := `DELETE FROM pipeline_parameter WHERE pipeline_id=$1 AND name=$2`
	_, err := db.Exec(query, pipelineID, paramName)
	if err != nil {
		return sdk.WithStack(err)
	}

	return nil
}

// DeleteAllParameterFromPipeline Delete all parameters from the given pipeline
func DeleteAllParameterFromPipeline(db gorp.SqlExecutor, pipelineID int64) error {
	query := `DELETE FROM pipeline_parameter WHERE pipeline_id=$1`
	_, err := db.Exec(query, pipelineID)
	return sdk.WrapError(err, "unable to delete all parameters")
}
