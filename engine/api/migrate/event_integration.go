package migrate

import (
	"context"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"

	"github.com/go-gorp/gorp"
)

type KafkaConfig struct {
	Enabled         bool
	Broker          string
	Topic           string
	User            string
	Password        string
	MaxMessageBytes int
}

func AddPublicEventIntegration(ctx context.Context, store cache.Store, DBFunc func() *gorp.DbMap, kConfig KafkaConfig) error {
	if !kConfig.Enabled {
		return nil
	}

	db := DBFunc()
	exist, err := integration.ModelExists(db, "cds-kafka-public")
	if err != nil {
		return sdk.WrapError(err, "cannot check if model already exist")
	}
	if exist {
		return nil
	}
	model := sdk.IntegrationModel{
		Name:   "cds-kafka-public",
		Author: "CDS",
		Event:  true,
		Public: true,
		PublicConfigurations: map[string]sdk.IntegrationConfig{
			"cdsevents": sdk.IntegrationConfig{
				"broker url": sdk.IntegrationConfigValue{
					Type:  sdk.IntegrationConfigTypeString,
					Value: kConfig.Broker,
				},
				"topic": sdk.IntegrationConfigValue{
					Type:  sdk.IntegrationConfigTypeString,
					Value: kConfig.Topic,
				},
				"username": sdk.IntegrationConfigValue{
					Type:  sdk.IntegrationConfigTypeString,
					Value: kConfig.User,
				},
				"password": sdk.IntegrationConfigValue{
					Type:  sdk.IntegrationConfigTypePassword,
					Value: kConfig.Password,
				},
			},
		},
	}

	if err := integration.InsertModel(db, &model); err != nil {
		return sdk.WrapError(err, "cannot insert model cds-kafka-public")
	}

	if err := propagatePublicIntegrationModel(db, store, model); err != nil {
		return sdk.WrapError(err, "cannot propagate public integration model")
	}

	return nil
}

func propagatePublicIntegrationModel(db *gorp.DbMap, store cache.Store, m sdk.IntegrationModel) error {
	if !m.Public && len(m.PublicConfigurations) > 0 {
		return nil
	}

	projs, err := project.LoadAll(context.Background(), db, store, nil, project.LoadOptions.WithClearIntegrations)
	if err != nil {
		return sdk.WrapError(err, "Unable to retrieve all projects")
	}

	for _, p := range projs {
		tx, err := db.Begin()
		if err != nil {
			log.Error("propagatePublicIntegrationModel> error: %v", err)
			continue
		}
		if err := propagatePublicIntegrationModelOnProject(tx, store, m, p); err != nil {
			log.Error("propagatePublicIntegrationModel> error: %v", err)
			_ = tx.Rollback()
			continue
		}
		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "unable to commit")
		}
	}

	return nil
}

func propagatePublicIntegrationModelOnProject(db gorp.SqlExecutor, store cache.Store, m sdk.IntegrationModel, p sdk.Project) error {
	if !m.Public {
		return nil
	}

	for pfName, immutableCfg := range m.PublicConfigurations {
		cfg := immutableCfg.Clone()
		oldPP, _ := integration.LoadProjectIntegrationByName(db, p.Key, pfName, true)
		if oldPP.ID == 0 {
			pp := sdk.ProjectIntegration{
				Model:              m,
				IntegrationModelID: m.ID,
				Name:               pfName,
				Config:             cfg,
				ProjectID:          p.ID,
			}
			if err := integration.InsertIntegration(db, &pp); err != nil {
				return sdk.WrapError(err, "Unable to insert integration %s", pp.Name)
			}
			continue
		}

		pp := sdk.ProjectIntegration{
			ID:                 oldPP.ID,
			Model:              m,
			IntegrationModelID: m.ID,
			Name:               pfName,
			Config:             cfg,
			ProjectID:          p.ID,
		}
		oldPP.Config = m.DefaultConfig
		if err := integration.UpdateIntegration(db, pp); err != nil {
			return err
		}
	}
	return nil
}
