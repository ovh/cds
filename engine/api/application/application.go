package application

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// LoadApplicationsRequestAdmin defines the query to load all applications in a project with its activity
const LoadApplicationsRequestAdmin = `
SELECT  project.id as projid, project.name as projname, project.last_modified as projlast_modified,
	application.name as appname, application.id as appid, application.last_modified as applast_modified,
	repo_fullname as repofullname,
	repositories_manager.id as rmid,  repositories_manager.name as rmname,  repositories_manager.type as rmType, repositories_manager.url as rmurl, repositories_manager.data as rmdata
FROM application
LEFT JOIN project ON project.id = application.project_id
LEFT OUTER JOIN repositories_manager on repositories_manager.id = application.repositories_manager_id
WHERE project.projectkey = $1
ORDER BY application.name ASC
`

// LoadApplicationsRequestNormalUser defines the query to load all applications in a project with its activity
const LoadApplicationsRequestNormalUser = `
SELECT  distinct project.id as projid, project.name as projname, project.last_modified as projlast_modified,
	application.name as appname, application.id as appid, application.last_modified as applast_modified,
	repo_fullname as repofullname,
	repositories_manager.id as rmid,  repositories_manager.name as rmname,  repositories_manager.type as rmType, repositories_manager.url as rmurl, repositories_manager.data as rmdata
FROM application
LEFT JOIN project ON project.id = application.project_id
LEFT OUTER JOIN repositories_manager on repositories_manager.id = application.repositories_manager_id
JOIN application_group ON application_group.application_id = application.id
JOIN group_user ON group_user.group_id = application_group.group_id
WHERE project.projectkey = $1 AND group_user.user_id = $2
ORDER BY application.name ASC
`

// LoadApplications load all application from the given project
func LoadApplications(db gorp.SqlExecutor, projectKey string, withPipelines, withVariables bool, user *sdk.User) ([]sdk.Application, error) {
	apps := []sdk.Application{}
	var err error
	var rows *sql.Rows

	if user == nil || user.Admin {
		query := LoadApplicationsRequestAdmin
		rows, err = db.Query(query, projectKey)
	} else {
		query := LoadApplicationsRequestNormalUser
		rows, err = db.Query(query, projectKey, user.ID)
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return apps, sdk.ErrApplicationNotFound
		}
		return apps, err
	}
	defer rows.Close()

	for rows.Next() {
		var projID int64
		var projName string
		var app sdk.Application

		//data for repositories_manager
		var rmID sql.NullInt64
		var rmType, rmName, rmURL, rmData, repoFullname sql.NullString
		var rm *sdk.RepositoriesManager
		var lastModified time.Time
		var appLastModified time.Time

		rows.Scan(&projID, &projName, &lastModified, &app.Name, &app.ID, &appLastModified, &repoFullname, &rmID, &rmName, &rmType, &rmURL, &rmData)
		//check data for repositories_manager
		if rmID.Valid && rmType.Valid && rmName.Valid && rmURL.Valid {
			rm, err = repositoriesmanager.New(sdk.RepositoriesManagerType(rmType.String), rmID.Int64, rmName.String, rmURL.String, map[string]string{}, rmData.String)
			if err != nil {
				log.Warning("LoadApplications> Error loading repositories manager %s", err)
				//anyway, continue loading apps...
			}
			if repoFullname.Valid {
				app.RepositoryFullname = repoFullname.String
			}
		}

		app.ProjectKey = projectKey
		app.RepositoriesManager = rm
		app.LastModified = appLastModified.Unix()

		apps = append(apps, app)
	}

	if withVariables {
		for i := range apps {
			var errV error
			apps[i].Variable, errV = GetAllVariableByID(db, apps[i].ID)
			if errV != nil {
				return apps, errV
			}
		}
	}

	if withPipelines {
		for i := range apps {
			pipelines, err := GetAllPipelinesByID(db, apps[i].ID)
			if err != nil && err != sdk.ErrNoAttachedPipeline {
				return apps, err
			}
			apps[i].Pipelines = pipelines
		}
	}
	return apps, nil
}

// CountApplicationByProject Count the number of applications in the given project
func CountApplicationByProject(db gorp.SqlExecutor, projectID int64) (int, error) {
	var count int
	query := `SELECT COUNT(application.id)
		  FROM application
		  WHERE application.project_id =$1`
	err := db.QueryRow(query, projectID).Scan(&count)
	return count, err
}

// LoadApplicationByName load the given application
func LoadApplicationByName(db gorp.SqlExecutor, projectKey, appName string, fargs ...FuncArg) (*sdk.Application, error) {
	var app sdk.Application
	app.ProjectKey = projectKey
	var k = cache.Key("application", projectKey, appName)
	//cache.Get(k, &app)
	//FIXME Cache

	if app.ID == 0 {
		query := `SELECT application.id, application.name, application.last_modified,
			repo_fullname as repofullname,
			repositories_manager.id as rmid,  repositories_manager.name as rmname,  repositories_manager.type as rmType, repositories_manager.url as rmurl, repositories_manager.data as rmdata
		  FROM application
		  JOIN project ON project.id = application.project_id
			LEFT OUTER JOIN repositories_manager on repositories_manager.id = application.repositories_manager_id
		  WHERE project.projectKey = $1 AND application.name = $2`

		//data for repositories_manager
		var rmID sql.NullInt64
		var rmType, rmName, rmURL, rmData, repoFullname sql.NullString
		var rm *sdk.RepositoriesManager
		var lastModified time.Time

		err := db.QueryRow(query, projectKey, appName).Scan(&app.ID, &app.Name, &lastModified, &repoFullname, &rmID, &rmName, &rmType, &rmURL, &rmData)
		if err != nil {
			if err == sql.ErrNoRows {
				return &app, sdk.ErrApplicationNotFound
			}
			return &app, err
		}
		app.LastModified = lastModified.Unix()

		//check data for repositories_manager
		if rmID.Valid && rmType.Valid && rmName.Valid && rmURL.Valid {
			rm, err = repositoriesmanager.New(sdk.RepositoriesManagerType(rmType.String), rmID.Int64, rmName.String, rmURL.String, map[string]string{}, rmData.String)
			if err != nil {
				log.Warning("LoadApplications> Error loading repositories manager %s", err)
				//anyway, continue loading apps...
			}
			if repoFullname.Valid {
				app.RepositoryFullname = repoFullname.String
			}
		}
		app.RepositoriesManager = rm
		//Put application in cache
		cache.Set(k, app)
	}

	err := loadDependencies(db, &app, fargs...)

	return &app, err
}

// LoadApplicationByID load the given application
func LoadApplicationByID(db gorp.SqlExecutor, applicationID int64, fargs ...FuncArg) (*sdk.Application, error) {
	var app sdk.Application
	query := `
			SELECT
					application.id, application.name, application.last_modified, application.repo_fullname,
					repositories_manager.id as rmid,  repositories_manager.name as rmname,
					repositories_manager.type as rmType, repositories_manager.url as rmurl,
					repositories_manager.data as rmdata,
					project.projectkey
		  FROM application
		  JOIN project ON project.ID = application.project_id
		  LEFT OUTER JOIN repositories_manager on repositories_manager.id = application.repositories_manager_id
		  WHERE application.id = $1
		  ORDER by application.name`

	//data for repositories_manager
	var rmID sql.NullInt64
	var rmType, rmName, rmURL, rmData, repoFullname sql.NullString
	var rm *sdk.RepositoriesManager
	var lastModified time.Time
	if err := db.QueryRow(query, applicationID).Scan(&app.ID, &app.Name, &lastModified, &repoFullname, &rmID, &rmName, &rmType, &rmURL, &rmData, &app.ProjectKey); err != nil {
		if err == sql.ErrNoRows {
			return &app, sdk.ErrApplicationNotFound
		}
		return &app, err
	}
	app.LastModified = lastModified.Unix()
	//check data for repositories_manager
	if rmID.Valid && rmType.Valid && rmName.Valid && rmURL.Valid {
		var err error
		rm, err = repositoriesmanager.New(sdk.RepositoriesManagerType(rmType.String), rmID.Int64, rmName.String, rmURL.String, map[string]string{}, rmData.String)
		if err != nil {
			log.Warning("LoadApplications> Error loading repositories manager %s", err)
			//anyway, continue loading apps...
		}
		if repoFullname.Valid {
			app.RepositoryFullname = repoFullname.String
		}
	}
	app.RepositoriesManager = rm

	return &app, loadDependencies(db, &app, fargs...)
}

func loadDependencies(db gorp.SqlExecutor, app *sdk.Application, fargs ...FuncArg) error {
	variables, errVariable := GetAllVariableByID(db, app.ID, fargs...)
	if errVariable != nil {
		return errVariable
	}
	app.Variable = variables

	if err := LoadGroupByApplication(db, app); err != nil {
		return err
	}

	pipelines, err := GetAllPipelinesByID(db, app.ID)
	if err != nil {
		return err
	}

	app.Pipelines = pipelines
	return nil
}

// InsertApplication Insert new application
func InsertApplication(db gorp.SqlExecutor, project *sdk.Project, app *sdk.Application) error {
	if app.Name == "" {
		return sdk.ErrInvalidName
	}

	query := `INSERT INTO application (name, project_id) VALUES($1, $2) RETURNING id`
	if err := db.QueryRow(query, app.Name, project.ID).Scan(&app.ID); err != nil {
		if errPG, ok := err.(*pq.Error); ok && errPG.Code == "23505" {
			return sdk.ErrApplicationExist
		}
		return err
	}
	// Update project
	query = `
		UPDATE project
		SET last_modified = current_timestamp
		WHERE id=$1
	`
	_, err := db.Exec(query, project.ID)
	return err
}

// UpdateApplication Update an application
func UpdateApplication(db gorp.SqlExecutor, application *sdk.Application) error {
	query := `UPDATE application SET name=$1, last_modified=current_timestamp WHERE id=$2`
	_, err := db.Exec(query, application.Name, application.ID)
	if err != nil {
		return err
	}
	var lastModified time.Time
	// Update project
	query = `
		UPDATE project
		SET last_modified = current_timestamp
		WHERE id IN (
			select project_id from application where id = $1
		)
		RETURNING last_modified

	`
	err = db.QueryRow(query, application.ID).Scan(&lastModified)
	if err == nil {
		application.LastModified = lastModified.Unix()
	}
	return err
}

// UpdateLastModified Update last_modified column in application table
func UpdateLastModified(db gorp.SqlExecutor, app *sdk.Application) error {
	query := `
		UPDATE application SET last_modified=current_timestamp WHERE id = $1 RETURNING last_modified
	`
	var lastModified time.Time
	err := db.QueryRow(query, app.ID).Scan(&lastModified)
	if err == nil {
		app.LastModified = lastModified.Unix()
	}
	return err
}

// DeleteApplication Delete the given application
func DeleteApplication(db gorp.SqlExecutor, applicationID int64) error {

	// Delete variables
	if err := DeleteAllVariable(db, applicationID); err != nil {
		log.Warning("DeleteApplication> Cannot delete application variable: %s\n", err)
		return err
	}

	// Delete groups
	query := `DELETE FROM application_group WHERE application_id = $1`
	if _, err := db.Exec(query, applicationID); err != nil {
		log.Warning("DeleteApplication> Cannot delete application gorup: %s\n", err)
		return err
	}

	// Delete application_pipeline
	if err := DeleteAllApplicationPipeline(db, applicationID); err != nil {
		log.Warning("DeleteApplication> Cannot delete application pipeline: %s\n", err)
		return err
	}

	// Delete pipeline builds
	var ids []int64
	query = `SELECT id FROM pipeline_build WHERE application_id = $1`
	rows, err := db.Query(query, applicationID)
	if err != nil {
		return fmt.Errorf("DeleteApplication> Cannot select application pipeline build> %s\n", err)
	}
	var id int64
	for rows.Next() {
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		if err := pipeline.DeletePipelineBuildByID(db, id); err != nil {
			return fmt.Errorf("DeleteApplication> Cannot delete pb %d> %s", id, err)
		}
	}

	// Delete application artifact left
	query = `DELETE FROM artifact WHERE application_id = $1`
	if _, err = db.Exec(query, applicationID); err != nil {
		log.Warning("DeleteApplication> Cannot delete old artifacts: %s\n", err)
		return err
	}

	// Delete hook
	query = `DELETE FROM hook WHERE application_id = $1`
	if _, err := db.Exec(query, applicationID); err != nil {
		log.Warning("DeleteApplication> Cannot delete hook: %s\n", err)
		return err
	}

	// Delete poller execution
	query = `DELETE FROM poller_execution WHERE application_id = $1`
	if _, err := db.Exec(query, applicationID); err != nil {
		log.Warning("DeleteApplication> Cannot delete poller execution: %s\n", err)
		return err
	}

	// Delete poller
	query = `DELETE FROM poller WHERE application_id = $1`
	if _, err := db.Exec(query, applicationID); err != nil {
		log.Warning("DeleteApplication> Cannot delete poller: %s\n", err)
		return err
	}

	// Delete triggers
	if err := trigger.DeleteApplicationTriggers(db, applicationID); err != nil {
		return err
	}

	query = `DELETE FROM application WHERE id=$1`
	if _, err := db.Exec(query, applicationID); err != nil {
		log.Warning("DeleteApplication> Cannot delete application: %s\n", err)
		return err
	}

	// Update project
	query = `
		UPDATE project
		SET last_modified = current_timestamp
		WHERE id IN (
			select project_id from application where id = $1
		)
	`
	_, err = db.Exec(query, applicationID)
	return err
}

// LoadGroupByApplication loads all the groups on the given application
func LoadGroupByApplication(db gorp.SqlExecutor, application *sdk.Application) error {
	application.ApplicationGroups = []sdk.GroupPermission{}
	query := `SELECT "group".id, "group".name, application_group.role FROM "group"
	 		  JOIN application_group ON application_group.group_id = "group".id
	 		  WHERE application_group.application_id = $1 ORDER BY "group".name ASC`

	rows, err := db.Query(query, application.ID)
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
		application.ApplicationGroups = append(application.ApplicationGroups, sdk.GroupPermission{
			Group:      group,
			Permission: perm,
		})
	}
	return nil
}

// LoadApplicationByPipeline Load application where pipeline is attached
func LoadApplicationByPipeline(db gorp.SqlExecutor, pipelineID int64) ([]sdk.Application, error) {
	applications := []sdk.Application{}
	query := `SELECT application.id, application.name, application.last_modified
		 FROM application
		 JOIN application_pipeline ON application.id = application_pipeline.application_id
		 WHERE application_pipeline.pipeline_id = $1
		 ORDER BY application.name`
	rows, err := db.Query(query, pipelineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var app sdk.Application
		var lastModified time.Time
		err = rows.Scan(&app.ID, &app.Name, &lastModified)
		if err != nil {
			return nil, err
		}
		app.LastModified = lastModified.Unix()
		applications = append(applications, app)
	}
	return applications, nil
}

// LoadApplicationByGroup loads all applications where group has access
func LoadApplicationByGroup(db gorp.SqlExecutor, group *sdk.Group) error {
	query := `SELECT project.projectKey,
	                 application.name,
	                 application.id,
					 application_group.role,
					 application.last_modified
	          FROM application
	          JOIN application_group ON application_group.application_id = application.id
	 	  JOIN project ON application.project_id = project.id
	 	  WHERE application_group.group_id = $1
	 	  ORDER BY application.name ASC`
	rows, err := db.Query(query, group.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var application sdk.Application
		var perm int
		var lastModified time.Time
		err = rows.Scan(&application.ProjectKey, &application.Name, &application.ID, &perm, &lastModified)
		if err != nil {
			return err
		}
		application.LastModified = lastModified.Unix()
		group.ApplicationGroups = append(group.ApplicationGroups, sdk.ApplicationGroup{
			Application: application,
			Permission:  perm,
		})
	}
	return nil
}

// PipelineAttached checks wether a pipeline is attached to given application
func PipelineAttached(db gorp.SqlExecutor, appID, pipID int64) (bool, error) {
	query := `SELECT id FROM application_pipeline WHERE application_id= $1 AND pipeline_id = $2`
	var id int64

	err := db.QueryRow(query, appID, pipID).Scan(&id)
	if err != nil && err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// AddKeyPairToApplication generate a ssh key pair and add them as application variables
func AddKeyPairToApplication(db gorp.SqlExecutor, app *sdk.Application, keyname string) error {
	pub, priv, errGenerate := keys.Generatekeypair(keyname)
	if errGenerate != nil {
		return errGenerate
	}

	v := sdk.Variable{
		Name:  keyname,
		Type:  sdk.KeyVariable,
		Value: priv,
	}

	if err := InsertVariable(db, app, v); err != nil {
		return err
	}

	p := sdk.Variable{
		Name:  keyname + ".pub",
		Type:  sdk.TextVariable,
		Value: pub,
	}

	return InsertVariable(db, app, p)
}
