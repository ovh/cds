package application

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gorpmapping"
	"github.com/ovh/cds/sdk/log"
)

type dbIntegration sdk.IntegrationConfig

// application_deployment_strategy
type dbApplicationDeploymentStrategy struct {
	gorpmapping.SignedEntity
	ID                   int64         `db:"id"`
	ProjectIntegrationID int64         `db:"project_integration_id"`
	ApplicationID        int64         `db:"application_id"`
	Config               dbIntegration `db:"cipher_config" gorpmapping:"encrypted,ProjectIntegrationID,ApplicationID"` //config
}

func (e dbApplicationDeploymentStrategy) Canonical() gorpmapping.CanonicalForms {
	var _ = []interface{}{e.ProjectIntegrationID, e.ApplicationID}
	return gorpmapping.CanonicalForms{
		"{{print .ProjectIntegrationID}}{{print .ApplicationID}}",
	}
}

func newDBApplicationDeploymentStrategy(projectIntegrationID, applicationID int64) *dbApplicationDeploymentStrategy {
	return &dbApplicationDeploymentStrategy{
		ProjectIntegrationID: projectIntegrationID,
		ApplicationID:        applicationID,
	}
}

func (e *dbApplicationDeploymentStrategy) SetConfig(cfg sdk.IntegrationConfig) {
	e.Config = dbIntegration(cfg)
}

func (e *dbApplicationDeploymentStrategy) IntegrationConfig() sdk.IntegrationConfig {
	return sdk.IntegrationConfig(e.Config).Clone()
}

// LoadDeploymentStrategies loads the deployment strategies for an application
func LoadDeploymentStrategies(db gorp.SqlExecutor, appID int64, withClearPassword bool) (map[string]sdk.IntegrationConfig, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
	  FROM application_deployment_strategy
    WHERE application_id = $1
  `).Args(appID)

	var res []dbApplicationDeploymentStrategy
	if err := gorpmapping.GetAll(context.Background(), db, query, &res, gorpmapping.GetOptions.WithDecryption); err != nil {
		return nil, sdk.WrapError(err, "unable to load deployment strategies")
	}

	deps := make(map[string]sdk.IntegrationConfig, len(res))
	for _, r := range res {
		isValid, err := gorpmapping.CheckSignature(r, r.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(context.Background(), "application.LoadDeploymentStrategies> application_deployment_strategy %d data corrupted", appID)
			continue
		}

		//Parse the config and replace password values by place holder if !withClearPassword
		newCfg := sdk.IntegrationConfig{}
		for k, v := range r.IntegrationConfig() {
			if v.Type == sdk.IntegrationConfigTypePassword {
				if !withClearPassword {
					newCfg[k] = sdk.IntegrationConfigValue{
						Type:  sdk.IntegrationConfigTypePassword,
						Value: sdk.PasswordPlaceholder,
					}
					continue
				}
				newCfg[k] = v
			} else {
				newCfg[k] = v
			}
		}
		// Sorry about that :(
		projectIntegrationName, err := db.SelectStr("SELECT name FROM project_integration WHERE id = $1 ", r.ProjectIntegrationID)
		if err != nil {
			return nil, sdk.WrapError(err, "unable to find project integration name for ID=%d", r.ProjectIntegrationID)
		}
		deps[projectIntegrationName] = newCfg
	}

	return deps, nil
}

// DeleteAllDeploymentStrategies delete all lines in table application_deployment_strategy for one application
func DeleteAllDeploymentStrategies(db gorp.SqlExecutor, appID int64) error {
	_, err := db.Exec("DELETE FROM application_deployment_strategy WHERE application_id = $1", appID)
	return sdk.WithStack(err)
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

func findDeploymentStrategy(db gorp.SqlExecutor, projectIntegrationID, applicationID int64) (*dbApplicationDeploymentStrategy, error) {
	query := gorpmapping.NewQuery(`SELECT *
	FROM application_deployment_strategy
	WHERE application_deployment_strategy.project_integration_id = $1
	AND application_deployment_strategy.application_id = $2`).Args(projectIntegrationID, applicationID)

	var i dbApplicationDeploymentStrategy
	found, err := gorpmapping.Get(context.Background(), db, query, &i)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to check if deployment strategy exist")
	}

	if !found {
		return nil, nil
	}

	return &i, nil
}

func getProjectIntegrationID(db gorp.SqlExecutor, projID, pfID int64, ppfName string) (int64, error) {
	query := gorpmapping.NewQuery(`SELECT project_integration.id
	FROM project_integration
	WHERE project_integration.project_id = $1
	AND project_integration.integration_model_id = $2
	AND project_integration.name = $3`).Args(projID, pfID, ppfName)
	id, err := gorpmapping.GetInt(db, query)
	if err != nil {
		return -1, err
	}
	return id, nil
}

// SetDeploymentStrategy update the application_deployment_strategy table
func SetDeploymentStrategy(db gorp.SqlExecutor, projID, appID, pfID int64, ppfName string, cfg sdk.IntegrationConfig) error {
	projectIntegrationID, err := getProjectIntegrationID(db, projID, pfID, ppfName)
	if err != nil {
		return err
	}

	dbCfg, err := findDeploymentStrategy(db, projectIntegrationID, appID)
	if err != nil {
		return err
	}

	if dbCfg == nil {
		dbCfg = newDBApplicationDeploymentStrategy(projectIntegrationID, appID)
		dbCfg.SetConfig(cfg.Clone())
		return gorpmapping.InsertAndSign(context.Background(), db, dbCfg)
	}

	dbCfg.SetConfig(cfg.Clone())
	return gorpmapping.UpdateAndSign(context.Background(), db, dbCfg)
}
