package pipeline

import (
	"context"
	"fmt"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/observability"
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
func GetAllParametersInPipeline(ctx context.Context, db gorp.SqlExecutor, pipelineID int64) ([]sdk.Parameter, error) {
	_, end := observability.Span(ctx, "pipeline.GetAllParametersInPipeline")
	defer end()

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
	if param.Type == "" {
		return sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("Invalid parameter, wrong type"))
	}

	rx := sdk.NamePatternRegex
	if !rx.MatchString(param.Name) {
		return sdk.NewError(sdk.ErrInvalidName, fmt.Errorf("Invalid parameter name. It should match %s", sdk.NamePattern))
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
		return sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("Invalid parameter, wrong type"))
	}

	rx := sdk.NamePatternRegex
	if !rx.MatchString(param.Name) {
		return sdk.NewError(sdk.ErrInvalidName, fmt.Errorf("Invalid parameter name. It should match %s", sdk.NamePattern))
	}

	// update parameter
	query := `UPDATE pipeline_parameter SET value=$1, type=$2, description=$3, name=$4 WHERE pipeline_id=$5 AND name=$6`
	_, err := db.Exec(query, param.Value, string(param.Type), param.Description, param.Name, pipelineID, oldParamName)
	if err != nil {
		if errPG, ok := err.(*pq.Error); ok && errPG.Code == gorpmapping.ViolateUniqueKeyPGCode {
			return sdk.ErrParameterExists
		}
		return err
	}

	return nil
}

// DeleteParameterFromPipeline Delete a parameter from the given pipeline
func DeleteParameterFromPipeline(db gorp.SqlExecutor, pipelineID int64, paramName string) error {
	query := `DELETE FROM pipeline_parameter WHERE pipeline_id=$1 AND name=$2`
	_, err := db.Exec(query, pipelineID, paramName)
	if err != nil {
		return err
	}

	return nil
}

// DeleteAllParameterFromPipeline Delete all parameters from the given pipeline
func DeleteAllParameterFromPipeline(db gorp.SqlExecutor, pipelineID int64) error {
	query := `DELETE FROM pipeline_parameter WHERE pipeline_id=$1`
	_, err := db.Exec(query, pipelineID)
	return sdk.WrapError(err, "Unable to delete all parameters")
}

// CountInParamValue counts how many time a pattern is in parameter value for the given project
func CountInParamValue(db gorp.SqlExecutor, key string, value string) ([]string, error) {
	query := `
		SELECT DISTINCT pipeline.name
		FROM pipeline_parameter
		JOIN pipeline ON pipeline.id = pipeline_parameter.pipeline_id
		JOIN project ON project.id = pipeline.project_id
		WHERE value like $2 AND project.projectkey = $1;
	`
	rows, err := db.Query(query, key, fmt.Sprintf("%%%s%%", value))
	if err != nil {
		return nil, sdk.WrapError(err, "Unable to count usage")
	}
	defer rows.Close()

	var results []string
	for rows.Next() {
		var pipName string
		if err := rows.Scan(&pipName); err != nil {
			return nil, sdk.WrapError(err, "Unable to scan")
		}
		results = append(results, pipName)
	}
	return results, nil
}
