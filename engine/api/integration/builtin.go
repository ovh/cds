package integration

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

var (
	// BuiltinModels list available integration models
	BuiltinModels = []sdk.IntegrationModel{
		sdk.KafkaIntegration,
		sdk.RabbitMQIntegration,
		sdk.OpenstackIntegration,
		sdk.AWSIntegration,
		sdk.ArtifactManagerIntegration,
	}
)

// CreateBuiltinModels creates integrations models
func CreateBuiltinModels(ctx context.Context, db *gorp.DbMap) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "Unable to start transaction")
	}
	defer tx.Rollback() // nolint

	if _, err := tx.Exec("LOCK TABLE integration_model IN ACCESS EXCLUSIVE MODE"); err != nil {
		return sdk.WrapError(err, "Unable to lock table")
	}

	for i := range BuiltinModels {
		p := &BuiltinModels[i]
		ok, err := ModelExists(tx, p.Name)
		if err != nil {
			return sdk.WrapError(err, "CreateModels")
		}

		if !ok {
			log.Debug(ctx, "CreateBuiltinModels> inserting integration config: %s", p.Name)
			if err := InsertModel(tx, p); err != nil {
				return sdk.WrapError(err, "error on insert")
			}
		} else {
			log.Debug(ctx, "CreateBuiltinModels> updating integration config: %s", p.Name)
			oldM, err := LoadModelByName(ctx, tx, p.Name)
			if err != nil {
				return sdk.WrapError(err, "error on load")
			}
			p.ID = oldM.ID
			if err := UpdateModel(ctx, tx, p); err != nil {
				return sdk.WrapError(err, "error on update")
			}
		}
	}
	return sdk.WithStack(tx.Commit())
}
