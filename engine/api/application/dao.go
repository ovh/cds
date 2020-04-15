package application

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type dbApplication struct {
	gorpmapping.SignedEntity
	sdk.Application
}

func (e dbApplication) Canonical() gorpmapping.CanonicalForms {
	var _ = []interface{}{e.ProjectID, e.Name}
	return gorpmapping.CanonicalForms{
		"{{print .ProjectID}}{{.Name}}",
	}
}

// LoadOptionFunc is a type for all options in LoadOptions
type LoadOptionFunc *func(gorp.SqlExecutor, *sdk.Application) error

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

// LoadAsCode load an ascode application from DB
func LoadAsCode(db gorp.SqlExecutor, projectKey, fromRepo string) ([]sdk.Application, error) {
	query := gorpmapping.NewQuery(`
		SELECT application.*
		FROM application
		JOIN project ON project.id = application.project_id
		WHERE project.projectkey = $1
		AND application.from_repository = $2`).Args(projectKey, fromRepo)
	return loadAll(context.Background(), db, nil, query)
}

// LoadByNameWithClearVCSStrategyPassword load an application from DB
func LoadByNameWithClearVCSStrategyPassword(db gorp.SqlExecutor, projectKey, appName string, opts ...LoadOptionFunc) (*sdk.Application, error) {
	query := gorpmapping.NewQuery(`
		SELECT application.*
		FROM application
		JOIN project ON project.id = application.project_id
		WHERE project.projectkey = $1
		AND application.name = $2`).Args(projectKey, appName)
	app, err := load(context.Background(), db, projectKey, opts, query)
	if err != nil {
		return nil, err
	}
	return app, nil
}

// LoadByName load an application from DB
func LoadByName(db gorp.SqlExecutor, projectKey, appName string, opts ...LoadOptionFunc) (*sdk.Application, error) {
	query := gorpmapping.NewQuery(`
		SELECT application.*
		FROM application
		JOIN project ON project.id = application.project_id
		WHERE project.projectkey = $1
		AND application.name = $2`).Args(projectKey, appName)
	app, err := load(context.Background(), db, projectKey, opts, query)
	if err != nil {
		return nil, err
	}
	app.RepositoryStrategy.Password = sdk.PasswordPlaceholder
	app.RepositoryStrategy.SSHKeyContent = ""
	return app, nil
}

func LoadByIDWithClearVCSStrategyPassword(db gorp.SqlExecutor, id int64, opts ...LoadOptionFunc) (*sdk.Application, error) {
	query := gorpmapping.NewQuery(`
                SELECT application.*
                FROM application
                WHERE application.id = $1`).Args(id)
	app, err := load(context.Background(), db, "", opts, query)
	if err != nil {
		return nil, err
	}
	return app, nil
}

// LoadByID load an application from DB
func LoadByID(db gorp.SqlExecutor, id int64, opts ...LoadOptionFunc) (*sdk.Application, error) {
	query := gorpmapping.NewQuery(`
                SELECT application.*
                FROM application
                WHERE application.id = $1`).Args(id)
	app, err := load(context.Background(), db, "", opts, query)
	if err != nil {
		return nil, err
	}
	app.RepositoryStrategy.Password = sdk.PasswordPlaceholder
	app.RepositoryStrategy.SSHKeyContent = ""
	return app, nil
}

// LoadByWorkflowID loads applications from database for a given workflow id
func LoadByWorkflowID(db gorp.SqlExecutor, workflowID int64) ([]sdk.Application, error) {
	query := gorpmapping.NewQuery(`
	SELECT DISTINCT application.*
	FROM application
	JOIN w_node_context ON w_node_context.application_id = application.id
	JOIN w_node ON w_node.id = w_node_context.node_id
	JOIN workflow ON workflow.id = w_node.workflow_id
	WHERE workflow.id = $1`).Args(workflowID)
	return loadAll(context.Background(), db, nil, query)
}

func load(ctx context.Context, db gorp.SqlExecutor, key string, opts []LoadOptionFunc, query gorpmapping.Query) (*sdk.Application, error) {
	dbApp := dbApplication{}
	// Allways load with decryption to get all the data for vcs_strategy
	found, err := gorpmapping.Get(ctx, db, query, &dbApp, gorpmapping.GetOptions.WithDecryption)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	isValid, err := gorpmapping.CheckSignature(dbApp, dbApp.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(context.Background(), "application.load> application %d data corrupted", dbApp.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	dbApp.ProjectKey = key
	return unwrap(db, opts, &dbApp)
}

func unwrap(db gorp.SqlExecutor, opts []LoadOptionFunc, dbApp *dbApplication) (*sdk.Application, error) {
	app := &dbApp.Application

	if app.ProjectKey == "" {
		pkey, errP := db.SelectStr("SELECT projectkey FROM project WHERE id = $1", app.ProjectID)
		if errP != nil {
			return nil, sdk.WrapError(errP, "application.unwrap")
		}
		app.ProjectKey = pkey
	}

	for _, f := range opts {
		if err := (*f)(db, app); err != nil && sdk.Cause(err) != sql.ErrNoRows {
			return nil, sdk.WrapError(err, "application.unwrap")
		}
	}
	return app, nil
}

// Insert add an application id database
func Insert(db gorp.SqlExecutor, proj sdk.Project, app *sdk.Application) error {
	if err := app.IsValid(); err != nil {
		return sdk.WrapError(err, "application is not valid")
	}

	app.ProjectID = proj.ID
	app.ProjectKey = proj.Key
	app.LastModified = time.Now()
	copyVCSStrategy := app.RepositoryStrategy

	dbApp := dbApplication{Application: *app}
	if err := gorpmapping.InsertAndSign(context.Background(), db, &dbApp); err != nil {
		return sdk.WrapError(err, "application.Insert %s(%d)", app.Name, app.ID)
	}
	*app = dbApp.Application
	// Reset the vcs_stragegy except the passowrd because it as been erased by the encryption layed
	app.RepositoryStrategy = copyVCSStrategy
	app.RepositoryStrategy.Password = sdk.PasswordPlaceholder
	app.RepositoryStrategy.SSHKeyContent = ""

	return nil
}

// UpdateColumns is only available for migration, it should be removed in a further release
func UpdateColumns(db gorp.SqlExecutor, app *sdk.Application, columnFilter gorp.ColumnFilter) error {
	app.LastModified = time.Now()
	dbApp := dbApplication{Application: *app}
	if err := gorpmapping.UpdateColumnsAndSign(context.Background(), db, &dbApp, columnFilter); err != nil {
		return sdk.WrapError(err, "application.Update %s(%d)", app.Name, app.ID)
	}
	app.RepositoryStrategy.Password = sdk.PasswordPlaceholder
	app.RepositoryStrategy.SSHKeyContent = ""
	return nil
}

// Update updates application id database
func Update(db gorp.SqlExecutor, app *sdk.Application) error {

	if app.RepositoryStrategy.Password == sdk.PasswordPlaceholder {
		appTmp, err := LoadByIDWithClearVCSStrategyPassword(db, app.ID)
		if err != nil {
			return err
		}
		app.RepositoryStrategy.Password = appTmp.RepositoryStrategy.Password
	}
	if app.RepositoryStrategy.ConnectionType == "ssh" {
		app.RepositoryStrategy.Password = ""
	}

	var copyVCSStrategy = app.RepositoryStrategy

	if err := app.IsValid(); err != nil {
		return sdk.WrapError(err, "application is not valid")
	}
	app.LastModified = time.Now()
	dbApp := dbApplication{Application: *app}
	if err := gorpmapping.UpdateAndSign(context.Background(), db, &dbApp); err != nil {
		return sdk.WrapError(err, "application.Update %s(%d)", app.Name, app.ID)
	}
	// Reset the vcs_stragegy except the passowrd because it as been erased by the encryption layed
	app.RepositoryStrategy = copyVCSStrategy
	app.RepositoryStrategy.Password = sdk.PasswordPlaceholder
	app.RepositoryStrategy.SSHKeyContent = ""
	return nil
}

// LoadAll returns all applications
func LoadAll(db gorp.SqlExecutor, key string, opts ...LoadOptionFunc) ([]sdk.Application, error) {
	query := gorpmapping.NewQuery(`
	SELECT application.*
	FROM application
	JOIN project ON project.id = application.project_id
	WHERE project.projectkey = $1
	ORDER BY application.name ASC`).Args(key)

	return loadAll(context.Background(), db, opts, query)
}

// LoadAllNames returns all application names
func LoadAllNames(db gorp.SqlExecutor, projID int64) (sdk.IDNames, error) {
	query := `
		SELECT application.id, application.name, application.description, application.icon
		FROM application
		WHERE application.project_id= $1
		ORDER BY application.name ASC`

	var res sdk.IDNames
	if _, err := db.Select(&res, query, projID); err != nil {
		if err == sql.ErrNoRows {
			return res, nil
		}
		return nil, sdk.WrapError(err, "application.loadapplicationnames")
	}

	return res, nil
}

func loadAll(ctx context.Context, db gorp.SqlExecutor, opts []LoadOptionFunc, query gorpmapping.Query) ([]sdk.Application, error) {
	var res []dbApplication
	if err := gorpmapping.GetAll(ctx, db, query, &res, gorpmapping.GetOptions.WithDecryption); err != nil {
		return nil, err
	}

	apps := make([]sdk.Application, len(res))
	for i := range res {
		isValid, err := gorpmapping.CheckSignature(res[i], res[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "application.loadApplications> application %d data corrupted", res[i].ID)
			continue
		}

		a := &res[i]
		app, err := unwrap(db, opts, a)
		if err != nil {
			return nil, sdk.WrapError(err, "application.loadapplications")
		}

		//By default all vcds_strategy password are masked
		app.RepositoryStrategy.Password = sdk.PasswordPlaceholder
		apps[i] = *app
	}

	return apps, nil
}

// LoadIcon return application icon given his application id
func LoadIcon(db gorp.SqlExecutor, appID int64) (string, error) {
	icon, err := db.SelectStr("SELECT icon FROM application WHERE id = $1", appID)
	return icon, sdk.WithStack(err)
}
