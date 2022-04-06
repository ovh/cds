package application

import (
	"context"
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

var (
	loadDefaultDependencies = func(ctx context.Context, db gorp.SqlExecutor, app *sdk.Application) error {
		if err := loadVariables(ctx, db, app); err != nil && sdk.Cause(err) != sql.ErrNoRows {
			return sdk.WrapError(err, "application.loadDefaultDependencies %s", app.Name)
		}
		return nil
	}

	loadVariables = func(ctx context.Context, db gorp.SqlExecutor, app *sdk.Application) error {
		variables, err := LoadAllVariables(db, app.ID)
		if err != nil && sdk.Cause(err) != sql.ErrNoRows {
			return sdk.WrapError(err, "Unable to load variables for application %d", app.ID)
		}
		app.Variables = variables
		return nil
	}

	loadVariablesWithClearPassword = func(ctx context.Context, db gorp.SqlExecutor, app *sdk.Application) error {
		variables, err := LoadAllVariablesWithDecrytion(db, app.ID)
		if err != nil && sdk.Cause(err) != sql.ErrNoRows {
			return sdk.WrapError(err, "Unable to load variables for application %d", app.ID)
		}
		app.Variables = variables
		return nil
	}

	loadKeys = func(ctx context.Context, db gorp.SqlExecutor, app *sdk.Application) error {
		keys, err := LoadAllKeys(db, app.ID)
		if err != nil {
			return err
		}
		app.Keys = keys
		return nil
	}

	loadClearKeys = func(ctx context.Context, db gorp.SqlExecutor, app *sdk.Application) error {
		keys, err := LoadAllKeysWithPrivateContent(db, app.ID)
		if err != nil {
			return err
		}
		app.Keys = keys
		return nil
	}

	loadDeploymentStrategies = func(ctx context.Context, db gorp.SqlExecutor, app *sdk.Application) error {
		var err error
		app.DeploymentStrategies, err = LoadDeploymentStrategies(ctx, db, app.ID, false)
		if err != nil && sdk.Cause(err) != sql.ErrNoRows {
			return sdk.WrapError(err, "Unable to load deployment strategies for application %d", app.ID)
		}
		return nil
	}

	loadIcon = func(ctx context.Context, db gorp.SqlExecutor, app *sdk.Application) error {
		var err error
		app.Icon, err = LoadIcon(db, app.ID)
		if err != nil && sdk.Cause(err) != sql.ErrNoRows {
			return sdk.WrapError(err, "Unable to load icon")
		}
		return nil
	}

	loadVulnerabilities = func(ctx context.Context, db gorp.SqlExecutor, app *sdk.Application) error {
		var err error
		app.Vulnerabilities, err = LoadVulnerabilities(db, app.ID)
		if err != nil {
			return sdk.WrapError(err, "unable to load vulnerabilities")
		}
		return nil
	}

	loadDeploymentStrategiesWithClearPassword = func(ctx context.Context, db gorp.SqlExecutor, app *sdk.Application) error {
		var err error
		app.DeploymentStrategies, err = LoadDeploymentStrategies(ctx, db, app.ID, true)
		if err != nil && sdk.Cause(err) != sql.ErrNoRows {
			return sdk.WrapError(err, "Unable to load deployment strategies for application %d", app.ID)
		}
		return nil
	}
)
