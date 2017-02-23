package application

import (
	"github.com/go-gorp/gorp"

	"database/sql"

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
            SELECT application.* 
            FROM application 
            JOIN project ON project.id = application.project_id
            JOIN application_group on application.id = application_group.application_id
            JOIN group_user on application_group.group_id = application_group.group_id
            WHERE project.projectkey = $1
            AND application.name = $2
            AND group_user.user_id = $3`
		args = []interface{}{projectKey, appName, u.ID}
	}

	return load(db, u, opts, query, args...)
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
            SELECT application.* 
            FROM application 
            JOIN application_group on application.id = application_group.application_id
            JOIN group_user on application_group.group_id = application_group.group_id
            AND application.id = $1
            AND group_user.user_id = $2`
		args = []interface{}{id, u.ID}
	}

	return load(db, u, opts, query, args...)
}

func load(db gorp.SqlExecutor, u *sdk.User, opts []LoadOptionFunc, query string, args ...interface{}) (*sdk.Application, error) {
	log.Debug("application.load> %s %v", query, args)
	dbApp := dbApplication{}
	if err := db.SelectOne(&dbApp, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrApplicationNotFound
		}
		return nil, sdk.WrapError(err, "application.load")
	}
	return unwrap(db, u, opts, &dbApp)
}

func unwrap(db gorp.SqlExecutor, u *sdk.User, opts []LoadOptionFunc, dbApp *dbApplication) (*sdk.Application, error) {
	app := sdk.Application(*dbApp)

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
