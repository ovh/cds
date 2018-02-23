package platform

import (
	"database/sql"
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var (
	Models = []sdk.PlatformModel{
		sdk.KafkaPlatform,
	}
)

// LoadModel Load a platform model by its ID
func LoadModel(db gorp.SqlExecutor, modelID int64) (sdk.PlatformModel, error) {
	var pm PlatformModel
	if err := db.SelectOne(&pm, "SELECT * from platform_model where id = $1", modelID); err != nil {
		return sdk.PlatformModel{}, sdk.WrapError(err, "LoadModel> Cannot select platform model %d", modelID)
	}
	return sdk.PlatformModel(pm), nil
}

// CreatePlatformModels create platforms models
func CreatePlatformModels(db *gorp.DbMap) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "CreatePlatformModels> Unable to start transaction")
	}
	defer tx.Rollback()

	if _, err := tx.Exec("LOCK TABLE platform_model IN ACCESS EXCLUSIVE MODE"); err != nil {
		return sdk.WrapError(err, "CreatePlatformModels> Unable to lock table")
	}

	for i := range Models {
		p := &Models[i]
		ok, err := checkPlatformExist(tx, p)
		if err != nil {
			return sdk.WrapError(err, "CreatePlatformModels")
		}

		if !ok {
			log.Debug("CreatePlatformModels> inserting platform config: %s", p.Name)
			if err := InsertPlatformModel(tx, p); err != nil {
				return sdk.WrapError(err, "CreatePlatformModels error on insert")
			}
		} else {
			log.Debug("CreatePlatformModels> updating platform config: %s", p.Name)
			// update default values
			if err := UpdatePlatformModel(tx, p); err != nil {
				return sdk.WrapError(err, "CreatePlatformModels  error on update")
			}
		}
	}
	return tx.Commit()
}

func checkPlatformExist(db gorp.SqlExecutor, p *sdk.PlatformModel) (bool, error) {
	var count = 0
	if err := db.QueryRow("select count(1), id from platform_model where name = $1 group by id", p.Name).Scan(&count, &p.ID); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, sdk.WrapError(err, "checkPlatformExist")
	}
	return count > 0, nil
}

// InsertPlatformModel inserts a platform model in database
func InsertPlatformModel(db gorp.SqlExecutor, m *sdk.PlatformModel) error {
	dbm := PlatformModel(*m)
	if err := db.Insert(&dbm); err != nil {
		return sdk.WrapError(err, "InsertPlatformModel> Unable to insert platform model %s", m.Name)
	}
	*m = sdk.PlatformModel(dbm)
	return nil
}

// UpdatePlatformModel updates a platform model in database
func UpdatePlatformModel(db gorp.SqlExecutor, m *sdk.PlatformModel) error {
	dbm := PlatformModel(*m)
	if n, err := db.Update(&dbm); err != nil {
		return sdk.WrapError(err, "UpdatePlatformModel> Unable to update platform model %s", m.Name)
	} else if n == 0 {
		return sdk.WrapError(sdk.ErrNotFound, "UpdatePlatformModel> Unable to update platform model %s", m.Name)
	}
	return nil
}

// PostGet is a db hook
func (pm *PlatformModel) PostGet(db gorp.SqlExecutor) error {
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
func (pm *PlatformModel) PostInsert(db gorp.SqlExecutor) error {
	return pm.PostUpdate(db)
}

// PostUpdate is a db hook
func (pm *PlatformModel) PostUpdate(db gorp.SqlExecutor) error {
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
