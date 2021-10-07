package migrate

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/sdk"
)

func ArtifactoryIntegration(ctx context.Context, dbFunc func() *gorp.DbMap) error {
	tx, err := dbFunc().Begin()
	if err != nil {
		return sdk.WithStack(err)
	}

	defer tx.Rollback()

	model, err := integration.LoadModelByNameWithClearPassword(tx, "ArtifactManager")
	if err != nil {
		return sdk.WithStack(err)
	}

	// We just need to update the name, because bootstrap.InitializeDB
	// that runs after will update all the data based on this name
	model.Name = sdk.ArtifactoryIntegration.Name
	if err := integration.UpdateModel(tx, &model); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	return nil
}
