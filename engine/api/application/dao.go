package application

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/sdk"
)

const appRows = `
application.id,
application.name, 
application.project_id,
application.repo_fullname,
application.repositories_manager_id,
application.last_modified,
application.metadata,
application.vcs_server,
application.vcs_strategy,
application.description
`

// LoadOptionFunc is a type for all options in LoadOptions
type LoadOptionFunc *func(gorp.SqlExecutor, cache.Store, *sdk.Application) error

// LoadOptions provides all options on project loads functions
var LoadOptions = struct {
	Default                        LoadOptionFunc
	WithVariables                  LoadOptionFunc
	WithVariablesWithClearPassword LoadOptionFunc
	WithKeys                       LoadOptionFunc
	WithClearKeys                  LoadOptionFunc
	WithDeploymentStrategies       LoadOptionFunc
	WithClearDeploymentStrategies  LoadOptionFunc
	WithVulnerabilities            LoadOptionFunc
	WithIcon                       LoadOptionFunc
}{
	Default:                        &loadDefaultDependencies,
	WithVariables:                  &loadVariables,
	WithVariablesWithClearPassword: &loadVariablesWithClearPassword,
	WithKeys:                       &loadKeys,
	WithClearKeys:                  &loadClearKeys,
	WithDeploymentStrategies:       &loadDeploymentStrategies,
	WithClearDeploymentStrategies:  &loadDeploymentStrategiesWithClearPassword,
	WithVulnerabilities:            &loadVulnerabilities,
	WithIcon:                       &loadIcon,
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
func LoadByName(db gorp.SqlExecutor, store cache.Store, projectKey, appName string, opts ...LoadOptionFunc) (*sdk.Application, error) {
	query := fmt.Sprintf(`
                SELECT %s
                FROM application
                	JOIN project ON project.id = application.project_id
                WHERE project.projectkey = $1
                AND application.name = $2`, appRows)
	args := []interface{}{projectKey, appName}

	return load(db, store, projectKey, opts, query, args...)
}

// LoadAndLockByID load and lock given application
func LoadAndLockByID(db gorp.SqlExecutor, store cache.Store, id int64, opts ...LoadOptionFunc) (*sdk.Application, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM application
		WHERE application.id = $1 FOR UPDATE NOWAIT`, appRows)
	args := []interface{}{id}

	return load(db, store, "", opts, query, args...)
}

// LoadByID load an application from DB
func LoadByID(db gorp.SqlExecutor, store cache.Store, id int64, opts ...LoadOptionFunc) (*sdk.Application, error) {
	query := fmt.Sprintf(`
                SELECT %s
                FROM application
                WHERE application.id = $1`, appRows)
	args := []interface{}{id}

	return load(db, store, "", opts, query, args...)
}

// LoadByWorkflowID loads applications from database for a given workflow id
func LoadByWorkflowID(db gorp.SqlExecutor, workflowID int64) ([]sdk.Application, error) {
	apps := []sdk.Application{}
	query := fmt.Sprintf(`SELECT DISTINCT %s
	FROM application
		JOIN workflow_node_context ON workflow_node_context.application_id = application.id
		JOIN workflow_node ON workflow_node.id = workflow_node_context.workflow_node_id
		JOIN workflow ON workflow.id = workflow_node.workflow_id
	WHERE workflow.id = $1`, appRows)

	if _, err := db.Select(&apps, query, workflowID); err != nil {
		if err == sql.ErrNoRows {
			return apps, nil
		}
		return nil, sdk.WrapError(err, "Unable to load applications linked to workflow id %d", workflowID)
	}

	return apps, nil
}

func load(db gorp.SqlExecutor, store cache.Store, key string, opts []LoadOptionFunc, query string, args ...interface{}) (*sdk.Application, error) {
	dbApp := dbApplication{}
	if err := db.SelectOne(&dbApp, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WrapError(sdk.ErrApplicationNotFound, "application.load")
		}
		return nil, sdk.WrapError(err, "application.load")
	}
	dbApp.ProjectKey = key
	return unwrap(db, store, opts, &dbApp)
}

func unwrap(db gorp.SqlExecutor, store cache.Store, opts []LoadOptionFunc, dbApp *dbApplication) (*sdk.Application, error) {
	app := sdk.Application(*dbApp)

	if app.ProjectKey == "" {
		pkey, errP := db.SelectStr("SELECT projectkey FROM project WHERE id = $1", app.ProjectID)
		if errP != nil {
			return nil, sdk.WrapError(errP, "application.unwrap")
		}
		app.ProjectKey = pkey
	}

	for _, f := range opts {
		if err := (*f)(db, store, &app); err != nil && sdk.Cause(err) != sql.ErrNoRows {
			return nil, sdk.WrapError(err, "application.unwrap")
		}
	}
	return &app, nil
}

// Insert add an application id database
func Insert(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, app *sdk.Application, u *sdk.User) error {
	if err := app.IsValid(); err != nil {
		return sdk.WrapError(err, "application is not valid")
	}

	app.ProjectID = proj.ID
	app.ProjectKey = proj.Key
	app.LastModified = time.Now()

	dbApp := dbApplication(*app)
	if err := db.Insert(&dbApp); err != nil {
		if errPG, ok := err.(*pq.Error); ok && errPG.Code == gorpmapping.ViolateUniqueKeyPGCode {
			err = sdk.ErrApplicationExist
		}
		return sdk.WrapError(err, "application.Insert %s(%d)", app.Name, app.ID)
	}
	*app = sdk.Application(dbApp)
	event.PublishAddApplication(proj.Key, *app, u)

	return nil
}

// Update updates application id database
func Update(db gorp.SqlExecutor, store cache.Store, app *sdk.Application) error {
	if err := app.IsValid(); err != nil {
		return sdk.WrapError(err, "application is not valid")
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

	return nil
}

// LoadAll returns all applications
func LoadAll(db gorp.SqlExecutor, store cache.Store, key string, opts ...LoadOptionFunc) ([]sdk.Application, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM application
			JOIN project ON project.id = application.project_id
		WHERE project.projectkey = $1
		ORDER BY application.name ASC`, appRows)
	args := []interface{}{key}

	return loadapplications(db, store, opts, query, args...)
}

// LoadAllNames returns all application names
func LoadAllNames(db gorp.SqlExecutor, projID int64) ([]sdk.IDName, error) {
	query := `
		SELECT application.id, application.name, application.description, application.icon
		FROM application
		WHERE application.project_id= $1
		ORDER BY application.name ASC`

	var res []sdk.IDName
	if _, err := db.Select(&res, query, projID); err != nil {
		if err == sql.ErrNoRows {
			return res, nil
		}
		return nil, sdk.WrapError(err, "application.loadapplicationnames")
	}

	return res, nil
}

func loadapplications(db gorp.SqlExecutor, store cache.Store, opts []LoadOptionFunc, query string, args ...interface{}) ([]sdk.Application, error) {
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
		app, err := unwrap(db, store, opts, a)
		if err != nil {
			return nil, sdk.WrapError(err, "application.loadapplications")
		}
		apps[i] = *app
	}

	return apps, nil
}

// LoadIcon return application icon given his application id
func LoadIcon(db gorp.SqlExecutor, appID int64) (string, error) {
	icon, err := db.SelectStr("SELECT icon FROM application WHERE id = $1", appID)

	return icon, sdk.WithStack(err)
}
