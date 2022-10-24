package hatchery

import (
	"context"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func insertHatcheryStatus(_ context.Context, db gorpmapper.SqlExecutorWithTx, s *sdk.HatcheryStatus) error {
	if err := gorpmapping.Insert(db, s); err != nil {
		return err
	}
	return nil
}

func updateHatcheryStatus(_ context.Context, db gorpmapper.SqlExecutorWithTx, s *sdk.HatcheryStatus) error {
	if err := gorpmapping.Update(db, s); err != nil {
		return err
	}
	return nil
}

func getHatcheryStatus(ctx context.Context, db gorpmapper.SqlExecutorWithTx, q gorpmapping.Query) (*sdk.HatcheryStatus, error) {
	var hatcheryStatus sdk.HatcheryStatus
	found, err := gorpmapping.Get(ctx, db, q, &hatcheryStatus)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &hatcheryStatus, nil
}

func loadHatcheryStatusByHatcheryID(ctx context.Context, db gorpmapper.SqlExecutorWithTx, hatcheryID string) (*sdk.HatcheryStatus, error) {
	query := gorpmapping.NewQuery("SELECT * from hatchery_status WHERE hatchery_id = $1").Args(hatcheryID)
	return getHatcheryStatus(ctx, db, query)
}

// UpsertStatus insert or update monitoring status
func UpsertStatus(ctx context.Context, db gorpmapper.SqlExecutorWithTx, hatcheryID string, s *sdk.HatcheryStatus) error {
	hatchStatus, err := loadHatcheryStatusByHatcheryID(ctx, db, hatcheryID)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return err
	}
	if hatchStatus == nil {
		return insertHatcheryStatus(ctx, db, s)
	}
	s.ID = hatchStatus.ID
	return updateHatcheryStatus(ctx, db, s)
}
