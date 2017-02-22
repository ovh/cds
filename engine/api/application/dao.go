package application

import (
	"github.com/go-gorp/gorp"

	"database/sql"

	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

type loadOptionFunc *func(gorp.SqlExecutor, *sdk.Application, *sdk.User) error

// LoadOptions provides all options on project loads functions
var LoadOptions = struct {
	Default               loadOptionFunc
	WithVariables         loadOptionFunc
	WithPipelines         loadOptionFunc
	WithTriggers          loadOptionFunc
	WithGroups            loadOptionFunc
	WithPermission        loadOptionFunc //doit etre fait systématiquement
	WithPollers           loadOptionFunc
	WithSchedulers        loadOptionFunc
	WithHooks             loadOptionFunc
	WithNotifs            loadOptionFunc
	WithRepositoryManager loadOptionFunc
}{
	Default:               &loadDefaultDependencies,
	WithVariables:         &loadVariables,
	WithPipelines:         &loadPipelines,
	WithTriggers:          &loadTriggers,
	WithGroups:            &loadGroups,
	WithPermission:        &loadPermission, //doit etre fait systématiquement
	WithPollers:           &loadPollers,
	WithSchedulers:        &loadSchedulers,
	WithHooks:             &loadHooks,
	WithNotifs:            &loadNotifs,
	WithRepositoryManager: &loadRepositoryManager,
}

// LoadByName load an application from DB
func LoadByName(db gorp.SqlExecutor, projectKey, appName string, u *sdk.User, opts ...loadOptionFunc) (*sdk.Application, error) {
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

func load(db gorp.SqlExecutor, u *sdk.User, opts []loadOptionFunc, query string, args ...interface{}) (*sdk.Application, error) {
	log.Debug("application.load> %s %v", query, args)
	dbApp := dbApplication{}
	if err := db.SelectOne(&dbApp, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrApplicationNotFound
		}
		return nil, err
	}
	return unwrap(db, u, opts, &dbApp)
}

func unwrap(db gorp.SqlExecutor, u *sdk.User, opts []loadOptionFunc, dbApp *dbApplication) (*sdk.Application, error) {
	app := sdk.Application(*dbApp)
	for _, f := range opts {
		if err := (*f)(db, &app, u); err != nil && err != sql.ErrNoRows {
			return nil, err
		}
	}
	return &app, nil
}
