package migrate

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// RefactorApplicationCrypto .
func RefactorApplicationCrypto(ctx context.Context, db *gorp.DbMap) error {
	query := "SELECT id FROM application WHERE sig IS NULL"
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
		if err := refactorApplicationCrypto(ctx, db, id); err != nil {
			mError.Append(err)
			log.Error(ctx, "migrate.RefactorApplicationCrypto> unable to migrate application %d: %v", id, err)
		}
	}

	if mError.IsEmpty() {
		return nil
	}
	return mError
}

func refactorApplicationCrypto(ctx context.Context, db *gorp.DbMap, id int64) error {
	log.Info(ctx, "migrate.refactorApplicationCrypto> application %d migration begin", id)

	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}

	defer tx.Rollback() // nolint

	// First part is application encryption and signature for vcs_strategy
	query := "SELECT project_id, name, vcs_strategy FROM application WHERE id = $1 AND sig IS NULL FOR UPDATE SKIP LOCKED"
	var projectID int64
	var btes []byte
	var name string
	if err := tx.QueryRow(query, id).Scan(&projectID, &name, &btes); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return sdk.WrapError(err, "unable to select and lock application %d", id)
	}

	var vcsStrategy sdk.RepositoryStrategy
	var clearPWD []byte
	if len(btes) != 0 {
		if err := json.Unmarshal(btes, &vcsStrategy); err != nil {
			return sdk.WrapError(err, "unable to unmarshal application RepositoryStrategy %d", id)
		}

		encryptedPassword, err := base64.StdEncoding.DecodeString(vcsStrategy.Password)
		if err != nil {
			return sdk.WrapError(err, "unable to decode password for application %d", id)
		}

		clearPWD, err = secret.Decrypt([]byte(encryptedPassword))
		if err != nil {
			return sdk.WrapError(err, "Unable to decrypt password for application %d", id)
		}

		vcsStrategy.Password = string(clearPWD)
	}

	var tmpApp = sdk.Application{
		ID:                 id,
		Name:               name,
		ProjectID:          projectID,
		RepositoryStrategy: vcsStrategy,
	}
	// We are faking the DAO layer with updating only the name to perform updating of the encrypted columns and signature
	var vcsStrategyColFilter = func(col *gorp.ColumnMap) bool {
		return col.ColumnName == "name"
	}

	if err := application.UpdateColumns(tx, &tmpApp, vcsStrategyColFilter); err != nil {
		return sdk.WrapError(err, "Unable to update application %d", id)
	}

	// No it is time to validate by loading from the DAO
	app, err := application.LoadByIDWithClearVCSStrategyPassword(tx, id)
	if err != nil {
		return sdk.WrapError(err, "Unable to reload application %d", id)
	}

	if app.RepositoryStrategy.Password != string(clearPWD) {
		return sdk.WrapError(errors.New("verification error"), "Application %d migration failure", id)
	}

	// Second part is application_deployment_strategy
	deploymentStragegies, err := loadApplicationDeploymentStrategies(tx, id)
	if err != nil {
		return sdk.WrapError(err, "unable to load application_deployment_strategy for application %d", id)
	}

	proj, err := project.LoadByID(tx, projectID, project.LoadOptions.WithIntegrations)
	if err != nil {
		return sdk.WrapError(err, "unable to load project %d", projectID)
	}

	for pfName := range deploymentStragegies {
		var pf *sdk.ProjectIntegration
		for i := range proj.Integrations {
			if proj.Integrations[i].Name == pfName {
				pf = &proj.Integrations[i]
				break
			}
		}
		if err := application.SetDeploymentStrategy(tx, proj.ID, id, pf.Model.ID, pfName, deploymentStragegies[pfName]); err != nil {
			return sdk.WrapError(err, "unable to set deployment strategy")
		}
	}

	// Reload all the things to check all deployments strategies
	app, err = application.LoadByID(tx, id, application.LoadOptions.WithClearDeploymentStrategies)
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(deploymentStragegies, app.DeploymentStrategies) {
		log.Debug("expected: %+v", deploymentStragegies)
		log.Debug("actual: %+v", app.DeploymentStrategies)
		return sdk.WrapError(err, "deployment strategies are not equals...")
	}

	log.Info(ctx, "migrate.refactorApplicationCrypto> application %d migration end", id)

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	return nil
}

// loadApplicationDeploymentStrategies loads the deployment strategies for an application
func loadApplicationDeploymentStrategies(db gorp.SqlExecutor, appID int64) (map[string]sdk.IntegrationConfig, error) {
	query := `SELECT project_integration.name, application_deployment_strategy.config
	FROM application_deployment_strategy
	JOIN project_integration ON project_integration.id = application_deployment_strategy.project_integration_id
	JOIN integration_model ON integration_model.id = project_integration.integration_model_id
	WHERE application_deployment_strategy.application_id = $1`

	res := []struct {
		Name   string         `db:"name"`
		Config sql.NullString `db:"config"`
	}{}

	if _, err := db.Select(&res, query, appID); err != nil {
		return nil, sdk.WrapError(err, "unable to load deployment strategies")
	}

	deps := make(map[string]sdk.IntegrationConfig, len(res))
	for _, r := range res {
		cfg := sdk.IntegrationConfig{}
		if err := gorpmapping.JSONNullString(r.Config, &cfg); err != nil {
			return nil, sdk.WrapError(err, "unable to parse config")
		}
		//Parse the config and replace password values
		newCfg := sdk.IntegrationConfig{}
		for k, v := range cfg {
			if v.Type == sdk.IntegrationConfigTypePassword {
				s, err := base64.StdEncoding.DecodeString(v.Value)
				if err != nil {
					return nil, sdk.WrapError(err, "unable to decode encrypted value")
				}

				decryptedValue, err := secret.Decrypt(s)
				if err != nil {
					return nil, sdk.WrapError(err, "unable to decrypt secret value")
				}

				newCfg[k] = sdk.IntegrationConfigValue{
					Type:  sdk.IntegrationConfigTypePassword,
					Value: string(decryptedValue),
				}
			} else {
				newCfg[k] = v
			}
		}
		deps[r.Name] = newCfg
	}

	return deps, nil
}
