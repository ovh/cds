package pipeline

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/sdk"
)

type structarg struct {
	loadstages     bool
	loadparameters bool
}

// LoadPipeline loads a pipeline from database
func LoadPipeline(db gorp.SqlExecutor, projectKey, name string, deep bool) (*sdk.Pipeline, error) {
	var p sdk.Pipeline

	var lastModified time.Time
	query := `SELECT pipeline.id, pipeline.name, pipeline.description, pipeline.project_id, pipeline.last_modified, pipeline.from_repository
			FROM pipeline
	 			JOIN project on pipeline.project_id = project.id
	 		WHERE pipeline.name = $1 AND project.projectKey = $2`

	if err := db.QueryRow(query, name, projectKey).Scan(&p.ID, &p.Name, &p.Description, &p.ProjectID, &lastModified, &p.FromRepository); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WithStack(sdk.ErrPipelineNotFound)
		}
		return nil, sdk.WithStack(err)
	}
	p.LastModified = lastModified.Unix()
	p.ProjectKey = projectKey

	if deep {
		if err := loadPipelineDependencies(context.TODO(), db, &p); err != nil {
			return nil, err
		}
	} else {
		parameters, err := GetAllParametersInPipeline(context.TODO(), db, p.ID)
		if err != nil {
			return nil, err
		}
		p.Parameter = parameters
	}

	return &p, nil
}

// LoadPipelineByID loads a pipeline from database
func LoadPipelineByID(ctx context.Context, db gorp.SqlExecutor, pipelineID int64, deep bool) (*sdk.Pipeline, error) {
	var end func()
	ctx, end = observability.Span(ctx, "pipeline.LoadPipelineByID",
		observability.Tag(observability.TagPipelineID, pipelineID),
		observability.Tag(observability.TagPipelineDeep, deep),
	)
	defer end()

	var lastModified time.Time
	var p sdk.Pipeline
	query := `SELECT pipeline.name, pipeline.description, project.projectKey, pipeline.last_modified, pipeline.from_repository
	FROM pipeline
		JOIN project on pipeline.project_id = project.id
	WHERE pipeline.id = $1`

	err := db.QueryRow(query, pipelineID).Scan(&p.Name, &p.Description, &p.ProjectKey, &lastModified, &p.FromRepository)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrPipelineNotFound
		}
		return nil, err
	}
	p.LastModified = lastModified.Unix()
	p.ID = pipelineID

	if deep {
		if err := loadPipelineDependencies(ctx, db, &p); err != nil {
			return nil, err
		}
	} else {
		parameters, err := GetAllParametersInPipeline(ctx, db, p.ID)
		if err != nil {
			return nil, err
		}
		p.Parameter = parameters
	}

	return &p, nil
}

// LoadByWorkerModelName loads pipelines from database for a given worker model name
func LoadByWorkerModelName(db gorp.SqlExecutor, workerModelName string, u *sdk.User) ([]sdk.Pipeline, error) {
	var pips []sdk.Pipeline
	query := `
	SELECT DISTINCT pipeline.*, project.projectkey AS projectKey
		FROM action_requirement
			JOIN pipeline_action ON action_requirement.action_id = pipeline_action.action_id
			JOIN pipeline_stage ON pipeline_action.pipeline_stage_id = pipeline_stage.id
			JOIN pipeline ON pipeline.id = pipeline_stage.pipeline_id
			JOIN project ON project.id = pipeline.project_id
		WHERE action_requirement.type = 'model' AND action_requirement.value = $1`
	args := []interface{}{workerModelName}

	if !u.Admin {
		query = `
	SELECT DISTINCT pipeline.*, project.projectkey AS projectKey
		FROM action_requirement
			JOIN pipeline_action ON action_requirement.action_id = pipeline_action.action_id
			JOIN pipeline_stage ON pipeline_action.pipeline_stage_id = pipeline_stage.id
			JOIN pipeline ON pipeline.id = pipeline_stage.pipeline_id
			JOIN project ON project.id = pipeline.project_id
		WHERE action_requirement.type = 'model'
			AND action_requirement.value = $1
			AND project.id IN (
				SELECT project_group.project_id
					FROM project_group
				WHERE
					project_group.group_id = ANY(string_to_array($2, ',')::int[])
					OR
					$3 = ANY(string_to_array($2, ',')::int[])
			)`
		args = append(args, gorpmapping.IDsToQueryString(sdk.GroupsToIDs(u.Groups)), group.SharedInfraGroup.ID)
	}

	if _, err := db.Select(&pips, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "Unable to load pipelines linked to worker model name %s", workerModelName)
	}

	return pips, nil
}

// LoadByWorkflowID loads pipelines from database for a given workflow id
func LoadByWorkflowID(db gorp.SqlExecutor, workflowID int64) ([]sdk.Pipeline, error) {
	pips := []sdk.Pipeline{}
	query := `SELECT DISTINCT pipeline.*
	FROM pipeline
		JOIN workflow_node ON pipeline.id = workflow_node.pipeline_id
		JOIN workflow ON workflow_node.workflow_id = workflow.id
	WHERE workflow.id = $1`

	if _, err := db.Select(&pips, query, workflowID); err != nil {
		if err == sql.ErrNoRows {
			return pips, nil
		}
		return nil, sdk.WrapError(err, "Unable to load pipelines linked to workflow id %d", workflowID)
	}

	return pips, nil
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
func DeletePipeline(db gorp.SqlExecutor, pipelineID int64, userID int64) error {
	if err := DeleteAllStage(db, pipelineID, userID); err != nil {
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

// LoadPipelines loads all pipelines in a project
func LoadPipelines(db gorp.SqlExecutor, projectID int64, loadDependencies bool) ([]sdk.Pipeline, error) {
	var pip []sdk.Pipeline
	query := `SELECT id, name, description, project_id, last_modified, from_repository
			  FROM pipeline
			  WHERE project_id = $1
			  ORDER BY pipeline.name`

	rows, errquery := db.Query(query, projectID)
	if errquery != nil {
		return nil, errquery
	}
	defer rows.Close()

	for rows.Next() {
		var p sdk.Pipeline
		var lastModified time.Time

		// scan pipeline id
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.ProjectID, &lastModified, &p.FromRepository); err != nil {
			return nil, err
		}
		p.LastModified = lastModified.Unix()

		if loadDependencies {
			// load pipeline stages
			if err := LoadPipelineStage(context.TODO(), db, &p); err != nil {
				return nil, err
			}
		}

		pip = append(pip, p)
	}

	for i := range pip {
		params, err := GetAllParametersInPipeline(context.TODO(), db, pip[i].ID)
		if err != nil {
			return nil, err
		}
		pip[i].Parameter = params
	}

	return pip, nil
}

// LoadAllNames returns all pipeline names
func LoadAllNames(db gorp.SqlExecutor, store cache.Store, projID int64) (sdk.IDNames, error) {
	query := `SELECT pipeline.id, pipeline.name, pipeline.description
			  FROM pipeline
			  WHERE project_id = $1
			  ORDER BY pipeline.name`

	var res sdk.IDNames
	if _, err := db.Select(&res, query, projID); err != nil {
		if err == sql.ErrNoRows {
			return res, nil
		}
		return nil, sdk.WrapError(err, "application.loadpipelinenames")
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
		return sdk.NewError(sdk.ErrInvalidName, fmt.Errorf("Invalid pipeline name. It should match %s", sdk.NamePattern))
	}

	//Update pipeline
	query := `UPDATE pipeline SET name=$1, description = $2, last_modified=$4, from_repository=$5 WHERE id=$3`
	_, err := db.Exec(query, p.Name, p.Description, p.ID, now, p.FromRepository)
	return sdk.WithStack(err)
}

// InsertPipeline inserts pipeline informations in database
func InsertPipeline(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, p *sdk.Pipeline, u *sdk.User) error {
	query := `INSERT INTO pipeline (name, description, project_id, last_modified, from_repository) VALUES ($1, $2, $3, current_timestamp, $4) RETURNING id`

	rx := sdk.NamePatternRegex
	if !rx.MatchString(p.Name) {
		return sdk.NewError(sdk.ErrInvalidName, fmt.Errorf("Invalid pipeline name. It should match %s", sdk.NamePattern))
	}

	if p.ProjectID == 0 {
		return sdk.WrapError(sdk.ErrInvalidProject, "InsertPipeline>")
	}

	if err := db.QueryRow(query, p.Name, p.Description, p.ProjectID, p.FromRepository).Scan(&p.ID); err != nil {
		return err
	}

	for i := range p.Parameter {
		if err := InsertParameterInPipeline(db, p.ID, &p.Parameter[i]); err != nil {
			return sdk.WithStack(err)
		}
	}

	event.PublishPipelineAdd(proj.Key, *p, u)

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
