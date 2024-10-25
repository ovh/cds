package application

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

type dbApplication struct {
	gorpmapper.SignedEntity
	sdk.Application
}

func (e dbApplication) Canonical() gorpmapper.CanonicalForms {
	var _ = []interface{}{e.ProjectID, e.Name}
	return gorpmapper.CanonicalForms{
		"{{printf .ProjectID}}{{.Name}}",
		"{{print .ProjectID}}{{.Name}}",
	}
}

// LoadOptionFunc is a type for all options in LoadOptions
type LoadOptionFunc func(context.Context, gorp.SqlExecutor, *sdk.Application) error

// LoadOptions provides all options on project loads functions
var LoadOptions = struct {
	Default                        LoadOptionFunc
	WithVariables                  LoadOptionFunc
	WithVariablesWithClearPassword LoadOptionFunc
	WithKeys                       LoadOptionFunc
	WithClearKeys                  LoadOptionFunc
	WithDeploymentStrategies       LoadOptionFunc
	WithClearDeploymentStrategies  LoadOptionFunc
	WithIcon                       LoadOptionFunc
}{
	Default:                        loadDefaultDependencies,
	WithVariables:                  loadVariables,
	WithVariablesWithClearPassword: loadVariablesWithClearPassword,
	WithKeys:                       loadKeys,
	WithClearKeys:                  loadClearKeys,
	WithDeploymentStrategies:       loadDeploymentStrategies,
	WithClearDeploymentStrategies:  loadDeploymentStrategiesWithClearPassword,
	WithIcon:                       loadIcon,
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
func LoadByName(ctx context.Context, db gorp.SqlExecutor, projectKey, appName string, opts ...LoadOptionFunc) (*sdk.Application, error) {
	query := gorpmapping.NewQuery(`
		SELECT application.*
		FROM application
		JOIN project ON project.id = application.project_id
		WHERE project.projectkey = $1
		AND application.name = $2`).Args(projectKey, appName)
	return get(ctx, db, projectKey, query, opts...)
}

// LoadByNameWithClearVCSStrategyPassword load an application from DB
func LoadByNameWithClearVCSStrategyPassword(ctx context.Context, db gorp.SqlExecutor, projectKey, appName string, opts ...LoadOptionFunc) (*sdk.Application, error) {
	query := gorpmapping.NewQuery(`
		SELECT application.*
		FROM application
		JOIN project ON project.id = application.project_id
		WHERE project.projectkey = $1
		AND application.name = $2`).Args(projectKey, appName)
	return getWithClearVCSStrategyPassword(ctx, db, projectKey, query, opts...)
}

func LoadByIDWithClearVCSStrategyPassword(ctx context.Context, db gorp.SqlExecutor, id int64, opts ...LoadOptionFunc) (*sdk.Application, error) {
	query := gorpmapping.NewQuery(`
                SELECT application.*
                FROM application
                WHERE application.id = $1`).Args(id)
	return getWithClearVCSStrategyPassword(ctx, db, "", query, opts...)
}

// LoadByID load an application from DB
func LoadByID(ctx context.Context, db gorp.SqlExecutor, id int64, opts ...LoadOptionFunc) (*sdk.Application, error) {
	query := gorpmapping.NewQuery(`
                SELECT application.*
                FROM application
                WHERE application.id = $1`).Args(id)
	return get(ctx, db, "", query, opts...)
}

// LoadByWorkflowID loads applications from database for a given workflow id
func LoadByWorkflowID(ctx context.Context, db gorp.SqlExecutor, workflowID int64) ([]sdk.Application, error) {
	query := gorpmapping.NewQuery(`
	SELECT DISTINCT application.*
	FROM application
	JOIN w_node_context ON w_node_context.application_id = application.id
	JOIN w_node ON w_node.id = w_node_context.node_id
	JOIN workflow ON workflow.id = w_node.workflow_id
	WHERE workflow.id = $1`).Args(workflowID)
	return getAll(ctx, db, query)
}

func get(ctx context.Context, db gorp.SqlExecutor, key string, query gorpmapping.Query, opts ...LoadOptionFunc) (*sdk.Application, error) {
	app, err := getWithClearVCSStrategyPassword(ctx, db, key, query, opts...)
	if err != nil {
		return nil, err
	}
	app.RepositoryStrategy.Password = sdk.PasswordPlaceholder
	app.RepositoryStrategy.SSHKeyContent = ""
	return app, nil
}

func getWithClearVCSStrategyPassword(ctx context.Context, db gorp.SqlExecutor, key string, query gorpmapping.Query, opts ...LoadOptionFunc) (*sdk.Application, error) {
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
		log.Error(ctx, "application.get> application %d data corrupted", dbApp.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	dbApp.ProjectKey = key
	return unwrap(ctx, db, &dbApp, opts...)
}

func unwrap(ctx context.Context, db gorp.SqlExecutor, dbApp *dbApplication, opts ...LoadOptionFunc) (*sdk.Application, error) {
	app := &dbApp.Application
	if app.ProjectKey == "" {
		pkey, errP := db.SelectStr("SELECT projectkey FROM project WHERE id = $1", app.ProjectID)
		if errP != nil {
			return nil, sdk.WrapError(errP, "application.unwrap")
		}
		app.ProjectKey = pkey
	}

	for _, f := range opts {
		if err := f(ctx, db, app); err != nil {
			return nil, err
		}
	}
	return app, nil
}

// Insert add an application id database
func Insert(db gorpmapper.SqlExecutorWithTx, proj sdk.Project, app *sdk.Application) error {
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

// Update updates application id database
func Update(ctx context.Context, db gorpmapper.SqlExecutorWithTx, app *sdk.Application) error {
	if app.RepositoryStrategy.Password == sdk.PasswordPlaceholder {
		appTmp, err := LoadByIDWithClearVCSStrategyPassword(ctx, db, app.ID)
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
func LoadAll(ctx context.Context, db gorp.SqlExecutor, key string, opts ...LoadOptionFunc) ([]sdk.Application, error) {
	query := gorpmapping.NewQuery(`
	SELECT application.*
	FROM application
	JOIN project ON project.id = application.project_id
	WHERE project.projectkey = $1
	ORDER BY application.name ASC`).Args(key)

	return getAll(ctx, db, query, opts...)
}

// LoadAllByIDsWithDecryption returns all applications with clear vcs strategy
func LoadAllByIDsWithDecryption(ctx context.Context, db gorp.SqlExecutor, ids []int64, opts ...LoadOptionFunc) ([]sdk.Application, error) {
	query := gorpmapping.NewQuery(`
	SELECT application.*
	FROM application
	WHERE application.id = ANY($1)`).Args(pq.Int64Array(ids))
	return getAllWithClearVCS(ctx, db, query, opts...)
}

// LoadAllByIDs returns all applications
func LoadAllByIDs(ctx context.Context, db gorp.SqlExecutor, ids []int64, opts ...LoadOptionFunc) ([]sdk.Application, error) {
	query := gorpmapping.NewQuery(`
	SELECT application.*
	FROM application
	WHERE application.id = ANY($1)
	ORDER BY application.name ASC`).Args(pq.Int64Array(ids))
	return getAll(ctx, db, query, opts...)
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

func getAllWithClearVCS(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query, opts ...LoadOptionFunc) ([]sdk.Application, error) {
	var res []dbApplication
	if err := gorpmapping.GetAll(ctx, db, query, &res, gorpmapping.GetOptions.WithDecryption); err != nil {
		return nil, err
	}

	apps := make([]sdk.Application, 0, len(res))
	for i := range res {
		isValid, err := gorpmapping.CheckSignature(res[i], res[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "application.getAllWithClearVCS> application %d data corrupted", res[i].ID)
			continue
		}
		a := &res[i]
		app, err := unwrap(ctx, db, a, opts...)
		if err != nil {
			return nil, sdk.WrapError(err, "application.getAllWithClearVCS")
		}
		apps = append(apps, *app)
	}
	return apps, nil
}

func getAll(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query, opts ...LoadOptionFunc) ([]sdk.Application, error) {
	var res []dbApplication
	if err := gorpmapping.GetAll(ctx, db, query, &res, gorpmapping.GetOptions.WithDecryption); err != nil {
		return nil, err
	}

	apps := make([]sdk.Application, 0, len(res))
	for i := range res {
		isValid, err := gorpmapping.CheckSignature(res[i], res[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "application.getAll> application %d data corrupted", res[i].ID)
			continue
		}

		a := &res[i]
		app, err := unwrap(ctx, db, a, opts...)
		if err != nil {
			return nil, sdk.WrapError(err, "application.getAll")
		}

		app.RepositoryStrategy.Password = sdk.PasswordPlaceholder
		apps = append(apps, *app)
	}

	return apps, nil
}

// LoadIcon return application icon given his application id
func LoadIcon(db gorp.SqlExecutor, appID int64) (string, error) {
	icon, err := db.SelectStr("SELECT icon FROM application WHERE id = $1", appID)
	return icon, sdk.WithStack(err)
}

// LoadAllNamesByFromRepository returns all application names for a repository
func LoadAllNamesByFromRepository(db gorp.SqlExecutor, projID int64, fromRepository string) (sdk.IDNames, error) {
	if fromRepository == "" {
		return nil, sdk.WithData(sdk.ErrUnknownError, "could not call LoadAllNamesByFromRepository with empty fromRepository")
	}
	query := `SELECT application.id, application.name
			  FROM application
			  WHERE project_id = $1 AND from_repository = $2
			  ORDER BY application.name`

	var res sdk.IDNames
	if _, err := db.Select(&res, query, projID, fromRepository); err != nil {
		if err == sql.ErrNoRows {
			return res, nil
		}
		return nil, sdk.WrapError(err, "application.LoadAllNamesByFromRepository")
	}

	return res, nil
}

// ResetFromRepository reset fromRepository for all applications using the same fromRepository in a given project
func ResetFromRepository(db gorp.SqlExecutor, projID int64, fromRepository string) error {
	if fromRepository == "" {
		return sdk.WithData(sdk.ErrUnknownError, "could not call LoadAllNamesByFromRepository with empty fromRepository")
	}
	query := `UPDATE application SET from_repository='' WHERE project_id = $1 AND from_repository = $2`
	_, err := db.Exec(query, projID, fromRepository)
	return sdk.WithStack(err)
}
