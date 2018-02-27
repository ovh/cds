package platform

import (
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var (
	// Models list available platform models
	Models = []sdk.PlatformModel{
		sdk.KafkaPlatform,
	}
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
		return sdk.PlatformModel{}, sdk.WrapError(err, "LoadModel> Cannot select platform model %d", modelID)
	}
	return sdk.PlatformModel(pm), nil
}

// CreateModels creates platforms models
func CreateModels(db *gorp.DbMap) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "CreateModels> Unable to start transaction")
	}
	defer tx.Rollback()

	if _, err := tx.Exec("LOCK TABLE platform_model IN ACCESS EXCLUSIVE MODE"); err != nil {
		return sdk.WrapError(err, "CreateModels> Unable to lock table")
	}

	for i := range Models {
		p := &Models[i]
		ok, err := ModelExists(tx, p)
		if err != nil {
			return sdk.WrapError(err, "CreateModels")
		}

		if !ok {
			log.Debug("CreateModels> inserting platform config: %s", p.Name)
			if err := InsertModel(tx, p); err != nil {
				return sdk.WrapError(err, "CreateModels error on insert")
			}
		} else {
			log.Debug("CreateModels> updating platform config: %s", p.Name)
			// update default values
			if err := UpdateModel(tx, p); err != nil {
				return sdk.WrapError(err, "CreateModels  error on update")
			}
		}
	}
	return tx.Commit()
}

// ModelExists tests if the given model exists
func ModelExists(db gorp.SqlExecutor, p *sdk.PlatformModel) (bool, error) {
	var count = 0
	if err := db.QueryRow("select count(1), id from platform_model where name = $1 GROUP BY id", p.Name).Scan(&count, &p.ID); err != nil {
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

// PostGet is a db hook
func (pm *platformModel) PostGet(db gorp.SqlExecutor) error {
	query := "SELECT default_config FROM platform_model where id = $1"
	s, err := db.SelectNullStr(query, pm.ID)
	if err != nil {
		return sdk.WrapError(err, "PlatformModel.PostGet> Cannot get default config")
	}
	if s.Valid {
		var defaultConfig sdk.PlatformConfig
		if err := json.Unmarshal([]byte(s.String), &defaultConfig); err != nil {
			return sdk.WrapError(err, "PlatformModel.PostGet> Cannot unmarshall default config")
		}
		pm.DefaultConfig = defaultConfig
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

	btes, errm := json.Marshal(pm.DefaultConfig)
	if errm != nil {
		return errm
	}
	if _, err := db.Exec("update platform_model set default_config = $2 where id = $1", pm.ID, btes); err != nil {
		return err
	}
	return nil
}
