package application

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/sdk"
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
		if errPG, ok := err.(*pq.Error); ok && errPG.Code == "23505" {
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
		var typePipeline string
		var lastModified, pLastModified time.Time
		err = rows.Scan(&p.ID, &p.Pipeline.ID, &p.Pipeline.Name, &args, &typePipeline, &lastModified, &pLastModified)
		if err != nil {
			return nil, err
		}
		p.Pipeline.Type = sdk.PipelineTypeFromString(typePipeline)
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

	// Delete warnings
	query = `DELETE FROM warning
		WHERE
		pip_id = (select pipeline.id from pipeline JOIN project ON project.id = pipeline.project_id WHERE pipeline.name = $1 AND projectkey = $3)
		AND
		app_id = (SELECT application.id FROM application JOIN project ON project.id = application.project_id WHERE application.name = $2 AND projectkey = $3)`
	_, err = db.Exec(query, pipelineName, appName, key)
	if err != nil {
		return err
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

// LoadCDTree Load the continuous delivery pipeline tree for the given application
func LoadCDTree(db gorp.SqlExecutor, projectkey, appName string, user *sdk.User) ([]sdk.CDPipeline, error) {
	cdTrees := []sdk.CDPipeline{}

	// Select root trigger element + non triggered pipeline
	query := `
		SELECT
			projID, projName, projlast_modified,
			appID , appName,
			src_pipeline_id, pipeline.name, pipeline.type,
			src_environment_id, environment.name, true as rootTrigger
		FROM (
			SELECT
				distinct on (src_pipeline_id, src_environment_id)src_pipeline_id ,
				COALESCE(src_environment_id,1) as src_environment_id,
				(src_pipeline_id || '-' || COALESCE(src_environment_id,1) ) as dis,
				application.project_id as projID, project.name as projName, project.last_modified as projlast_modified,
				application.id as appID, application.name as appName
			FROM pipeline_trigger
			JOIN application ON src_application_id = application.id
			JOIN project ON project.id = application.project_id
			WHERE src_application_id = (
				SELECT application.id from application
				JOIN project ON project.id = application.project_id
				WHERE project.projectkey = $1 AND application.name = $2
			)
		) sub
		JOIN pipeline ON pipeline.id = src_pipeline_id
		LEFT JOIN environment ON environment.id = src_environment_id
		WHERE
			dis not in (
				select
					dest_pipeline_id || '-' || COALESCE(dest_environment_id,1)
				FROM pipeline_trigger
				WHERE
					dest_application_id = appID
					AND src_application_id = appID
		)
		UNION
		SELECT
			application.project_id as projID, project.name as projName, project.last_modified as projlast_modified,
			application.id as appID, application.name as appName,
			pipeline.id, pipeline.name, pipeline.type,
			0, 'NoEnv', false as rootTrigger
		FROM pipeline
		JOIN application_pipeline ON application_pipeline.pipeline_id = pipeline.id
		JOIN application ON application.id = application_pipeline.application_id
		JOIN project ON project.id = application.project_id
		WHERE pipeline.id not in (
			-- Not initiate trigger
			select src_pipeline_id
			FROM pipeline_trigger
			WHERE src_application_id = (
				SELECT application.id from application
				JOIN project ON project.id = application.project_id
				WHERE project.projectkey = $1 AND application.name = $2
			)
			UNION
			-- Not call from a trigger in the same app
			select dest_pipeline_id
			FROM  pipeline_trigger
			WHERE dest_application_id = (
				SELECT application.id from application
				JOIN project ON project.id = application.project_id
				WHERE project.projectkey = $1 AND application.name = $2
			) AND src_application_id = (
				SELECT application.id from application
				JOIN project ON project.id = application.project_id
				WHERE project.projectkey = $1 AND application.name = $2
			)
		)
		AND application_pipeline.application_id = (
				SELECT application.id from application
				JOIN project ON project.id = application.project_id
				WHERE project.projectkey = $1 AND application.name = $2
			)`

	rows, err := db.Query(query, projectkey, appName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var root sdk.CDPipeline
		var typePipeline string
		var rootTrigger bool
		var lastModified time.Time

		if err = rows.Scan(&root.Project.ID, &root.Project.Name, &lastModified, &root.Application.ID, &root.Application.Name, &root.Pipeline.ID, &root.Pipeline.Name, &typePipeline,
			&root.Environment.ID, &root.Environment.Name, &rootTrigger); err != nil {
			return nil, err
		}
		root.Pipeline.Type = sdk.PipelineTypeFromString(typePipeline)

		if root.Environment.ID == 0 {
			root.Environment = sdk.DefaultEnv
		}

		if permission.AccessToPipeline(root.Environment.ID, root.Pipeline.ID, user, permission.PermissionRead) {
			if rootTrigger {
				err = getChild(db, &root, user)
				if err != nil {
					return nil, err
				}
			}
			root.Project.Key = projectkey
			root.Project.LastModified = lastModified
			root.Application.Permission = permission.ApplicationPermission(root.Application.ID, user)
			root.Pipeline.Permission = permission.PipelinePermission(root.Pipeline.ID, user)
			if root.Environment.ID != sdk.DefaultEnv.ID {
				root.Environment.Permission = permission.EnvironmentPermission(root.Environment.ID, user)
			}

			cdTrees = append(cdTrees, root)
		}
	}
	return cdTrees, nil
}

func getChild(db gorp.SqlExecutor, parent *sdk.CDPipeline, user *sdk.User) error {
	listTrigger := []sdk.CDPipeline{}

	query := `
	WITH RECURSIVE parent(id, src_application_id, dest_application_id, src_pipeline_id, src_environment_id, dest_pipeline_id, dest_environment_id) AS (
		SELECT
			pt.id, pt.src_application_id, pt.dest_application_id, pt.src_pipeline_id, COALESCE(pt.src_environment_id,1), pt.dest_pipeline_id, COALESCE(pt.dest_environment_id,1), pt.manual,
			srcPip.name as srcPipName, srcPip.type as srcPipType, destPip.name as destPipName, destPip.type as destpipType,
			srcApp.name as srcAppName, destApp.name as destAppName,
			srcEnv.name as srcEnvName, destEnv.name as destEnvName,
			srcProj.id as srcProjID, srcProj.projectkey as srcProjKey, srcProj.name as srcProjName,
			destProj.id as destProjID, destProj.projectkey as destProjKey, destProj.name as destProjName,
			ptp.name as paramName, ptp.type as paramType, ptp.value as paramValue, ptp.description as paramDescription,
			pre.parameter as prerequisiteName, pre.expected_value as prerequisiteValue
		FROM pipeline_trigger as pt
		JOIN pipeline srcPip ON srcPip.id = src_pipeline_id
		JOIN pipeline destPip ON destPip.id = dest_pipeline_id
		JOIN application srcApp ON srcApp.id = src_application_id
		JOIN application destApp ON destApp.id = dest_application_id
		JOIN environment srcEnv ON srcEnv.id = COALESCE(src_environment_id,1)
		JOIN environment destEnv ON destEnv.id = COALESCE(dest_environment_id,1)
		JOIN project as srcProj ON srcProj.id = srcApp.project_id
		JOIN project as destProj ON destProj.id = destApp.project_id
		LEFT JOIN pipeline_trigger_parameter AS ptp ON ptp.pipeline_trigger_id = pt.id
		LEFT JOIN pipeline_trigger_prerequisite pre ON pre.pipeline_trigger_id = pt.id
		WHERE pt.src_application_id = $1 AND pt.src_pipeline_id = $2 AND COALESCE(pt.src_environment_id,1) = $3
		UNION
			SELECT pt.id, pt.src_application_id, pt.dest_application_id, pt.src_pipeline_id, COALESCE(pt.src_environment_id,1), pt.dest_pipeline_id, COALESCE(pt.dest_environment_id,1), pt.manual,
			srcPip.name as srcPipName, srcPip.type as srcPipType, destPip.name as destPipName, destPip.type as destpipType,
			srcApp.name as srcAppName, destApp.name as destAppName,
			srcEnv.name as srcEnvName, destEnv.name as destEnvName,
			srcProj.id as srcProjID, srcProj.projectkey as srcProjKey, srcProj.name as srcProjName,
			destProj.id as destProjID, destProj.projectkey as destProjKey, destProj.name as destProjName,
			ptp.name as paramName, ptp.type as paramType, ptp.value as paramValue, ptp.description as paramDescription,
			pre.parameter as prerequisiteName, pre.expected_value as prerequisiteValue
			FROM parent pr, pipeline_trigger pt
			JOIN pipeline srcPip ON srcPip.id = src_pipeline_id
			JOIN pipeline destPip ON destPip.id = dest_pipeline_id
			JOIN application srcApp ON srcApp.id = src_application_id
			JOIN application destApp ON destApp.id = dest_application_id
			JOIN environment srcEnv ON srcEnv.id = COALESCE(src_environment_id,1)
			JOIN environment destEnv ON destEnv.id = COALESCE(dest_environment_id,1)
			JOIN project as srcProj ON srcProj.id = srcApp.project_id
			JOIN project as destProj ON destProj.id = destApp.project_id
			LEFT JOIN pipeline_trigger_parameter AS ptp ON ptp.pipeline_trigger_id = pt.id
			LEFT JOIN pipeline_trigger_prerequisite pre ON pre.pipeline_trigger_id = pt.id
			WHERE pt.src_pipeline_id = pr.dest_pipeline_id AND COALESCE(pt.src_environment_id,1) = COALESCE(pr.dest_environment_id,1)
	)
	SELECT id,
		src_application_id, dest_application_id,
		src_pipeline_id, src_environment_id, dest_pipeline_id, dest_environment_id,
		manual,
		srcPipName, srcPipType, destPipName, destPipType,
		srcAppName, destAppName,
		srcEnvName, destEnvName,
		srcProjId, srcProjkey, srcProjName, destProjId, destProjKey, destProjName,
		COALESCE(
			json_agg(json_build_object('name', paramName, 'type', paramType, 'value', paramValue, 'description', paramDescription ))
			FILTER (WHERE paramName IS NOT NULL), '[]'
		),
		COALESCE(
			json_agg(json_build_object('parameter', prerequisiteName, 'expected_value', prerequisiteValue))
			FILTER (WHERE prerequisiteName IS NOT NULL), '[]'
		)
	FROM parent
	GROUP BY
		id,
		src_application_id, dest_application_id, srcAppName, destAppName,
		src_pipeline_id, dest_pipeline_id,
		src_environment_id, dest_environment_id, srcEnvName, destEnvName,
		srcProjId, destProjId, srcProjkey, destProjKey, srcProjName, destProjName,
		manual, srcpipname, destpipname, srcpiptype, destpiptype
	ORDER BY srcEnvName;
	`
	rows, errQuery := db.Query(query, parent.Application.ID, parent.Pipeline.ID, parent.Environment.ID)
	if errQuery != nil {
		return errQuery
	}
	defer rows.Close()

	for rows.Next() {
		var child sdk.CDPipeline
		var srcType, destType string
		var params, prerequisites string
		if err := rows.Scan(&child.Trigger.ID,
			&child.Trigger.SrcApplication.ID, &child.Trigger.DestApplication.ID,
			&child.Trigger.SrcPipeline.ID, &child.Trigger.SrcEnvironment.ID, &child.Trigger.DestPipeline.ID, &child.Trigger.DestEnvironment.ID,
			&child.Trigger.Manual,
			&child.Trigger.SrcPipeline.Name, &srcType, &child.Trigger.DestPipeline.Name, &destType,
			&child.Trigger.SrcApplication.Name, &child.Trigger.DestApplication.Name,
			&child.Trigger.SrcEnvironment.Name, &child.Trigger.DestEnvironment.Name,
			&child.Trigger.SrcProject.ID, &child.Trigger.SrcProject.Key, &child.Trigger.SrcProject.Name,
			&child.Trigger.DestProject.ID, &child.Trigger.DestProject.Key, &child.Trigger.DestProject.Name,
			&params, &prerequisites); err != nil {
			return err
		}

		if permission.AccessToPipeline(child.Trigger.DestEnvironment.ID, child.Trigger.DestPipeline.ID, user, permission.PermissionRead) {
			child.Trigger.SrcPipeline.Type = sdk.PipelineTypeFromString(srcType)
			child.Trigger.DestPipeline.Type = sdk.PipelineTypeFromString(destType)

			child.Project = child.Trigger.DestProject
			child.Application = child.Trigger.DestApplication
			child.Pipeline = child.Trigger.DestPipeline
			child.Environment = child.Trigger.DestEnvironment
			if err := json.Unmarshal([]byte(params), &child.Trigger.Parameters); err != nil {
				return err
			}

			if err := json.Unmarshal([]byte(prerequisites), &child.Trigger.Prerequisites); err != nil {
				return err
			}

			listTrigger = append(listTrigger, child)
		}
	}

	buildTreeOrder(parent, listTrigger, user)
	return nil
}

func buildTreeOrder(parent *sdk.CDPipeline, listChild []sdk.CDPipeline, user *sdk.User) sdk.CDPipeline {

	for _, child := range listChild {
		if child.Trigger.SrcProject.ID == parent.Project.ID &&
			child.Trigger.SrcApplication.ID == parent.Application.ID &&
			child.Trigger.SrcPipeline.ID == parent.Pipeline.ID &&
			child.Trigger.SrcEnvironment.ID == parent.Environment.ID {

			child.Application.Permission = permission.ApplicationPermission(child.Application.ID, user)
			child.Pipeline.Permission = permission.PipelinePermission(child.Pipeline.ID, user)
			if child.Environment.ID != sdk.DefaultEnv.ID {
				child.Environment.Permission = permission.EnvironmentPermission(child.Environment.ID, user)
			}
			parent.SubPipelines = append(parent.SubPipelines, buildTreeOrder(&child, listChild, user))
		}
	}
	return *parent
}
