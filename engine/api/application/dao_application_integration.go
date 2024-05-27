package application

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

type dbIntegration sdk.IntegrationConfig

// application_deployment_strategy
type dbApplicationDeploymentStrategy struct {
	gorpmapper.SignedEntity
	ID                   int64         `db:"id"`
	ProjectIntegrationID int64         `db:"project_integration_id"`
	ApplicationID        int64         `db:"application_id"`
	Config               dbIntegration `db:"cipher_config" gorpmapping:"encrypted,ProjectIntegrationID,ApplicationID"` //config
}

func (e dbApplicationDeploymentStrategy) Canonical() gorpmapper.CanonicalForms {
	var _ = []interface{}{e.ID, e.ProjectIntegrationID, e.ApplicationID}
	return gorpmapper.CanonicalForms{
		"{{printf .ID}}{{printf .ProjectIntegrationID}}{{printf .ApplicationID}}",
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
func LoadDeploymentStrategies(ctx context.Context, db gorp.SqlExecutor, appID int64, withClearPassword bool) (map[string]sdk.IntegrationConfig, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
	  FROM application_deployment_strategy
    WHERE application_id = $1
  `).Args(appID)

	var res []dbApplicationDeploymentStrategy
	if err := gorpmapping.GetAll(ctx, db, query, &res, gorpmapping.GetOptions.WithDecryption); err != nil {
		return nil, sdk.WrapError(err, "unable to load deployment strategies")
	}

	deps := make(map[string]sdk.IntegrationConfig, len(res))
	for _, r := range res {
		isValid, err := gorpmapping.CheckSignature(r, r.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "application.LoadDeploymentStrategies> application_deployment_strategy %d data corrupted", appID)
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
		projectIntegration, err := integration.LoadProjectIntegrationByID(ctx, db, r.ProjectIntegrationID)
		if err != nil {
			return nil, sdk.WrapError(err, "unable to find project integration name for ID=%d", r.ProjectIntegrationID)
		}
		for name, val := range projectIntegration.Model.AdditionalDefaultConfig {
			if _, ok := newCfg[name]; !ok {
				newCfg[name] = val
			}
		}
		deps[projectIntegration.Name] = newCfg
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

func getProjectIntegrationID(db gorp.SqlExecutor, projID, pfModelID int64, ppfName string) (int64, error) {
	query := gorpmapping.NewQuery(`SELECT project_integration.id
	FROM project_integration
	WHERE project_integration.project_id = $1
	AND project_integration.integration_model_id = $2
	AND project_integration.name = $3`).Args(projID, pfModelID, ppfName)
	id, err := gorpmapping.GetInt(db, query)
	if err != nil {
		return -1, err
	}
	return id, nil
}

// SetDeploymentStrategy update the application_deployment_strategy table
func SetDeploymentStrategy(db gorpmapper.SqlExecutorWithTx, projID, appID, pfModelID int64, ppfName string, cfg sdk.IntegrationConfig) error {
	projectIntegrationID, err := getProjectIntegrationID(db, projID, pfModelID, ppfName)
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

// LoadAllDeploymnentForAppsWithDecryption load all deployments for all given applications, with decryption
func LoadAllDeploymnentForAppsWithDecryption(ctx context.Context, db gorp.SqlExecutor, appIDs []int64) (map[int64]map[int64]sdk.IntegrationConfig, error) {
	return loadAllDeploymentsForApps(ctx, db, appIDs, gorpmapping.GetOptions.WithDecryption)
}

func loadAllDeploymentsForApps(ctx context.Context, db gorp.SqlExecutor, appsID []int64, opts ...gorpmapping.GetOptionFunc) (map[int64]map[int64]sdk.IntegrationConfig, error) {
	var res []dbApplicationDeploymentStrategy
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM application_deployment_strategy
		WHERE application_id = ANY($1)
		ORDER BY application_id
	`).Args(pq.Int64Array(appsID))
	if err := gorpmapping.GetAll(ctx, db, query, &res, opts...); err != nil {
		return nil, err
	}
	appsDeployments := make(map[int64]map[int64]sdk.IntegrationConfig)
	for i := range res {
		dbAppDeploy := res[i]
		isValid, err := gorpmapping.CheckSignature(dbAppDeploy, dbAppDeploy.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "application.loadAllDeploymentsForApps> application integration id %d data corrupted", dbAppDeploy.ID)
			continue
		}
		if _, ok := appsDeployments[dbAppDeploy.ApplicationID]; !ok {
			appsDeployments[dbAppDeploy.ApplicationID] = make(map[int64]sdk.IntegrationConfig, 0)
		}
		appsDeployments[dbAppDeploy.ApplicationID][dbAppDeploy.ProjectIntegrationID] = dbAppDeploy.IntegrationConfig()
	}
	return appsDeployments, nil
}
