package integration

import (
	"context"
	"database/sql"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

// LoadModels load integration models
func LoadModels(db gorp.SqlExecutor) ([]sdk.IntegrationModel, error) {
	var pms integrationModelSlice

	query := gorpmapping.NewQuery(`SELECT * FROM integration_model`)
	if err := gorpmapping.GetAll(context.Background(), db, query, &pms, gorpmapping.GetOptions.WithDecryption); err != nil {
		return nil, err
	}

	var res []sdk.IntegrationModel
	for _, pm := range pms {
		isValid, err := gorpmapping.CheckSignature(pm, pm.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(context.Background(), "integration.LoadModel> model  %d data corrupted", pm.ID)
			continue
		}
		x := pm.IntegrationModel
		x.Blur()
		res = append(res, x)
	}

	return res, nil
}

func LoadPublicModelsByTypeWithDecryption(db gorp.SqlExecutor, integrationType *sdk.IntegrationType) ([]sdk.IntegrationModel, error) {
	q := "SELECT * from integration_model WHERE public = true"
	if integrationType != nil {
		switch *integrationType {
		case sdk.IntegrationTypeEvent:
			q += " AND integration_model.event = true"
		case sdk.IntegrationTypeCompute:
			q += " AND integration_model.compute = true"
		case sdk.IntegrationTypeStorage:
			q += " AND integration_model.storage = true"
		case sdk.IntegrationTypeHook:
			q += " AND integration_model.hook = true"
		case sdk.IntegrationTypeDeployment:
			q += " AND integration_model.deployment = true"
		}
	}

	query := gorpmapping.NewQuery(q)
	var pms integrationModelSlice

	if err := gorpmapping.GetAll(context.Background(), db, query, &pms, gorpmapping.GetOptions.WithDecryption); err != nil {
		return nil, err
	}

	var res []sdk.IntegrationModel
	for _, pm := range pms {
		isValid, err := gorpmapping.CheckSignature(pm, pm.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(context.Background(), "integration.LoadModel> model  %d data corrupted", pm.ID)
			continue
		}
		res = append(res, pm.IntegrationModel)
	}

	return res, nil
}

// LoadModel Load a integration model by its ID
func LoadModel(ctx context.Context, db gorp.SqlExecutor, modelID int64) (sdk.IntegrationModel, error) {
	query := gorpmapping.NewQuery("SELECT * from integration_model where id = $1").Args(modelID)
	return getModel(ctx, db, query)
}

func LoadModelWithClearPassword(ctx context.Context, db gorp.SqlExecutor, modelID int64) (sdk.IntegrationModel, error) {
	query := gorpmapping.NewQuery("SELECT * from integration_model where id = $1").Args(modelID)
	return getModelWithClearPassword(ctx, db, query)
}

// LoadModelByName Load a integration model by its name
func LoadModelByName(ctx context.Context, db gorp.SqlExecutor, name string) (sdk.IntegrationModel, error) {
	query := gorpmapping.NewQuery("SELECT * from integration_model where name = $1").Args(name)
	return getModel(ctx, db, query)
}

func LoadModelByNameWithClearPassword(ctx context.Context, db gorp.SqlExecutor, name string) (sdk.IntegrationModel, error) {
	query := gorpmapping.NewQuery("SELECT * from integration_model where name = $1").Args(name)
	return getModelWithClearPassword(ctx, db, query)
}

func getModel(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) (sdk.IntegrationModel, error) {
	m, err := getModelWithClearPassword(ctx, db, query)
	m.Blur()
	return m, err
}

func getModelWithClearPassword(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) (sdk.IntegrationModel, error) {
	var pm integrationModel

	found, err := gorpmapping.Get(ctx, db, query, &pm, gorpmapping.GetOptions.WithDecryption)
	if err != nil {
		return sdk.IntegrationModel{}, err
	}
	if !found {
		return sdk.IntegrationModel{}, sdk.WithStack(sdk.ErrNotFound)
	}

	isValid, err := gorpmapping.CheckSignature(pm, pm.Signature)
	if err != nil {
		return sdk.IntegrationModel{}, err
	}
	if !isValid {
		log.Error(ctx, "integration.LoadModelByName> model  %d data corrupted", pm.ID)
		return sdk.IntegrationModel{}, sdk.WithStack(sdk.ErrNotFound)
	}

	return pm.IntegrationModel, nil
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
func InsertModel(db gorpmapper.SqlExecutorWithTx, m *sdk.IntegrationModel) error {
	givenPublicConfig := m.PublicConfigurations.Clone()
	dbm := integrationModel{IntegrationModel: *m}
	if err := gorpmapping.InsertAndSign(context.Background(), db, &dbm); err != nil {
		return sdk.WrapError(err, "Unable to insert integration model %s", m.Name)
	}
	*m = dbm.IntegrationModel
	m.PublicConfigurations = givenPublicConfig
	m.PublicConfigurations.Blur()
	return nil
}

// UpdateModel updates a integration model in database
func UpdateModel(ctx context.Context, db gorpmapper.SqlExecutorWithTx, m *sdk.IntegrationModel) error {
	// reload the previous config to encuse we don't store placeholder
	var oldModel sdk.IntegrationModel

	givenPublicConfig := m.PublicConfigurations.Clone()
	for k := range givenPublicConfig {
		for kk, cfg := range givenPublicConfig[k] {
			if cfg.Type == sdk.IntegrationConfigTypePassword && cfg.Value == sdk.PasswordPlaceholder {
				if oldModel.ID == 0 {
					var err error
					oldModel, err = LoadModelWithClearPassword(ctx, db, m.ID)
					if err != nil {
						return err
					}
				}
				cfg.Value = ""
				if _, hasPublicConfig := oldModel.PublicConfigurations[k]; hasPublicConfig {
					if _, hasPublicConfigKey := oldModel.PublicConfigurations[k][kk]; hasPublicConfigKey {
						cfg.Value = oldModel.PublicConfigurations[k][kk].Value
					}
				}
			}
			givenPublicConfig[k][kk] = cfg
		}
	}

	m.PublicConfigurations = givenPublicConfig
	dbm := integrationModel{IntegrationModel: *m}
	if err := gorpmapping.UpdateAndSign(ctx, db, &dbm); err != nil {
		return sdk.WrapError(err, "Unable to update integration model %s", m.Name)
	}
	m.PublicConfigurations.Blur()
	return nil
}

// DeleteModel deletes a integration model in database
func DeleteModel(ctx context.Context, db gorp.SqlExecutor, id int64) error {
	m, err := LoadModel(ctx, db, id)
	if err != nil {
		return sdk.WrapError(err, "DeleteModel")
	}

	dbm := integrationModel{IntegrationModel: m}
	if _, err := db.Delete(&dbm); err != nil {
		return sdk.WrapError(err, "unable to delete model %s", m.Name)
	}

	return nil
}
