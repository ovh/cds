package pipeline

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

// LoadPipeline loads a pipeline from database
func LoadPipeline(ctx context.Context, db gorp.SqlExecutor, projectKey, name string, deep bool) (*sdk.Pipeline, error) {
	ctx, end := telemetry.Span(ctx, "pipeline.LoadPipeline",
		telemetry.Tag(telemetry.TagProjectKey, projectKey),
		telemetry.Tag(telemetry.TagPipeline, name),
		telemetry.Tag(telemetry.TagPipelineDeep, deep),
	)
	defer end()

	var p Pipeline
	query := `SELECT pipeline.id, pipeline.name, pipeline.description, pipeline.project_id, pipeline.last_modified, pipeline.from_repository
			FROM pipeline
	 			JOIN project on pipeline.project_id = project.id
	 		WHERE pipeline.name = $1 AND project.projectKey = $2`

	if err := db.SelectOne(&p, query, name, projectKey); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WithData(sdk.ErrPipelineNotFound, name)
		}
		return nil, sdk.WithStack(err)
	}

	pip := sdk.Pipeline(p)
	if deep {
		if err := loadPipelineDependencies(ctx, db, &pip); err != nil {
			return nil, err
		}
	} else {
		parameters, err := GetAllParametersInPipeline(ctx, db, pip.ID)
		if err != nil {
			return nil, err
		}
		pip.Parameter = parameters
	}
	return &pip, nil
}

// LoadPipelineByID loads a pipeline from database
func LoadPipelineByID(ctx context.Context, db gorp.SqlExecutor, pipelineID int64, deep bool) (*sdk.Pipeline, error) {
	var end func()
	ctx, end = telemetry.Span(ctx, "pipeline.LoadPipelineByID",
		telemetry.Tag(telemetry.TagPipelineID, pipelineID),
		telemetry.Tag(telemetry.TagPipelineDeep, deep),
	)
	defer end()

	var p Pipeline
	query := `SELECT pipeline.*
	FROM pipeline
	WHERE pipeline.id = $1`

	if err := db.SelectOne(&p, query, pipelineID); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrPipelineNotFound
		}
		return nil, sdk.WithStack(err)
	}
	pip := sdk.Pipeline(p)

	if deep {
		if err := loadPipelineDependencies(ctx, db, &pip); err != nil {
			return nil, err
		}
	} else {
		parameters, err := GetAllParametersInPipeline(ctx, db, pip.ID)
		if err != nil {
			return nil, err
		}
		pip.Parameter = parameters
	}

	return &pip, nil
}

// LoadByWorkerModel loads pipelines from database for a given worker model.
func LoadByWorkerModel(ctx context.Context, db gorp.SqlExecutor, model *sdk.Model) ([]sdk.Pipeline, error) {
	var query gorpmapping.Query

	isSharedInfraModel := model.GroupID == group.SharedInfraGroup.ID
	modelNamePatternWithGroup := model.Group.Name + "/" + model.Name

	modelNamePattern1 := fmt.Sprintf("^%s(?!\\S)", model.Name)
	modelNamePattern2 := fmt.Sprintf("^%s(?!\\S)", modelNamePatternWithGroup)

	if isSharedInfraModel {
		query = gorpmapping.NewQuery(`
      SELECT DISTINCT pipeline.*, project.projectkey AS projectKey
      FROM action_requirement
        JOIN pipeline_action ON action_requirement.action_id = pipeline_action.action_id
        JOIN pipeline_stage ON pipeline_action.pipeline_stage_id = pipeline_stage.id
        JOIN pipeline ON pipeline.id = pipeline_stage.pipeline_id
        JOIN project ON project.id = pipeline.project_id
      WHERE action_requirement.type = 'model'
        AND (action_requirement.value ~ $1 OR action_requirement.value ~ $2)
    `).Args(modelNamePattern1, modelNamePattern2)
	} else {
		query = gorpmapping.NewQuery(`
      SELECT DISTINCT pipeline.*, project.projectkey AS projectKey
      FROM action_requirement
        JOIN pipeline_action ON action_requirement.action_id = pipeline_action.action_id
        JOIN pipeline_stage ON pipeline_action.pipeline_stage_id = pipeline_stage.id
        JOIN pipeline ON pipeline.id = pipeline_stage.pipeline_id
        JOIN project ON project.id = pipeline.project_id
      WHERE action_requirement.type = 'model'
        AND action_requirement.value ~ $1
    `).Args(modelNamePattern2)
	}

	var dbPips Pipelines
	if err := gorpmapping.GetAll(ctx, db, query, &dbPips); err != nil {
		return nil, sdk.WrapError(err, "unable to load pipelines linked to worker model pattern %s", modelNamePattern2)
	}

	return dbPips.Cast(), nil
}

// LoadByWorkerModelAndGroupIDs loads pipelines from database for a given worker model and group ids.
func LoadByWorkerModelAndGroupIDs(ctx context.Context, db gorp.SqlExecutor, model *sdk.Model, groupIDs []int64) ([]sdk.Pipeline, error) {
	var query gorpmapping.Query

	isSharedInfraModel := model.GroupID == group.SharedInfraGroup.ID
	modelNamePatternWithGroup := model.Group.Name + "/" + model.Name

	modelNamePattern1 := fmt.Sprintf("^%s(?!\\S)", model.Name)
	modelNamePattern2 := fmt.Sprintf("^%s(?!\\S)", modelNamePatternWithGroup)

	if isSharedInfraModel {
		query = gorpmapping.NewQuery(`
      SELECT DISTINCT pipeline.*, project.projectkey AS projectKey
      FROM action_requirement
        JOIN pipeline_action ON action_requirement.action_id = pipeline_action.action_id
        JOIN pipeline_stage ON pipeline_action.pipeline_stage_id = pipeline_stage.id
        JOIN pipeline ON pipeline.id = pipeline_stage.pipeline_id
        JOIN project ON project.id = pipeline.project_id
      WHERE action_requirement.type = 'model'
        AND (action_requirement.value ~ $1 OR action_requirement.value ~ $2)
        AND project.id IN (
          SELECT project_group.project_id
            FROM project_group
          WHERE project_group.group_id = ANY(string_to_array($3, ',')::int[])
        )
    `).Args(modelNamePattern1, modelNamePattern2, gorpmapping.IDsToQueryString(groupIDs))
	} else {
		query = gorpmapping.NewQuery(`
      SELECT DISTINCT pipeline.*, project.projectkey AS projectKey
      FROM action_requirement
        JOIN pipeline_action ON action_requirement.action_id = pipeline_action.action_id
        JOIN pipeline_stage ON pipeline_action.pipeline_stage_id = pipeline_stage.id
        JOIN pipeline ON pipeline.id = pipeline_stage.pipeline_id
        JOIN project ON project.id = pipeline.project_id
      WHERE action_requirement.type = 'model'
        AND action_requirement.value ~ $1
        AND project.id IN (
          SELECT project_group.project_id
            FROM project_group
          WHERE
            project_group.group_id = ANY(string_to_array($2, ',')::int[])
        )
    `).Args(modelNamePattern2, gorpmapping.IDsToQueryString(groupIDs))
	}

	var pips Pipelines
	if err := gorpmapping.GetAll(ctx, db, query, &pips); err != nil {
		return nil, sdk.WrapError(err, "unable to load pipelines linked to worker model pattern %s", modelNamePattern2)
	}

	return pips.Cast(), nil
}

// LoadByWorkflowID loads pipelines from database for a given workflow id
func LoadByWorkflowID(db gorp.SqlExecutor, workflowID int64) ([]sdk.Pipeline, error) {
	pips := []Pipeline{}
	query := `SELECT DISTINCT pipeline.*
	FROM pipeline
		JOIN w_node_context ON pipeline.id = w_node_context.pipeline_id
    JOIN w_node ON w_node.id = w_node_context.node_id
		JOIN workflow ON w_node.workflow_id = workflow.id
	WHERE workflow.id = $1`

	if _, err := db.Select(&pips, query, workflowID); err != nil {
		if err == sql.ErrNoRows {
			return []sdk.Pipeline{}, nil
		}
		return nil, sdk.WrapError(err, "Unable to load pipelines linked to workflow id %d", workflowID)
	}
	pipsSdk := make([]sdk.Pipeline, len(pips))
	for i := range pips {
		pipsSdk[i] = sdk.Pipeline(pips[i])
	}

	return pipsSdk, nil
}

func loadPipelineDependencies(ctx context.Context, db gorp.SqlExecutor, p *sdk.Pipeline) error {
	if err := LoadPipelineStage(ctx, db, p); err != nil {
		return err
	}

	parameters, err := GetAllParametersInPipeline(ctx, db, p.ID)
	if err != nil {
		return err
	}
	p.Parameter = parameters
	return nil
}

// DeletePipeline remove given pipeline and all history from database
func DeletePipeline(ctx context.Context, db gorp.SqlExecutor, pipelineID int64) error {
	if err := DeleteAllStage(ctx, db, pipelineID); err != nil {
		return err
	}

	if err := DeleteAllParameterFromPipeline(db, pipelineID); err != nil {
		return err
	}

	// Delete pipeline
	query := `DELETE FROM pipeline WHERE id = $1`
	if _, err := db.Exec(query, pipelineID); err != nil {
		return sdk.WithStack(err)
	}

	return nil
}

// LoadAllByIDs loads all pipelines
func LoadAllByIDs(db gorp.SqlExecutor, ids []int64, loadDependencies bool) ([]sdk.Pipeline, error) {
	var pips []sdk.Pipeline
	query := `SELECT id, name, description, project_id, last_modified, from_repository
			  FROM pipeline
			  WHERE id = ANY($1)
			  ORDER BY pipeline.name`

	if _, err := db.Select(&pips, query, pq.Int64Array(ids)); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WithStack(err)
	}

	for i := range pips {
		if loadDependencies {
			if err := LoadPipelineStage(context.TODO(), db, &pips[i]); err != nil {
				return nil, err
			}
		}
		params, err := GetAllParametersInPipeline(context.TODO(), db, pips[i].ID)
		if err != nil {
			return nil, err
		}
		pips[i].Parameter = params
	}

	return pips, nil
}

// LoadPipelines loads all pipelines in a project
func LoadPipelines(db gorp.SqlExecutor, projectID int64, loadDependencies bool) ([]sdk.Pipeline, error) {
	var pips []sdk.Pipeline
	query := `SELECT id, name, description, project_id, last_modified, from_repository
			  FROM pipeline
			  WHERE project_id = $1
			  ORDER BY pipeline.name`

	if _, err := db.Select(&pips, query, projectID); err != nil {
		return nil, sdk.WithStack(err)
	}

	for i := range pips {
		if loadDependencies {
			// load pipeline stages
			if err := LoadPipelineStage(context.TODO(), db, &pips[i]); err != nil {
				return nil, err
			}
		}
		params, err := GetAllParametersInPipeline(context.TODO(), db, pips[i].ID)
		if err != nil {
			return nil, err
		}
		pips[i].Parameter = params
	}

	return pips, nil
}

// LoadAllNames returns all pipeline names
func LoadAllNames(db gorp.SqlExecutor, projID int64) (sdk.IDNames, error) {
	query := `SELECT pipeline.id, pipeline.name, pipeline.description
			  FROM pipeline
			  WHERE project_id = $1
			  ORDER BY pipeline.name`

	var res sdk.IDNames
	if _, err := db.Select(&res, query, projID); err != nil {
		if err == sql.ErrNoRows {
			return res, nil
		}
		return nil, sdk.WrapError(err, "pipeline.loadpipelinenames")
	}

	return res, nil
}

func updateParamInList(params []sdk.Parameter, paramAction sdk.Parameter) (bool, []sdk.Parameter) {
	for i := range params {
		p := &params[i]
		if p.Name == paramAction.Name {
			p.Type = paramAction.Type
			return true, params
		}
	}
	return false, params
}

// UpdatePipeline update the pipeline
func UpdatePipeline(db gorp.SqlExecutor, p *sdk.Pipeline) error {
	now := time.Now()
	p.LastModified = now.Unix()
	rx := sdk.NamePatternRegex
	if !rx.MatchString(p.Name) {
		return sdk.NewErrorFrom(sdk.ErrInvalidName, "pipeline name should match %s", sdk.NamePattern)
	}

	//Update pipeline
	query := `UPDATE pipeline SET name=$1, description = $2, last_modified=$4, from_repository=$5 WHERE id=$3`
	_, err := db.Exec(query, p.Name, p.Description, p.ID, now, p.FromRepository)
	return sdk.WithStack(err)
}

// InsertPipeline inserts pipeline informations in database
func InsertPipeline(db gorp.SqlExecutor, p *sdk.Pipeline) error {
	query := `INSERT INTO pipeline (name, description, project_id, last_modified, from_repository) VALUES ($1, $2, $3, current_timestamp, $4) RETURNING id`

	rx := sdk.NamePatternRegex
	if !rx.MatchString(p.Name) {
		return sdk.NewErrorFrom(sdk.ErrInvalidName, "pipeline name should match %s", sdk.NamePattern)
	}

	if p.ProjectID == 0 {
		return sdk.WithStack(sdk.ErrInvalidProject)
	}

	if err := db.QueryRow(query, p.Name, p.Description, p.ProjectID, p.FromRepository).Scan(&p.ID); err != nil {
		return sdk.WithStack(err)
	}

	for i := range p.Parameter {
		if err := InsertParameterInPipeline(db, p.ID, &p.Parameter[i]); err != nil {
			return sdk.WithStack(err)
		}
	}

	return nil
}

// ExistPipeline Check if the given pipeline exist in database
func ExistPipeline(db gorp.SqlExecutor, projectID int64, name string) (bool, error) {
	query := `SELECT COUNT(id) FROM pipeline WHERE pipeline.project_id = $1 AND pipeline.name = $2`

	var nb int64
	err := db.QueryRow(query, projectID, name).Scan(&nb)
	if err != nil {
		return false, err
	}
	if nb != 0 {
		return true, nil
	}
	return false, nil
}

// LoadAllNamesByFromRepository returns all pipeline names for a repository
func LoadAllNamesByFromRepository(db gorp.SqlExecutor, projID int64, fromRepository string) (sdk.IDNames, error) {
	if fromRepository == "" {
		return nil, sdk.WithData(sdk.ErrUnknownError, "could not call LoadAllNamesByFromRepository with empty fromRepository")
	}
	query := `SELECT pipeline.id, pipeline.name
			  FROM pipeline
			  WHERE project_id = $1 AND from_repository = $2
			  ORDER BY pipeline.name`

	var res sdk.IDNames
	if _, err := db.Select(&res, query, projID, fromRepository); err != nil {
		if err == sql.ErrNoRows {
			return res, nil
		}
		return nil, sdk.WrapError(err, "pipeline.LoadAllNamesByFromRepository")
	}

	return res, nil
}

// ResetFromRepository reset fromRepository for all pipelines using the same fromRepository in a given project
func ResetFromRepository(db gorp.SqlExecutor, projID int64, fromRepository string) error {
	if fromRepository == "" {
		return sdk.WithData(sdk.ErrUnknownError, "could not call LoadAllNamesByFromRepository with empty fromRepository")
	}
	query := `UPDATE pipeline SET from_repository='' WHERE project_id = $1 AND from_repository = $2`
	_, err := db.Exec(query, projID, fromRepository)
	return sdk.WithStack(err)
}
