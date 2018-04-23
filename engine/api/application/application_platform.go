package application

import (
	"database/sql"
	"encoding/base64"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/secret"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
)

// LoadDeploymentStrategies loads the deployment strategies for an application
func LoadDeploymentStrategies(db gorp.SqlExecutor, appID int64, withClearPassword bool) (map[string]sdk.PlatformConfig, error) {
	query := `SELECT project_platform.name, application_deployment_strategy.config
	FROM application_deployment_strategy
	JOIN project_platform ON project_platform.id = application_deployment_strategy.project_platform_id
	JOIN platform_model ON platform_model.id = project_platform.platform_model_id
	WHERE application_deployment_strategy.application_id = $1`

	res := []struct {
		Name   string         `db:"name"`
		Config sql.NullString `db:"config"`
	}{}

	if _, err := db.Select(&res, query, appID); err != nil {
		return nil, sdk.WrapError(err, "application.LoadDeploymentStrategies> unable to load deployment strategies")
	}

	deps := map[string]sdk.PlatformConfig{}
	for _, r := range res {
		cfg := sdk.PlatformConfig{}
		if err := gorpmapping.JSONNullString(r.Config, &cfg); err != nil {
			return nil, sdk.WrapError(err, "application.LoadDeploymentStrategies> unable to parse config")
		}
		//Parse the config and replace password values
		newCfg := sdk.PlatformConfig{}
		for k, v := range cfg {
			if v.Type == sdk.PlatformConfigTypePassword {
				if !withClearPassword {
					newCfg[k] = sdk.PlatformConfigValue{
						Type:  sdk.PlatformConfigTypePassword,
						Value: sdk.PasswordPlaceholder,
					}
				} else {
					s, err := base64.StdEncoding.DecodeString(v.Value)
					if err != nil {
						return nil, sdk.WrapError(err, "application.LoadDeploymentStrategies> unable to decode encrypted value")
					}

					decryptedValue, err := secret.Decrypt(s)
					if err != nil {
						return nil, sdk.WrapError(err, "application.LoadDeploymentStrategies> unable to decrypt secret value")
					}

					newCfg[k] = sdk.PlatformConfigValue{
						Type:  sdk.PlatformConfigTypePassword,
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

// SetDeploymentStrategy update the application_deployment_strategy table
func SetDeploymentStrategy(db gorp.SqlExecutor, projID, appID, pfID int64, cfg sdk.PlatformConfig) error {
	//Parse the config and encrypt password values
	newcfg := sdk.PlatformConfig{}
	for k, v := range cfg {
		if v.Type == sdk.PlatformConfigTypePassword {
			e, err := secret.Encrypt([]byte(v.Value))
			if err != nil {
				return sdk.WrapError(err, "SetDeploymentStrategy> unable to encrypt data")
			}
			newcfg[k] = sdk.PlatformConfigValue{
				Type:  sdk.PlatformConfigTypePassword,
				Value: base64.StdEncoding.EncodeToString(e),
			}
		} else {
			newcfg[k] = sdk.PlatformConfigValue{
				Type:  sdk.PlatformConfigTypeString,
				Value: v.Value,
			}
		}
	}

	count, err := db.SelectInt(`SELECT COUNT(1)
	FROM application_deployment_strategy 
	JOIN project_platform ON project_platform.id = application_deployment_strategy.project_platform_id
	WHERE project_platform.project_id = $1
	AND project_platform.platform_model_id = $2
	AND application_deployment_strategy.application_id = $3`, projID, pfID, appID)

	if err != nil {
		return sdk.WrapError(err, "SetDeploymentStrategy> unable to check if deployment strategy exist")
	}

	scfg, err := gorpmapping.JSONToNullString(newcfg)
	if err != nil {
		return sdk.WrapError(err, "SetDeploymentStrategy> unable to parse deployment strategy ")
	}

	if count == 1 {
		query := `UPDATE application_deployment_strategy 
		SET config = $1
		FROM project_platform
		WHERE project_platform.id = application_deployment_strategy.project_platform_id
		AND application_deployment_strategy.application_id = $2
		AND project_platform.project_id = $3
		AND project_platform.platform_model_id = $4`
		if _, err := db.Exec(query, scfg, appID, projID, pfID); err != nil {
			return sdk.WrapError(err, "SetDeploymentStrategy> unable to update deployment strategy")
		}
		return nil
	}

	query := `INSERT INTO application_deployment_strategy (config, application_id, project_platform_id) 
	VALUES ($1, $2, 
		(
			SELECT 	project_platform.id 
			FROM project_platform
			WHERE project_platform.project_id = $3
			AND project_platform.platform_model_id = $4
		)
	)`
	if _, err := db.Exec(query, scfg, appID, projID, pfID); err != nil {
		return sdk.WrapError(err, "SetDeploymentStrategy> unable to update deployment strategy")
	}
	return nil
}
