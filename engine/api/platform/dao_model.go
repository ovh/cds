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
func LoadModel(db gorp.SqlExecutor, modelID int64) (sdk.PlatformModel, error) {
	var pm platformModel
	if err := db.SelectOne(&pm, "SELECT * from platform_model where id = $1", modelID); err != nil {
		return sdk.PlatformModel{}, sdk.WrapError(sdk.ErrNotFound, "LoadModel> Cannot select platform model %d", modelID)
	}
	return sdk.PlatformModel(pm), nil
}

// LoadModelByName Load a platform model by its name
func LoadModelByName(db gorp.SqlExecutor, name string) (sdk.PlatformModel, error) {
	var pm platformModel
	if err := db.SelectOne(&pm, "SELECT * from platform_model where name = $1", name); err != nil {
		return sdk.PlatformModel{}, sdk.WrapError(sdk.ErrNotFound, "LoadModel> Cannot select platform model %s", name)
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
	m, err := LoadModel(db, id)
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
		DefaultConfig       sql.NullString `db:"default_config"`
		PlatformModelPlugin sql.NullString `db:"platform_mode_plugin"`
	}{}

	query := "SELECT default_config, platform_model_plugin FROM platform_model where id = $1"
	if _, err := db.Select(&res, query, pm.ID); err != nil {
		return sdk.WrapError(err, "PlatformModel.PostGet> Cannot get default_config, platform_model_plugin")
	}
	if err := gorpmapping.JSONNullString(res.DefaultConfig, &pm.DefaultConfig); err != nil {
		return sdk.WrapError(err, "PlatformModel.PostGet> Unable to load default_config")
	}
	if err := gorpmapping.JSONNullString(res.PlatformModelPlugin, &pm.PlatformModelPlugin); err != nil {
		return sdk.WrapError(err, "PlatformModel.PostGet> Unable to load platform_model_plugin")
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

	defaultConfig, err := gorpmapping.JSONToNullString(pm.DefaultConfig)
	platformModelPlugin, err := gorpmapping.JSONToNullString(pm.PlatformModelPlugin)

	_, err = db.Exec("update platform_model set default_config = $2, platform_model_plugin = $3 where id = $1", pm.ID, defaultConfig, platformModelPlugin)
	return sdk.WrapError(err, "PostUpdate> Unable to update platform_model")
}
