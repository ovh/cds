package migrate

import (
	"context"
	"database/sql"
	"errors"
	"reflect"

	"github.com/ovh/cds/engine/api/integration"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// RefactorIntegrationModelCrypto .
func RefactorIntegrationModelCrypto(ctx context.Context, db *gorp.DbMap) error {
	query := "SELECT id FROM integration_model WHERE sig IS NULL"
	rows, err := db.Query(query)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return sdk.WithStack(err)
	}

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close() // nolint
			return sdk.WithStack(err)
		}
		ids = append(ids, id)
	}

	if err := rows.Close(); err != nil {
		return sdk.WithStack(err)
	}

	var mError = new(sdk.MultiError)
	for _, id := range ids {
		if err := refactorIntegrationModelCrypto(ctx, db, id); err != nil {
			mError.Append(err)
			log.Error(ctx, "migrate.RefactorIntegrationModelCrypto> unable to migrate integration_model %d: %v", id, err)
		}
	}

	if mError.IsEmpty() {
		return nil
	}
	return mError
}

func refactorIntegrationModelCrypto(ctx context.Context, db *gorp.DbMap, id int64) error {
	log.Info(ctx, "migrate.refactorIntegrationModelCrypto> integration_model %d migration begin", id)

	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}

	defer tx.Rollback() // nolint

	query := `
	SELECT id, name, author, identifier, icon ,default_config, disabled, hook, storage, deployment, compute, deployment_default_config, public, public_configurations, event
	FROM integration_model 
	WHERE id = $1 
	AND sig IS NULL 
	FOR UPDATE SKIP LOCKED`

	var integrationModel sdk.IntegrationModel

	if err := tx.QueryRow(query, id).Scan(&integrationModel.ID,
		&integrationModel.Name,
		&integrationModel.Author,
		&integrationModel.Identifier,
		&integrationModel.Icon,
		&integrationModel.DefaultConfig,
		&integrationModel.Disabled,
		&integrationModel.Hook,
		&integrationModel.Storage,
		&integrationModel.Deployment,
		&integrationModel.Compute,
		&integrationModel.DeploymentDefaultConfig,
		&integrationModel.Public,
		&integrationModel.PublicConfigurations,
		&integrationModel.Event,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return sdk.WrapError(err, "unable to select and lock application %d", id)
	}

	for pfName, pfCfg := range integrationModel.PublicConfigurations {
		newCfg := pfCfg.Clone()
		if err := newCfg.DecryptSecrets(secret.DecryptValue); err != nil {
			return sdk.WrapError(err, "unable to encrypt config PublicConfigurations")
		}
		integrationModel.PublicConfigurations[pfName] = newCfg
	}

	oldPublicConfigurations := integrationModel.PublicConfigurations.Clone()

	if err := integration.UpdateModel(tx, &integrationModel); err != nil {
		return sdk.WrapError(err, "unable to update integration_model %d", id)
	}

	newIntegrationModel, err := integration.LoadModelWithClearPassword(tx, id)
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(oldPublicConfigurations, newIntegrationModel.PublicConfigurations) {
		return sdk.WrapError(errors.New("verification error"), "integration_model %d migration failure", id)
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	log.Info(ctx, "migrate.refactorIntegrationModelCrypto> integration_model %d migration end", id)
	return nil
}

func RefactorProjectIntegrationCrypto(ctx context.Context, db *gorp.DbMap) error {
	query := "SELECT id FROM project_integration WHERE sig IS NULL"
	rows, err := db.Query(query)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return sdk.WithStack(err)
	}

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close() // nolint
			return sdk.WithStack(err)
		}
		ids = append(ids, id)
	}

	if err := rows.Close(); err != nil {
		return sdk.WithStack(err)
	}

	var mError = new(sdk.MultiError)
	for _, id := range ids {
		if err := refactorProjectIntegrationCrypto(ctx, db, id); err != nil {
			mError.Append(err)
			log.Error(ctx, "migrate.RefactorProjectIntegrationCrypto> unable to migrate integration_model %d: %v", id, err)
		}
	}

	if mError.IsEmpty() {
		return nil
	}
	return mError
}

func refactorProjectIntegrationCrypto(ctx context.Context, db *gorp.DbMap, id int64) error {
	log.Info(ctx, "migrate.refactorProjectIntegrationCrypto> project_integration %d migration begin", id)

	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}

	query := `SELECT id, name, project_id, integration_model_id, config
	FROM project_integration 
	WHERE id = $1 
	AND sig IS NULL 
	FOR UPDATE SKIP LOCKED`

	defer tx.Rollback() // nolint

	var projectIntegration sdk.ProjectIntegration
	if err := tx.QueryRow(query, id).Scan(
		&projectIntegration.ID,
		&projectIntegration.Name,
		&projectIntegration.ProjectID,
		&projectIntegration.IntegrationModelID,
		&projectIntegration.Config,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return sdk.WrapError(err, "unable to select and lock application %d", id)
	}

	newCfg := projectIntegration.Config.Clone()
	if err := newCfg.DecryptSecrets(secret.DecryptValue); err != nil {
		return sdk.WrapError(err, "unable to encrypt config PublicConfigurations")
	}
	projectIntegration.Config = newCfg
	oldCfg := projectIntegration.Config.Clone()

	if err := integration.UpdateIntegration(tx, projectIntegration); err != nil {
		return sdk.WithStack(err)
	}

	newProjectIntegration, err := integration.LoadProjectIntegrationByID(tx, id, true)
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(oldCfg, newProjectIntegration.Config) {
		return sdk.WrapError(errors.New("verification error"), "project_integration %d migration failure", id)
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	log.Info(ctx, "migrate.refactorProjectIntegrationCrypto> project_integration %d migration end", id)
	return nil
}
