package integration

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
)

// LoadModels load integration models
func LoadModels(db gorp.SqlExecutor) ([]sdk.IntegrationModel, error) {
	var pm []integrationModel
	if _, err := db.Select(&pm, "SELECT * from integration_model"); err != nil {
		return nil, sdk.WrapError(err, "Cannot select all integration model")
	}

	var integrations = make([]sdk.IntegrationModel, len(pm))
	for i, p := range pm {
		if err := p.PostGet(db); err != nil {
			return nil, sdk.WrapError(err, "Cannot postGet integration model")
		}
		integrations[i] = sdk.IntegrationModel(p)
	}
	return integrations, nil
}

// LoadModel Load a integration model by its ID
func LoadModel(db gorp.SqlExecutor, modelID int64, clearPassword bool) (sdk.IntegrationModel, error) {
	var pm integrationModel
	if err := db.SelectOne(&pm, "SELECT * from integration_model where id = $1", modelID); err != nil {
		if err == sql.ErrNoRows {
			return sdk.IntegrationModel{}, sdk.NewErrorFrom(sdk.ErrNotFound, "Cannot select integration model %d", modelID)
		}
		return sdk.IntegrationModel{}, sdk.WrapError(err, "Cannot select integration model %d", modelID)
	}
	if clearPassword {
		for pfName, pfCfg := range pm.PublicConfigurations {
			newCfg := pfCfg.Clone()
			if err := newCfg.DecryptSecrets(secret.DecryptValue); err != nil {
				return sdk.IntegrationModel{}, sdk.WrapError(err, "unable to encrypt config")
			}
			pm.PublicConfigurations[pfName] = newCfg
		}
	} else {
		for pfName, pfCfg := range pm.PublicConfigurations {
			newCfg := pfCfg.Clone()
			newCfg.HideSecrets()
			pm.PublicConfigurations[pfName] = newCfg
		}
	}
	return sdk.IntegrationModel(pm), nil
}

// LoadModelByName Load a integration model by its name
func LoadModelByName(db gorp.SqlExecutor, name string, clearPassword bool) (sdk.IntegrationModel, error) {
	var pm integrationModel
	if err := db.SelectOne(&pm, "SELECT * from integration_model where name = $1", name); err != nil {
		if err == sql.ErrNoRows {
			return sdk.IntegrationModel{}, sdk.NewErrorFrom(sdk.ErrNotFound, "integration model %s not found", name)
		}
		return sdk.IntegrationModel{}, sdk.WrapError(err, "Cannot select integration model %s", name)
	}
	if clearPassword {
		for pfName, pfCfg := range pm.PublicConfigurations {
			newCfg := pfCfg.Clone()
			if err := newCfg.DecryptSecrets(secret.DecryptValue); err != nil {
				return sdk.IntegrationModel{}, sdk.WrapError(err, "unable to encrypt config")
			}
			pm.PublicConfigurations[pfName] = newCfg
		}
	} else {
		for pfName, pfCfg := range pm.PublicConfigurations {
			newCfg := pfCfg.Clone()
			newCfg.HideSecrets()
			pm.PublicConfigurations[pfName] = newCfg
		}
	}
	return sdk.IntegrationModel(pm), nil
}

// ModelExists tests if the given model exists
func ModelExists(db gorp.SqlExecutor, name string) (bool, error) {
	var count = 0
	if err := db.QueryRow("select count(1) from integration_model where name = $1 GROUP BY id", name).Scan(&count); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, sdk.WrapError(err, "ModelExists")
	}
	return count > 0, nil
}

// InsertModel inserts a integration model in database
func InsertModel(db gorp.SqlExecutor, m *sdk.IntegrationModel) error {
	dbm := integrationModel(*m)
	if err := db.Insert(&dbm); err != nil {
		return sdk.WrapError(err, "Unable to insert integration model %s", m.Name)
	}
	*m = sdk.IntegrationModel(dbm)
	return nil
}

// UpdateModel updates a integration model in database
func UpdateModel(db gorp.SqlExecutor, m *sdk.IntegrationModel) error {
	dbm := integrationModel(*m)
	if n, err := db.Update(&dbm); err != nil {
		return sdk.WrapError(err, "Unable to update integration model %s", m.Name)
	} else if n == 0 {
		return sdk.NewErrorFrom(sdk.ErrNotFound, "Unable to update integration model %s", m.Name)
	}
	return nil
}

// DeleteModel deletes a integration model in database
func DeleteModel(db gorp.SqlExecutor, id int64) error {
	m, err := LoadModel(db, id, false)
	if err != nil {
		return sdk.WrapError(err, "DeleteModel")
	}

	dbm := integrationModel(m)
	if _, err := db.Delete(&dbm); err != nil {
		return sdk.WrapError(err, "unable to delete model %s", m.Name)
	}

	return nil
}

// PostGet is a db hook
func (pm *integrationModel) PostGet(db gorp.SqlExecutor) error {
	var res = struct {
		DefaultConfig           sql.NullString `db:"default_config"`
		DeploymentDefaultConfig sql.NullString `db:"deployment_default_config"`
		PluginName              sql.NullString `db:"plugin_name"`
		PublicConfigurations    sql.NullString `db:"public_configurations"`
	}{}

	query := `SELECT default_config, grpc_plugin.name as "plugin_name", deployment_default_config, public_configurations
	FROM integration_model
	LEFT OUTER JOIN grpc_plugin ON grpc_plugin.integration_model_id = integration_model.id
	WHERE integration_model.id = $1`
	if err := db.SelectOne(&res, query, pm.ID); err != nil {
		return sdk.WrapError(err, "Cannot get default_config, integration_model_plugin, deployment_default_config for integrationModel: %v", pm.ID)
	}

	if err := gorpmapping.JSONNullString(res.DefaultConfig, &pm.DefaultConfig); err != nil {
		return sdk.WrapError(err, "Unable to load default_config")
	}

	if err := gorpmapping.JSONNullString(res.DeploymentDefaultConfig, &pm.DeploymentDefaultConfig); err != nil {
		return sdk.WrapError(err, "Unable to load deployment_default_config")
	}

	if err := gorpmapping.JSONNullString(res.PublicConfigurations, &pm.PublicConfigurations); err != nil {
		return sdk.WrapError(err, "Unable to load public_configurations")
	}
	return nil
}

// PostInsert is a db hook
func (pm *integrationModel) PostInsert(db gorp.SqlExecutor) error {
	return pm.PostUpdate(db)
}

// PostUpdate is a db hook
func (pm *integrationModel) PostUpdate(db gorp.SqlExecutor) error {
	if pm.DefaultConfig == nil {
		pm.DefaultConfig = sdk.IntegrationConfig{}
	}

	defaultConfig, _ := gorpmapping.JSONToNullString(pm.DefaultConfig)
	deploymentDefaultConfig, _ := gorpmapping.JSONToNullString(pm.DeploymentDefaultConfig)
	cfg := make(map[string]sdk.IntegrationConfig, len(pm.PublicConfigurations))
	for pfName, pfCfg := range pm.PublicConfigurations {
		newCfg := pfCfg.Clone()
		if err := newCfg.EncryptSecrets(secret.EncryptValue); err != nil {
			return sdk.WrapError(err, "unable to encrypt config")
		}
		cfg[pfName] = newCfg
	}
	publicConfig, _ := gorpmapping.JSONToNullString(cfg)

	_, err := db.Exec("update integration_model set default_config = $2, deployment_default_config = $3, public_configurations = $4 where id = $1", pm.ID, defaultConfig, deploymentDefaultConfig, publicConfig)
	return sdk.WrapError(err, "Unable to update integration_model")
}
