package application

import (
	"database/sql"
	"encoding/base64"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
)

// LoadDeploymentStrategies loads the deployment strategies for an application
func LoadDeploymentStrategies(db gorp.SqlExecutor, appID int64, withClearPassword bool) (map[string]sdk.IntegrationConfig, error) {
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
				if !withClearPassword {
					newCfg[k] = sdk.IntegrationConfigValue{
						Type:  sdk.IntegrationConfigTypePassword,
						Value: sdk.PasswordPlaceholder,
					}
				} else {
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
				}
			} else {
				newCfg[k] = v
			}
		}
		deps[r.Name] = newCfg
	}

	return deps, nil
}

// DeleteAllDeploymentStrategies delete all lines in table application_deployment_strategy for one application
func DeleteAllDeploymentStrategies(db gorp.SqlExecutor, appID int64) error {
	_, err := db.Exec("DELETE FROM application_deployment_strategy WHERE application_id = $1", appID)
	return sdk.WrapError(err, "DeleteAllDeploymentStrategies")
}

// DeleteDeploymentStrategy delete a line in table application_deployment_strategy
func DeleteDeploymentStrategy(db gorp.SqlExecutor, projID, appID, pfID int64) error {
	query := `DELETE FROM application_deployment_strategy 
	WHERE application_id = $1
	AND project_integration_id IN (
		SELECT 	project_integration.id 
		FROM project_integration
		WHERE project_integration.project_id = $2
		AND project_integration.id = $3
	)`

	_, err := db.Exec(query, appID, projID, pfID)
	return sdk.WrapError(err, "unable to delete deployment strategy appID=%d pfID=%d projID=%d", appID, pfID, projID)
}

// SetDeploymentStrategy update the application_deployment_strategy table
func SetDeploymentStrategy(db gorp.SqlExecutor, projID, appID, pfID int64, ppfName string, cfg sdk.IntegrationConfig) error {
	//Parse the config and encrypt password values
	newcfg := sdk.IntegrationConfig{}
	for k, v := range cfg {
		if v.Type == sdk.IntegrationConfigTypePassword {
			e, err := secret.Encrypt([]byte(v.Value))
			if err != nil {
				return sdk.WrapError(err, "unable to encrypt data")
			}
			newcfg[k] = sdk.IntegrationConfigValue{
				Type:        sdk.IntegrationConfigTypePassword,
				Value:       base64.StdEncoding.EncodeToString(e),
				Description: v.Description,
			}
		} else {
			newcfg[k] = sdk.IntegrationConfigValue{
				Type:        sdk.IntegrationConfigTypeString,
				Value:       v.Value,
				Description: v.Description,
			}
		}
	}

	count, err := db.SelectInt(`SELECT COUNT(1)
	FROM application_deployment_strategy 
	JOIN project_integration ON project_integration.id = application_deployment_strategy.project_integration_id
	WHERE project_integration.project_id = $1
	AND project_integration.integration_model_id = $2
	AND application_deployment_strategy.application_id = $3
	AND project_integration.name = $4`, projID, pfID, appID, ppfName)

	if err != nil {
		return sdk.WrapError(err, "unable to check if deployment strategy exist")
	}

	scfg, err := gorpmapping.JSONToNullString(newcfg)
	if err != nil {
		return sdk.WrapError(err, "unable to parse deployment strategy ")
	}

	if count == 1 {
		query := `UPDATE application_deployment_strategy 
		SET config = $1
		FROM project_integration
		WHERE project_integration.id = application_deployment_strategy.project_integration_id
		AND application_deployment_strategy.application_id = $2
		AND project_integration.project_id = $3
		AND project_integration.integration_model_id = $4
		AND project_integration.name = $5`
		if _, err := db.Exec(query, scfg, appID, projID, pfID, ppfName); err != nil {
			return sdk.WrapError(err, "unable to update deployment strategy")
		}
		return nil
	}

	query := `INSERT INTO application_deployment_strategy (config, application_id, project_integration_id) 
	VALUES ($1, $2, 
		(
			SELECT 	project_integration.id 
			FROM project_integration
			WHERE project_integration.project_id = $3
			AND project_integration.integration_model_id = $4
			AND project_integration.name = $5
		)
	)`
	if _, err := db.Exec(query, scfg, appID, projID, pfID, ppfName); err != nil {
		return sdk.WrapError(err, "unable to update deployment strategy")
	}
	return nil
}
