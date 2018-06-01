package platform

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// LoadModels load platform models
func LoadModels(db gorp.SqlExecutor) ([]sdk.PlatformModel, error) {
	var pm []platformModel
	if _, err := db.Select(&pm, "SELECT * from platform_model"); err != nil {
		return nil, sdk.WrapError(err, "LoadModels> Cannot select all platform model")
	}

	var platforms = make([]sdk.PlatformModel, len(pm))
	for i, p := range pm {
		if err := p.PostGet(db); err != nil {
			return nil, sdk.WrapError(err, "LoadModels> Cannot post et platform model")
		}
		platforms[i] = sdk.PlatformModel(p)
	}
	return platforms, nil
}

// LoadModel Load a platform model by its ID
func LoadModel(db gorp.SqlExecutor, modelID int64, clearPassword bool) (sdk.PlatformModel, error) {
	var pm platformModel
	if err := db.SelectOne(&pm, "SELECT * from platform_model where id = $1", modelID); err != nil {
		if err == sql.ErrNoRows {
			return sdk.PlatformModel{}, sdk.WrapError(sdk.ErrNotFound, "LoadModel> Cannot select platform model %d", modelID)
		}
		return sdk.PlatformModel{}, sdk.WrapError(err, "LoadModel> Cannot select platform model %d", modelID)
	}
	if clearPassword {
		for pfName, pfCfg := range pm.PublicConfigurations {
			newCfg := pfCfg.Clone()
			if err := newCfg.DecryptSecrets(decryptPlatformValue); err != nil {
				return sdk.PlatformModel{}, sdk.WrapError(err, "LoadModel> unable to encrypt config")
			}
			pm.PublicConfigurations[pfName] = newCfg
		}
	}
	return sdk.PlatformModel(pm), nil
}

// LoadModelByName Load a platform model by its name
func LoadModelByName(db gorp.SqlExecutor, name string, clearPassword bool) (sdk.PlatformModel, error) {
	var pm platformModel
	if err := db.SelectOne(&pm, "SELECT * from platform_model where name = $1", name); err != nil {
		if err == sql.ErrNoRows {
			return sdk.PlatformModel{}, sdk.WrapError(sdk.ErrNotFound, "LoadModel> platform model %s not found", name)
		}
		return sdk.PlatformModel{}, sdk.WrapError(err, "LoadModel> Cannot select platform model %s", name)
	}
	if clearPassword {
		for pfName, pfCfg := range pm.PublicConfigurations {
			newCfg := pfCfg.Clone()
			if err := newCfg.DecryptSecrets(decryptPlatformValue); err != nil {
				return sdk.PlatformModel{}, sdk.WrapError(err, "LoadModel> unable to encrypt config")
			}
			pm.PublicConfigurations[pfName] = newCfg
		}
	}
	return sdk.PlatformModel(pm), nil
}

// ModelExists tests if the given model exists
func ModelExists(db gorp.SqlExecutor, name string) (bool, error) {
	var count = 0
	if err := db.QueryRow("select count(1) from platform_model where name = $1 GROUP BY id", name).Scan(&count); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, sdk.WrapError(err, "ModelExists")
	}
	return count > 0, nil
}

// InsertModel inserts a platform model in database
func InsertModel(db gorp.SqlExecutor, m *sdk.PlatformModel) error {
	dbm := platformModel(*m)
	if err := db.Insert(&dbm); err != nil {
		return sdk.WrapError(err, "InsertModel> Unable to insert platform model %s", m.Name)
	}
	*m = sdk.PlatformModel(dbm)
	return nil
}

// UpdateModel updates a platform model in database
func UpdateModel(db gorp.SqlExecutor, m *sdk.PlatformModel) error {
	dbm := platformModel(*m)
	if n, err := db.Update(&dbm); err != nil {
		return sdk.WrapError(err, "UpdateModel> Unable to update platform model %s", m.Name)
	} else if n == 0 {
		return sdk.WrapError(sdk.ErrNotFound, "UpdateModel> Unable to update platform model %s", m.Name)
	}
	return nil
}

// DeleteModel deletes a platform model in database
func DeleteModel(db gorp.SqlExecutor, id int64) error {
	m, err := LoadModel(db, id, false)
	if err != nil {
		return sdk.WrapError(err, "DeleteModel")
	}

	dbm := platformModel(m)
	if _, err := db.Delete(&dbm); err != nil {
		return sdk.WrapError(err, "DeleteModel> unable to delete model %s", m.Name)
	}

	return nil
}

// PostGet is a db hook
func (pm *platformModel) PostGet(db gorp.SqlExecutor) error {
	var res = struct {
		DefaultConfig           sql.NullString `db:"default_config"`
		DeploymentDefaultConfig sql.NullString `db:"deployment_default_config"`
		PluginName              sql.NullString `db:"plugin_name"`
		PublicConfigurations    sql.NullString `db:"public_configurations"`
	}{}

	query := `SELECT default_config, grpc_plugin.name as "plugin_name", deployment_default_config, public_configurations
	FROM platform_model 
	LEFT OUTER JOIN grpc_plugin ON grpc_plugin.id = platform_model.grpc_plugin_id
	WHERE platform_model.id = $1`
	if err := db.SelectOne(&res, query, pm.ID); err != nil {
		return sdk.WrapError(err, "PlatformModel.PostGet> Cannot get default_config, platform_model_plugin, deployment_default_config")
	}

	if err := gorpmapping.JSONNullString(res.DefaultConfig, &pm.DefaultConfig); err != nil {
		return sdk.WrapError(err, "PlatformModel.PostGet> Unable to load default_config")
	}

	if err := gorpmapping.JSONNullString(res.DeploymentDefaultConfig, &pm.DeploymentDefaultConfig); err != nil {
		return sdk.WrapError(err, "PlatformModel.PostGet> Unable to load deployment_default_config")
	}

	if err := gorpmapping.JSONNullString(res.PublicConfigurations, &pm.PublicConfigurations); err != nil {
		return sdk.WrapError(err, "PlatformModel.PostGet> Unable to load public_configurations")
	}

	if res.PluginName.Valid {
		pm.PluginName = res.PluginName.String
	}

	return nil
}

// PostInsert is a db hook
func (pm *platformModel) PostInsert(db gorp.SqlExecutor) error {
	return pm.PostUpdate(db)
}

// PostUpdate is a db hook
func (pm *platformModel) PostUpdate(db gorp.SqlExecutor) error {
	if pm.DefaultConfig == nil {
		pm.DefaultConfig = sdk.PlatformConfig{}
	}

	defaultConfig, _ := gorpmapping.JSONToNullString(pm.DefaultConfig)
	deploymentDefaultConfig, _ := gorpmapping.JSONToNullString(pm.DeploymentDefaultConfig)
	cfg := make(map[string]sdk.PlatformConfig, len(pm.PublicConfigurations))
	for pfName, pfCfg := range pm.PublicConfigurations {
		newCfg := pfCfg.Clone()
		if err := newCfg.EncryptSecrets(encryptPlatformValue); err != nil {
			return sdk.WrapError(err, "PostUpdate> unable to encrypt config")
		}
		cfg[pfName] = newCfg
	}
	publicConfig, _ := gorpmapping.JSONToNullString(cfg)

	_, err := db.Exec("update platform_model set default_config = $2, deployment_default_config = $3, public_configurations = $4 where id = $1", pm.ID, defaultConfig, deploymentDefaultConfig, publicConfig)
	return sdk.WrapError(err, "PostUpdate> Unable to update platform_model")
}
