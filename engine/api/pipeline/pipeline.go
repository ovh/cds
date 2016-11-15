package pipeline

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ovh/cds/engine/api/build"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/sdk"
)

type structarg struct {
	//clearsecret    bool
	loadstages     bool
	loadparameters bool
}

// UpdatePipelineLastModified Update last_modified date on pipeline
func UpdatePipelineLastModified(db database.Executer, pipelineID int64) error {
	query := "UPDATE pipeline SET last_modified = current_timestamp WHERE id = $1"
	_, err := db.Exec(query, pipelineID)
	return err
}

// CountPipelineByProject Count the number of pipelines for the given project
func CountPipelineByProject(db database.Querier, projectID int64) (int, error) {
	var nbPipelines int
	query := `SELECT count(pipeline.id)
	          FROM pipeline
	 	  WHERE pipeline.project_id = $1`
	err := db.QueryRow(query, projectID).Scan(&nbPipelines)
	return nbPipelines, err
}

// LoadPipeline loads a pipeline from database
func LoadPipeline(db database.Querier, projectKey, name string, deep bool) (*sdk.Pipeline, error) {
	var p sdk.Pipeline

	//Try to find pipeline in cache
	_ = cache.Key("pipeline", projectKey, name)
	//FIXME cache
	//cache.Get(k, &p)
	//if p.ID != 0 && p.Name != "" && len(p.Stages) > 0 {
	//	return &p, nil
	//}

	var pType string
	var lastModified time.Time
	query := `SELECT pipeline.id, pipeline.name, pipeline.project_id, pipeline.type, pipeline.last_modified FROM pipeline
	 		JOIN project on pipeline.project_id = project.id
	 		WHERE pipeline.name = $1 AND project.projectKey = $2`

	err := db.QueryRow(query, name, projectKey).Scan(&p.ID, &p.Name, &p.ProjectID, &pType, &lastModified)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrPipelineNotFound
		}
		return nil, err
	}
	p.LastModified = lastModified.Unix()
	p.Type = sdk.PipelineTypeFromString(pType)
	p.ProjectKey = projectKey

	if deep {
		// load pipeline actions by stage
		err = loadPipelineStage(db, &p)
		if err != nil {
			return nil, fmt.Errorf("cannot loadPipelineStage> %s", err)
		}

		err = loadGroupByPipeline(db, &p)
		if err != nil {
			return nil, fmt.Errorf("cannot loadGroupByPipeline> %s", err)
		}

		parameters, err := GetAllParametersInPipeline(db, p.ID)
		if err != nil {
			return nil, fmt.Errorf("cannot GetAllParametersInpipeline> %s", err)
		}
		p.Parameter = parameters
		//cache.Set(k, p)
	}
	return &p, nil
}

// LoadPipelineByID loads a pipeline from database
func LoadPipelineByID(db database.Querier, pipelineID int64) (*sdk.Pipeline, error) {
	var p sdk.Pipeline
	var pType string
	query := `SELECT pipeline.name, pipeline.type, project.projectKey FROM pipeline
	JOIN project on pipeline.project_id = project.id
	WHERE pipeline.id = $1`

	err := db.QueryRow(query, pipelineID).Scan(&p.Name, &pType, &p.ProjectKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrPipelineNotFound
		}
		return nil, err
	}

	p.Type = sdk.PipelineTypeFromString(pType)
	p.ID = pipelineID
	return &p, nil
}

// DeletePipeline remove given pipeline and all history from database
func DeletePipeline(db database.QueryExecuter, pipelineID int64, userID int64) error {
	err := DeleteAllStage(db, pipelineID, userID)
	if err != nil {
		return err
	}

	// Update project
	query := `
		UPDATE project 
		SET last_modified = current_timestamp
		WHERE id in (
			SELECT project_id from pipeline WHERE id = $1
		)
	`
	_, err = db.Exec(query, pipelineID)
	if err != nil {
		return err
	}

	// Delete pipeline groups
	query = `DELETE FROM pipeline_group WHERE pipeline_id = $1`
	_, err = db.Exec(query, pipelineID)
	if err != nil {
		return err
	}

	err = DeleteAllParameterFromPipeline(db, pipelineID)
	if err != nil {
		return err
	}

	// Delete triggers
	err = trigger.DeletePipelineTriggers(db, pipelineID)
	if err != nil {
		return err
	}

	// Delete test results
	err = build.DeletePipelineTestResults(db, pipelineID)
	if err != nil {
		return err
	}

	var pipelineBuildIDs []int64
	query = `SELECT id FROM pipeline_build where pipeline_id = $1`
	rows, err := db.Query(query, pipelineID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var pbID int64
		err = rows.Scan(&pbID)
		if err != nil {
			return err
		}
		pipelineBuildIDs = append(pipelineBuildIDs, pbID)
	}
	for _, id := range pipelineBuildIDs {
		err = DeletePipelineBuild(db, id)
		if err != nil {
			return err
		}
	}

	// Delete artifacts left
	query = `DELETE FROM artifact WHERE pipeline_id = $1`
	_, err = db.Exec(query, pipelineID)
	if err != nil {
		return err
	}

	// Delete histoy
	query = `DELETE FROM pipeline_history WHERE pipeline_id = $1`
	_, err = db.Exec(query, pipelineID)
	if err != nil {
		return err
	}

	// Delete application_pipeline_notif
	query = `
		DELETE FROM application_pipeline_notif WHERE application_pipeline_id IN (
			SELECT id FROM application_pipeline WHERE pipeline_id = $1
		)`
	if _, err := db.Exec(query, pipelineID); err != nil {
		return err
	}

	// Delete pipeline
	query = `DELETE FROM pipeline WHERE id = $1`
	_, err = db.Exec(query, pipelineID)
	if err != nil {
		return err
	}

	return nil
}

// LoadPipelines loads all pipelines in a project
func LoadPipelines(db database.Querier, projectID int64, loadDependencies bool, user *sdk.User) ([]sdk.Pipeline, error) {
	var pip []sdk.Pipeline

	var rows *sql.Rows
	var err error

	if user.Admin {
		query := `SELECT id, name, project_id, type, last_modified
			  FROM pipeline
			  WHERE project_id = $1
			  ORDER BY pipeline.name`
		rows, err = db.Query(query, projectID)
	} else {
		query := `SELECT distinct(pipeline.id), pipeline.name, pipeline.project_id, pipeline.type, last_modified
			  FROM pipeline
			  JOIN pipeline_group ON pipeline.id = pipeline_group.pipeline_id
			  JOIN group_user ON pipeline_group.group_id = group_user.group_id
			  WHERE group_user.user_id = $1
			  AND pipeline.project_id = $2
			  ORDER by pipeline.name`
		rows, err = db.Query(query, user.ID, projectID)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p sdk.Pipeline
		var pType string
		var lastModified time.Time

		// scan pipeline id
		err = rows.Scan(&p.ID, &p.Name, &p.ProjectID, &pType, &lastModified)
		if err != nil {
			return nil, err
		}
		p.Type = sdk.PipelineTypeFromString(pType)
		p.LastModified = lastModified.Unix()

		if loadDependencies {
			// load pipeline stages
			err = loadPipelineStage(db, &p)
			if err != nil {
				return nil, err
			}

			params, err := GetAllParametersInPipeline(db, p.ID)
			if err != nil {
				return nil, err
			}
			p.Parameter = params
		}

		pip = append(pip, p)
	}

	return pip, err
}

// LoadPipelineByGroup loads all pipelines where group has access
func LoadPipelineByGroup(db database.Querier, group *sdk.Group) error {
	query := `SELECT project.projectKey, pipeline.id, pipeline.name,pipeline_group.role FROM pipeline
	 		  JOIN pipeline_group ON pipeline_group.pipeline_id = pipeline.id
	 		  JOIN project ON pipeline.project_id = project.id
	 		  WHERE pipeline_group.group_id = $1 ORDER BY pipeline.name ASC`
	rows, err := db.Query(query, group.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var pipeline sdk.Pipeline
		var perm int
		err = rows.Scan(&pipeline.ProjectKey, &pipeline.ID, &pipeline.Name, &perm)
		if err != nil {
			return err
		}
		group.PipelineGroups = append(group.PipelineGroups, sdk.PipelineGroup{
			Pipeline:   pipeline,
			Permission: perm,
		})
	}
	return nil
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

func loadGroupByPipeline(db database.Querier, pipeline *sdk.Pipeline) error {
	query := `SELECT "group".id,"group".name,pipeline_group.role FROM "group"
	 		  JOIN pipeline_group ON pipeline_group.group_id = "group".id
	 		  WHERE pipeline_group.pipeline_id = $1 ORDER BY "group".name ASC`

	rows, err := db.Query(query, pipeline.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var group sdk.Group
		var perm int
		err = rows.Scan(&group.ID, &group.Name, &perm)
		if err != nil {
			return err
		}
		pipeline.GroupPermission = append(pipeline.GroupPermission, sdk.GroupPermission{
			Group:      group,
			Permission: perm,
		})
	}
	return nil
}

// UpdatePipeline update the pipeline
func UpdatePipeline(db database.Executer, p *sdk.Pipeline) error {
	// Update project
	query := `
		UPDATE project 
		SET last_modified = current_timestamp
		WHERE id IN (SELECT project_id from pipeline WHERE id = $1)
	`
	_, err := db.Exec(query, p.ID)
	if err != nil {
		return err
	}

	//Update pipeline
	query = `UPDATE pipeline SET name=$1, type=$2, last_modified = current_timestamp WHERE id=$3`
	_, err = db.Exec(query, p.Name, string(p.Type), p.ID)
	return err
}

// InsertPipeline inserts pipeline informations in database
func InsertPipeline(db database.QueryExecuter, p *sdk.Pipeline) error {
	query := `INSERT INTO pipeline (name, project_id, type) VALUES ($1,$2,$3) RETURNING id`

	if p.Name == "" {
		return sdk.ErrInvalidName
	}

	if p.Type != sdk.BuildPipeline && p.Type != sdk.DeploymentPipeline && p.Type != sdk.TestingPipeline {
		return sdk.ErrInvalidType
	}

	if p.ProjectID == 0 {
		return sdk.ErrInvalidProject
	}

	return db.QueryRow(query, p.Name, p.ProjectID, string(p.Type)).Scan(&p.ID)
}

// ExistPipeline Check if the given pipeline exist in database
func ExistPipeline(db database.Querier, projectID int64, name string) (bool, error) {
	query := `SELECT COUNT(id) FROM pipeline WHERE pipeline.project_id = $1 AND pipeline.name= $2`

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
