package application

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// LoadOptionFunc is a type for all options in LoadOptions
type LoadOptionFunc *func(gorp.SqlExecutor, *sdk.Application, *sdk.User) error

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
	WithRepositoryManager          LoadOptionFunc
}{
	Default:                        &loadDefaultDependencies,
	WithVariables:                  &loadVariables,
	WithVariablesWithClearPassword: &loadVariablesWithClearPassword,
	WithPipelines:                  &loadPipelines,
	WithTriggers:                   &loadTriggers,
	WithGroups:                     &loadGroups,
	WithHooks:                      &loadHooks,
	WithNotifs:                     &loadNotifs,
	WithRepositoryManager:          &loadRepositoryManager,
}

// LoadByName load an application from DB
func LoadByName(db gorp.SqlExecutor, projectKey, appName string, u *sdk.User, opts ...LoadOptionFunc) (*sdk.Application, error) {
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

	return load(db, projectKey, u, opts, query, args...)
}

// LoadByID load an application from DB
func LoadByID(db gorp.SqlExecutor, id int64, u *sdk.User, opts ...LoadOptionFunc) (*sdk.Application, error) {
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

	return load(db, "", u, opts, query, args...)
}

// LoadByPipeline Load application where pipeline is attached
func LoadByPipeline(db gorp.SqlExecutor, pipelineID int64, u *sdk.User, opts ...LoadOptionFunc) ([]sdk.Application, error) {
	query := `SELECT distinct application.*
		 FROM application
		 JOIN application_pipeline ON application.id = application_pipeline.application_id
		 WHERE application_pipeline.pipeline_id = $1
		 ORDER BY application.name`
	app, err := loadapplications(db, u, opts, query, pipelineID)
	if err != nil {
		return nil, sdk.WrapError(err, "LoadByPipeline (%d)", pipelineID)
	}
	return app, nil
}

func load(db gorp.SqlExecutor, key string, u *sdk.User, opts []LoadOptionFunc, query string, args ...interface{}) (*sdk.Application, error) {
	log.Debug("application.load> %s %v", query, args)
	dbApp := dbApplication{}
	if err := db.SelectOne(&dbApp, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WrapError(sdk.ErrApplicationNotFound, "application.load")
		}
		return nil, sdk.WrapError(err, "application.load")
	}
	dbApp.ProjectKey = key
	return unwrap(db, u, opts, &dbApp)
}

func unwrap(db gorp.SqlExecutor, u *sdk.User, opts []LoadOptionFunc, dbApp *dbApplication) (*sdk.Application, error) {
	app := sdk.Application(*dbApp)

	if app.ProjectKey == "" {
		pkey, errP := db.SelectStr("select projectkey from project where id = $1", app.ProjectID)
		if errP != nil {
			return nil, sdk.WrapError(errP, "application.unwrap")
		}
		app.ProjectKey = pkey
	}

	if u != nil {
		loadPermission(db, &app, u)
	}

	for _, f := range opts {
		if err := (*f)(db, &app, u); err != nil && err != sql.ErrNoRows {
			return nil, sdk.WrapError(err, "application.unwrap")
		}
	}
	return &app, nil
}

// Insert add an application id database
func Insert(db gorp.SqlExecutor, proj *sdk.Project, app *sdk.Application) error {
	app.ProjectID = proj.ID
	app.ProjectKey = proj.Key
	app.LastModified = time.Now()
	dbApp := dbApplication(*app)
	if err := db.Insert(&dbApp); err != nil {
		return sdk.WrapError(err, "application.Insert %s(%d)", app.Name, app.ID)
	}
	*app = sdk.Application(dbApp)
	return nil
}

// Update updates application id database
func Update(db gorp.SqlExecutor, app *sdk.Application) error {
	app.LastModified = time.Now()
	dbApp := dbApplication(*app)
	n, err := db.Update(&dbApp)
	if err != nil {
		return sdk.WrapError(err, "application.Update %s(%d)", app.Name, app.ID)
	}
	if n == 0 {
		return sdk.WrapError(sdk.ErrApplicationNotFound, "application.Update %s(%d)", app.Name, app.ID)
	}
	return nil
}

// UpdateLastModified Update last_modified column in application table
func UpdateLastModified(db gorp.SqlExecutor, app *sdk.Application, u *sdk.User) error {
	query := `
		UPDATE application SET last_modified=current_timestamp WHERE id = $1 RETURNING last_modified
	`
	var lastModified time.Time
	err := db.QueryRow(query, app.ID).Scan(&lastModified)
	if err == nil {
		app.LastModified = lastModified
	}
	return sdk.WrapError(err, "application.UpdateLastModified %s(%d)", app.Name, app.ID)
}

// LoadAll returns all applications
func LoadAll(db gorp.SqlExecutor, key string, u *sdk.User, opts ...LoadOptionFunc) ([]sdk.Application, error) {
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
	return loadapplications(db, u, opts, query, args...)
}

func loadapplications(db gorp.SqlExecutor, u *sdk.User, opts []LoadOptionFunc, query string, args ...interface{}) ([]sdk.Application, error) {
	log.Debug("application.loadapplications> %s %v", query, args)

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
		app, err := unwrap(db, u, opts, a)
		if err != nil {
			return nil, sdk.WrapError(err, "application.loadapplications")
		}
		apps[i] = *app
	}

	return apps, nil
}
