package application

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/sdk"
)

var (
	loadDefaultDependencies = func(db gorp.SqlExecutor, store cache.Store, app *sdk.Application, u *sdk.User) error {
		if err := loadVariables(db, store, app, u); err != nil && sdk.Cause(err) != sql.ErrNoRows {
			return sdk.WrapError(err, "application.loadDefaultDependencies %s", app.Name)
		}
		return nil
	}

	loadVariables = func(db gorp.SqlExecutor, store cache.Store, app *sdk.Application, u *sdk.User) error {
		variables, err := GetAllVariableByID(db, app.ID)
		if err != nil && sdk.Cause(err) != sql.ErrNoRows {
			return sdk.WrapError(err, "Unable to load variables for application %d", app.ID)
		}
		app.Variable = variables
		return nil
	}

	loadVariablesWithClearPassword = func(db gorp.SqlExecutor, store cache.Store, app *sdk.Application, u *sdk.User) error {
		variables, err := GetAllVariableByID(db, app.ID, WithClearPassword())
		if err != nil && sdk.Cause(err) != sql.ErrNoRows {
			return sdk.WrapError(err, "Unable to load variables for application %d", app.ID)
		}
		app.Variable = variables
		return nil
	}

	loadKeys = func(db gorp.SqlExecutor, store cache.Store, app *sdk.Application, u *sdk.User) error {
		return LoadAllKeys(db, app)
	}

	loadClearKeys = func(db gorp.SqlExecutor, store cache.Store, app *sdk.Application, u *sdk.User) error {
		return LoadAllDecryptedKeys(db, app)
	}

	loadGroups = func(db gorp.SqlExecutor, store cache.Store, app *sdk.Application, u *sdk.User) error {
		if err := LoadGroupByApplication(db, app); err != nil && sdk.Cause(err) != sql.ErrNoRows {
			return sdk.WrapError(err, "Unable to load group permission for application %d", app.ID)
		}
		return nil
	}

	//LoadPermission loads the permission on an application
	LoadPermission = func(db gorp.SqlExecutor, store cache.Store, app *sdk.Application, u *sdk.User) error {
		app.Permission = permission.ApplicationPermission(app.ProjectKey, app.Name, u)
		return nil
	}

	loadDeploymentStrategies = func(db gorp.SqlExecutor, store cache.Store, app *sdk.Application, u *sdk.User) error {
		var err error
		app.DeploymentStrategies, err = LoadDeploymentStrategies(db, app.ID, false)
		if err != nil && sdk.Cause(err) != sql.ErrNoRows {
			return sdk.WrapError(err, "Unable to load deployment strategies for application %d", app.ID)
		}
		return nil
	}

	loadIcon = func(db gorp.SqlExecutor, store cache.Store, app *sdk.Application, u *sdk.User) error {
		var err error
		app.Icon, err = LoadIcon(db, app.ID)
		if err != nil && sdk.Cause(err) != sql.ErrNoRows {
			return sdk.WrapError(err, "Unable to load icon")
		}
		return nil
	}

	loadVulnerabilities = func(db gorp.SqlExecutor, store cache.Store, app *sdk.Application, u *sdk.User) error {
		var err error
		app.Vulnerabilities, err = LoadVulnerabilities(db, app.ID)
		if err != nil && sdk.Cause(err) != sql.ErrNoRows {
			return sdk.WrapError(err, "Unable to load vulnerabilities")
		}
		return nil
	}

	loadDeploymentStrategiesWithClearPassword = func(db gorp.SqlExecutor, store cache.Store, app *sdk.Application, u *sdk.User) error {
		var err error
		app.DeploymentStrategies, err = LoadDeploymentStrategies(db, app.ID, true)
		if err != nil && sdk.Cause(err) != sql.ErrNoRows {
			return sdk.WrapError(err, "Unable to load deployment strategies for application %d", app.ID)
		}
		return nil
	}
)
