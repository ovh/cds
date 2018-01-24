package application

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
)

// LoadOptionFunc is a type for all options in LoadOptions
type LoadOptionFunc *func(gorp.SqlExecutor, cache.Store, *sdk.Application, *sdk.User) error

// LoadOptions provides all options on project loads functions
var LoadOptions = struct {
	Default                        LoadOptionFunc
	WithVariables                  LoadOptionFunc
	WithVariablesWithClearPassword LoadOptionFunc
	WithPipelines                  LoadOptionFunc
	WithTriggers                   LoadOptionFunc
	WithGroups                     LoadOptionFunc
	WithHooks                      LoadOptionFunc
	WithNotifs                     LoadOptionFunc
	WithKeys                       LoadOptionFunc
	WithClearKeys                  LoadOptionFunc
}{
	Default:                        &loadDefaultDependencies,
	WithVariables:                  &loadVariables,
	WithVariablesWithClearPassword: &loadVariablesWithClearPassword,
	WithPipelines:                  &loadPipelines,
	WithTriggers:                   &loadTriggers,
	WithGroups:                     &loadGroups,
	WithHooks:                      &loadHooks,
	WithNotifs:                     &loadNotifs,
	WithKeys:                       &loadKeys,
	WithClearKeys:                  &loadClearKeys,
}

// LoadOldApplicationWorkflowToClean load application to clean
func LoadOldApplicationWorkflowToClean(db gorp.SqlExecutor) ([]sdk.Application, error) {
	apps := []sdk.Application{}
	query := `SELECT application.* FROM application where workflow_migration = 'CLEANING'`
	if _, err := db.Select(&apps, query); err != nil {
		if err == sql.ErrNoRows {
			return apps, nil
		}
		return nil, sdk.WrapError(err, "LoadOldApplicationWorkflowToClean> Cannot load application to clean")
	}
	return apps, nil
}

// Exists checks if an application given its name exists
func Exists(db gorp.SqlExecutor, projectKey, appName string) (bool, error) {
	count, err := db.SelectInt("SELECT count(1) FROM application join project ON project.id = application.project_id WHERE project.projectkey = $1 AND application.name = $2", projectKey, appName)
	if err != nil {
		return false, err
	}
	return count == 1, nil
}

// LoadByName load an application from DB
func LoadByName(db gorp.SqlExecutor, store cache.Store, projectKey, appName string, u *sdk.User, opts ...LoadOptionFunc) (*sdk.Application, error) {
	var query string
	var args []interface{}

	if u == nil || u.Admin {
		query = `
                SELECT application.*
                FROM application
                JOIN project ON project.id = application.project_id
                WHERE project.projectkey = $1
                AND application.name = $2`
		args = []interface{}{projectKey, appName}
	} else {
		query = `
            SELECT distinct application.*
            FROM application
            JOIN project ON project.id = application.project_id
            JOIN application_group on application.id = application_group.application_id
            WHERE project.projectkey = $1
            AND application.name = $2
            AND (
				application_group.group_id = ANY(string_to_array($3, ',')::int[])
				OR
				$4 = ANY(string_to_array($3, ',')::int[])
			)`
		var groupID string
		for i, g := range u.Groups {
			if i == 0 {
				groupID = fmt.Sprintf("%d", g.ID)
			} else {
				groupID += "," + fmt.Sprintf("%d", g.ID)
			}
		}
		args = []interface{}{projectKey, appName, groupID, group.SharedInfraGroup.ID}
	}

	return load(db, store, projectKey, u, opts, query, args...)
}

// LoadByID load an application from DB
func LoadByID(db gorp.SqlExecutor, store cache.Store, id int64, u *sdk.User, opts ...LoadOptionFunc) (*sdk.Application, error) {
	var query string
	var args []interface{}

	if u == nil || u.Admin {
		query = `
                SELECT application.*
                FROM application
                WHERE application.id = $1`
		args = []interface{}{id}
	} else {
		query = `
            SELECT distinct application.*
            FROM application
            JOIN application_group on application.id = application_group.application_id
            WHERE application.id = $1
            AND (
				application_group.group_id = ANY(string_to_array($2, ',')::int[])
				OR
				$3 = ANY(string_to_array($2, ',')::int[])
			)`
		var groupID string

		for i, g := range u.Groups {
			if i == 0 {
				groupID = fmt.Sprintf("%d", g.ID)
			} else {
				groupID += "," + fmt.Sprintf("%d", g.ID)
			}
		}
		args = []interface{}{id, groupID, group.SharedInfraGroup.ID}
	}

	return load(db, store, "", u, opts, query, args...)
}

// LoadByWorkflowID loads applications from database for a given workflow id
func LoadByWorkflowID(db gorp.SqlExecutor, workflowID int64) ([]sdk.Application, error) {
	apps := []sdk.Application{}
	query := `SELECT DISTINCT application.* FROM application
	JOIN workflow_node_context ON workflow_node_context.application_id = application.id
	JOIN workflow_node ON workflow_node.id = workflow_node_context.workflow_node_id
	JOIN workflow ON workflow.id = workflow_node.workflow_id
	WHERE workflow.id = $1`

	if _, err := db.Select(&apps, query, workflowID); err != nil {
		if err == sql.ErrNoRows {
			return apps, nil
		}
		return nil, sdk.WrapError(err, "LoadByWorkflow> Unable to load applications linked to workflow id %d", workflowID)
	}

	return apps, nil
}

// LoadByEnvName loads applications from database for a given project key and environment name
func LoadByEnvName(db gorp.SqlExecutor, projKey, envName string) ([]sdk.Application, error) {
	apps := []sdk.Application{}
	query := `SELECT DISTINCT application.* FROM application
	JOIN pipeline_trigger ON application.id = pipeline_trigger.src_application_id OR application.id = pipeline_trigger.dest_application_id
	JOIN environment ON environment.id = pipeline_trigger.src_environment_id OR environment.id = pipeline_trigger.dest_environment_id
	JOIN project ON application.project_id = project.id
	WHERE project.projectkey = $1 AND environment.name = $2`

	if _, err := db.Select(&apps, query, projKey, envName); err != nil {
		if err == sql.ErrNoRows {
			return apps, nil
		}
		return nil, sdk.WrapError(err, "LoadByEnvName> Unable to load applications linked to environment %s", envName)
	}

	return apps, nil
}

// LoadByPipeline Load application where pipeline is attached
func LoadByPipeline(db gorp.SqlExecutor, store cache.Store, pipelineID int64, u *sdk.User, opts ...LoadOptionFunc) ([]sdk.Application, error) {
	query := `SELECT distinct application.*
		 FROM application
		 JOIN application_pipeline ON application.id = application_pipeline.application_id
		 WHERE application_pipeline.pipeline_id = $1
		 ORDER BY application.name`
	app, err := loadapplications(db, store, u, opts, query, pipelineID)
	if err != nil {
		return nil, sdk.WrapError(err, "LoadByPipeline (%d)", pipelineID)
	}
	return app, nil
}

func load(db gorp.SqlExecutor, store cache.Store, key string, u *sdk.User, opts []LoadOptionFunc, query string, args ...interface{}) (*sdk.Application, error) {
	dbApp := dbApplication{}
	if err := db.SelectOne(&dbApp, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WrapError(sdk.ErrApplicationNotFound, "application.load")
		}
		return nil, sdk.WrapError(err, "application.load")
	}
	dbApp.ProjectKey = key
	return unwrap(db, store, u, opts, &dbApp)
}

func unwrap(db gorp.SqlExecutor, store cache.Store, u *sdk.User, opts []LoadOptionFunc, dbApp *dbApplication) (*sdk.Application, error) {
	app := sdk.Application(*dbApp)

	if app.ProjectKey == "" {
		pkey, errP := db.SelectStr("select projectkey from project where id = $1", app.ProjectID)
		if errP != nil {
			return nil, sdk.WrapError(errP, "application.unwrap")
		}
		app.ProjectKey = pkey
	}

	if u != nil {
		loadPermission(db, store, &app, u)
	}

	for _, f := range opts {
		if err := (*f)(db, store, &app, u); err != nil && err != sql.ErrNoRows {
			return nil, sdk.WrapError(err, "application.unwrap")
		}
	}
	return &app, nil
}

// Insert add an application id database
func Insert(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, app *sdk.Application, u *sdk.User) error {
	// check application name pattern
	regexp := sdk.NamePatternRegex
	if !regexp.MatchString(app.Name) {
		return sdk.WrapError(sdk.ErrInvalidApplicationPattern, "Insert: Application name %s do not respect pattern %s", app.Name, sdk.NamePattern)
	}

	switch proj.WorkflowMigration {
	case "NOT_BEGUN":
		app.WorkflowMigration = "NOT_BEGUN"
	default:
		app.WorkflowMigration = "DONE"
	}
	app.ProjectID = proj.ID
	app.ProjectKey = proj.Key
	app.LastModified = time.Now()
	dbApp := dbApplication(*app)
	if err := db.Insert(&dbApp); err != nil {
		if errPG, ok := err.(*pq.Error); ok && errPG.Code == "23505" {
			err = sdk.ErrApplicationExist
		}
		return sdk.WrapError(err, "application.Insert %s(%d)", app.Name, app.ID)
	}
	*app = sdk.Application(dbApp)
	return UpdateLastModified(db, store, app, u)
}

// Update updates application id database
func Update(db gorp.SqlExecutor, store cache.Store, app *sdk.Application, u *sdk.User) error {
	rx := sdk.NamePatternRegex
	if !rx.MatchString(app.Name) {
		return sdk.NewError(sdk.ErrInvalidName, fmt.Errorf("Invalid application name. It should match %s", sdk.NamePattern))
	}

	app.LastModified = time.Now()
	dbApp := dbApplication(*app)
	n, err := db.Update(&dbApp)
	if err != nil {
		return sdk.WrapError(err, "application.Update %s(%d)", app.Name, app.ID)
	}
	if n == 0 {
		return sdk.WrapError(sdk.ErrApplicationNotFound, "application.Update %s(%d)", app.Name, app.ID)
	}
	return UpdateLastModified(db, store, app, u)
}

// UpdateLastModified Update last_modified column in application table
func UpdateLastModified(db gorp.SqlExecutor, store cache.Store, app *sdk.Application, u *sdk.User) error {
	query := `
		UPDATE application SET last_modified = current_timestamp WHERE id = $1 RETURNING last_modified
	`
	var lastModified time.Time
	err := db.QueryRow(query, app.ID).Scan(&lastModified)
	if err == nil {
		app.LastModified = lastModified
	}

	if u != nil {
		store.SetWithTTL(cache.Key("lastModified", app.ProjectKey, "application", app.Name), sdk.LastModification{
			Name:         app.Name,
			Username:     u.Username,
			LastModified: lastModified.Unix(),
		}, 0)

		updates := sdk.LastModification{
			Key:          app.ProjectKey,
			Name:         app.Name,
			LastModified: lastModified.Unix(),
			Username:     u.Username,
			Type:         sdk.ApplicationLastModificationType,
		}
		b, errP := json.Marshal(updates)
		if errP == nil {
			store.Publish("lastUpdates", string(b))
		}
	}

	return sdk.WrapError(err, "application.UpdateLastModified %s(%d)", app.Name, app.ID)
}

// LoadAll returns all applications
func LoadAll(db gorp.SqlExecutor, store cache.Store, key string, u *sdk.User, opts ...LoadOptionFunc) ([]sdk.Application, error) {
	var query string
	var args []interface{}

	if u == nil || u.Admin {
		query = `
		SELECT  application.*
		FROM application
		JOIN project ON project.id = application.project_id
		WHERE project.projectkey = $1
		ORDER BY application.name ASC`
		args = []interface{}{key}
	} else {
		query = `
			SELECT distinct application.*
			FROM application
			JOIN project ON project.id = application.project_id
			WHERE application.id IN (
				SELECT application_group.application_id
				FROM application_group
				JOIN group_user ON application_group.group_id = group_user.group_id
				WHERE group_user.user_id = $2
			)
			AND project.projectkey = $1
			ORDER by application.name ASC`
		args = []interface{}{key, u.ID}
	}
	return loadapplications(db, store, u, opts, query, args...)
}

// LoadAllNames returns all application names
func LoadAllNames(db gorp.SqlExecutor, projID int64, u *sdk.User) ([]sdk.IDName, error) {
	var query string
	var args []interface{}

	if u == nil || u.Admin {
		query = `
		SELECT application.id, application.name
		FROM application
		WHERE application.project_id= $1
		ORDER BY application.name ASC`
		args = []interface{}{projID}
	} else {
		query = `
			SELECT distinct(application.id) AS id, application.name
			FROM application
			WHERE application.id IN (
				SELECT application_group.application_id
				FROM application_group
				JOIN group_user ON application_group.group_id = group_user.group_id
				WHERE group_user.user_id = $2
			)
			AND application.project_id = $1
			ORDER by application.name ASC`
		args = []interface{}{projID, u.ID}
	}

	res := []sdk.IDName{}
	if _, err := db.Select(&res, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return res, nil
		}
		return nil, sdk.WrapError(err, "application.loadapplicationnames")
	}

	return res, nil
}

func loadapplications(db gorp.SqlExecutor, store cache.Store, u *sdk.User, opts []LoadOptionFunc, query string, args ...interface{}) ([]sdk.Application, error) {
	var res []dbApplication
	if _, err := db.Select(&res, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WrapError(sdk.ErrApplicationNotFound, "application.loadapplications")
		}
		return nil, sdk.WrapError(err, "application.loadapplications")
	}

	apps := make([]sdk.Application, len(res))
	for i := range res {
		a := &res[i]
		if err := a.PostGet(db); err != nil {
			return nil, sdk.WrapError(err, "application.loadapplications")
		}
		app, err := unwrap(db, store, u, opts, a)
		if err != nil {
			return nil, sdk.WrapError(err, "application.loadapplications")
		}
		apps[i] = *app
	}

	return apps, nil
}
