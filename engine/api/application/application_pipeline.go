package application

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// IsAttached checks if an application is attach to a pipeline given its name
func IsAttached(db gorp.SqlExecutor, projectID, appID int64, pipelineName string) (bool, error) {
	query := `SELECT count(1)
		from application_pipeline, pipeline
		WHERE application_pipeline.pipeline_id = pipeline.id
		AND pipeline.name = $3
		AND pipeline.project_id = $1
		AND application_pipeline.application_id = $2`
	var n int
	if err := db.QueryRow(query, projectID, appID, pipelineName).Scan(&n); err != nil {
		return false, err
	}
	return n == 1, nil
}

// AttachPipeline Attach a pipeline to an application
func AttachPipeline(db gorp.SqlExecutor, appID, pipelineID int64) (int64, error) {
	query := `INSERT INTO application_pipeline(application_id, pipeline_id, args) VALUES($1, $2, $3) RETURNING id`
	var id int64
	if err := db.QueryRow(query, appID, pipelineID, "[]").Scan(&id); err != nil {
		if errPG, ok := err.(*pq.Error); ok && errPG.Code == database.ViolateUniqueKeyPGCode {
			return 0, sdk.ErrPipelineAlreadyAttached
		}
	}
	return id, nil
}

// GetAllPipelines Get all pipelines for the given application
func GetAllPipelines(db gorp.SqlExecutor, projectKey, applicationName string) ([]sdk.Pipeline, error) {
	pipelines := []sdk.Pipeline{}
	query := `SELECT pipeline.name
	          FROM application_pipeline
	          JOIN application ON application.id = application_pipeline.application_id
	          JOIN project ON project.id = application.project_id
	          JOIN pipeline ON pipeline.id = application_pipeline.pipeline_id
	          WHERE project.projectkey = $1 AND application.name = $2
	          ORDER BY pipeline.name
						LIMIT 1000`
	rows, err := db.Query(query, projectKey, applicationName)
	if err != nil {
		return pipelines, err
	}
	defer rows.Close()
	for rows.Next() {
		var p sdk.Pipeline
		err = rows.Scan(&p.Name)
		if err != nil {
			return nil, err
		}

		pipelines = append(pipelines, p)
	}
	return pipelines, nil
}

// GetAllPipelinesByID Get all pipelines for the given application
func GetAllPipelinesByID(db gorp.SqlExecutor, applicationID int64) ([]sdk.ApplicationPipeline, error) {
	appPipelines := []sdk.ApplicationPipeline{}
	query := `SELECT application_pipeline.id, pipeline.id, pipeline.name, application_pipeline.args, pipeline.type, application_pipeline.last_modified, pipeline.last_modified
	          FROM application_pipeline
	          JOIN application ON application.id = application_pipeline.application_id
	          JOIN pipeline ON pipeline.id = application_pipeline.pipeline_id
	          WHERE application.id = $1
	          ORDER BY pipeline.name
						LIMIT 1000`
	rows, err := db.Query(query, applicationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return appPipelines, sdk.ErrNoAttachedPipeline
		}
		return appPipelines, err
	}
	defer rows.Close()
	for rows.Next() {
		var p sdk.ApplicationPipeline
		var args string
		var lastModified, pLastModified time.Time
		err = rows.Scan(&p.ID, &p.Pipeline.ID, &p.Pipeline.Name, &args, &p.Pipeline.Type, &lastModified, &pLastModified)
		if err != nil {
			return nil, err
		}
		p.LastModified = lastModified.Unix()
		p.Pipeline.LastModified = pLastModified.Unix()
		err := json.Unmarshal([]byte(args), &p.Parameters)
		if err != nil {
			return nil, err
		}
		//TODO: Uncypher parameters here

		appPipelines = append(appPipelines, p)
	}

	for i := range appPipelines {
		params, err := pipeline.GetAllParametersInPipeline(db, appPipelines[i].Pipeline.ID)
		if err != nil {
			return nil, err
		}
		appPipelines[i].Pipeline.Parameter = params
	}

	return appPipelines, nil
}

// DeleteAllApplicationPipeline Detach all pipeline
func DeleteAllApplicationPipeline(db gorp.SqlExecutor, applicationID int64) error {
	query := `
		DELETE FROM application_pipeline_notif WHERE application_pipeline_id IN (
			SELECT id FROM application_pipeline WHERE application_id = $1
		)`
	if _, err := db.Exec(query, applicationID); err != nil {
		return err
	}

	query = `DELETE FROM application_pipeline WHERE application_id= $1`
	if _, err := db.Exec(query, applicationID); err != nil {
		return err
	}

	query = `DELETE FROM hook WHERE application_id = $1`
	if _, err := db.Exec(query, applicationID); err != nil {
		return err
	}

	return nil
}

// CountPipeline Count the number of application that use the given pipeline
func CountPipeline(db gorp.SqlExecutor, pipelineID int64) (bool, error) {
	query := `SELECT count(*) FROM application_pipeline WHERE pipeline_id= $1`
	nbApp := -1
	err := db.QueryRow(query, pipelineID).Scan(&nbApp)
	return nbApp != 0, err
}

// RemovePipeline Remove a pipeline from the application
func RemovePipeline(db gorp.SqlExecutor, key, appName, pipelineName string) error {
	query := `SELECT pipeline_build.id FROM pipeline_build
							JOIN application ON application.id = pipeline_build.application_id
							JOIN pipeline ON pipeline.id = pipeline_build.pipeline_id
							JOIN project ON pipeline.project_id = project.id
							WHERE application.name = $1 AND pipeline.name = $2 AND project.projectKey = $3`
	rows, err := db.Query(query, appName, pipelineName, key)
	if err != nil {
		return err
	}
	defer rows.Close()
	var pipelineBuildIDs []int64
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			return err
		}
		pipelineBuildIDs = append(pipelineBuildIDs, id)
	}

	for _, id := range pipelineBuildIDs {
		err := pipeline.DeletePipelineBuildByID(db, id)
		if err != nil {
			return fmt.Errorf("RemovePipeline> cannot delete pb %d> %s", id, err)
		}
	}

	// Delete hook
	query = `DELETE FROM hook
		WHERE
		pipeline_id = (select pipeline.id from pipeline JOIN project ON project.id = pipeline.project_id WHERE pipeline.name = $1 AND projectkey = $3)
		AND
		application_id = (SELECT application.id FROM application JOIN project ON project.id = application.project_id WHERE application.name = $2 AND projectkey = $3)`
	_, err = db.Exec(query, pipelineName, appName, key)
	if err != nil {
		return err
	}

	err = trigger.DeleteApplicationPipelineTriggers(db, key, appName, pipelineName)
	if err != nil {
		return fmt.Errorf("RemovePipeline> cannot delete app trigger> %s", err)
	}

	// Delete application_pipeline_notif
	query = `
		DELETE	FROM application_pipeline_notif
		USING 	application_pipeline, application, project, pipeline
		WHERE 	application_pipeline_notif.application_pipeline_id = application_pipeline.id
		AND 	application.project_id = project.id
		AND 	application.id = application_pipeline.application_id
		AND 	pipeline.id = application_pipeline.pipeline_id
	    AND 	application.name = $1
		AND 	project.projectKey = $2
		AND  	pipeline.name = $3`
	_, err = db.Exec(query, appName, key, pipelineName)
	if err != nil {
		return err
	}

	// Delete scheduler
	query = `
		DELETE 	FROM pipeline_scheduler
		USING	application, project, pipeline
		WHERE 	pipeline_scheduler.application_id = application.id
		AND 	pipeline_scheduler.pipeline_id = pipeline.id
		AND 	application.project_id = project.id
	    AND 	application.name = $1
		AND 	project.projectKey = $2
		AND  	pipeline.name = $3`
	res, err := db.Exec(query, appName, key, pipelineName)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	log.Debug("RemovePipeline> removed %d pipeline_scheduler", n)

	// Delete poller
	query = `
		DELETE 	FROM poller
		USING	application, project, pipeline
		WHERE 	poller.application_id = application.id
		AND 	poller.pipeline_id = pipeline.id
		AND 	application.project_id = project.id
	    AND 	application.name = $1
		AND 	project.projectKey = $2
		AND  	pipeline.name = $3`
	res, err = db.Exec(query, appName, key, pipelineName)
	if err != nil {
		return err
	}
	n, _ = res.RowsAffected()
	log.Debug("RemovePipeline> removed %d poller", n)

	// Delete poller_execution
	query = `
		DELETE 	FROM poller_execution
		USING	application, project, pipeline
		WHERE 	poller_execution.application_id = application.id
		AND 	poller_execution.pipeline_id = pipeline.id
		AND 	application.project_id = project.id
	    AND 	application.name = $1
		AND 	project.projectKey = $2
		AND  	pipeline.name = $3`
	res, err = db.Exec(query, appName, key, pipelineName)
	if err != nil {
		return err
	}
	n, _ = res.RowsAffected()
	log.Debug("RemovePipeline> removed %d poller_execution", n)

	// Delete application_pipeline link
	query = `DELETE FROM application_pipeline
	          USING application, project, pipeline
	          WHERE application.project_id = project.id AND application.id = application_pipeline.application_id AND pipeline.id = application_pipeline.pipeline_id
	          AND application.name = $1 AND project.projectKey = $2 AND  pipeline.name = $3`
	result, err := db.Exec(query, appName, key, pipelineName)
	if err != nil {
		return fmt.Errorf("RemovePipeline> cannot application_pipeline link> %s", err)
	}
	rowAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowAffected == 0 {
		return sdk.ErrNoAttachedPipeline
	}

	return nil
}

// GetAllPipelineParam Get all the pipeline parameters
//func GetAllPipelineParam(db gorp.SqlExecutor, applicationID, pipelineID int64, fargs ...FuncArg) ([]sdk.Parameter, error) {
func GetAllPipelineParam(db gorp.SqlExecutor, applicationID, pipelineID int64) ([]sdk.Parameter, error) {
	var params []sdk.Parameter
	query := `SELECT args FROM application_pipeline WHERE application_id=$1 AND pipeline_id=$2`

	var args string
	err := db.QueryRow(query, applicationID, pipelineID).Scan(&args)
	if err != nil {
		return params, err
	}

	err = json.Unmarshal([]byte(args), &params)
	if err != nil {
		return nil, err
	}
	return params, nil
}
