package hatchery

import (
  "context"
  "github.com/rockbears/log"

  "github.com/go-gorp/gorp"

  "github.com/ovh/cds/engine/api/database/gorpmapping"
  "github.com/ovh/cds/engine/gorpmapper"
  "github.com/ovh/cds/sdk"
)

func getHatchery(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) (*sdk.Hatchery, error) {
	var dbHatchery dbHatchery
	found, err := gorpmapping.Get(ctx, db, query, &dbHatchery)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.WrapError(sdk.ErrNotFound, "unable to get hatchery")
	}

	isValid, err := gorpmapping.CheckSignature(dbHatchery, dbHatchery.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "hatchery %s / %s data corrupted", dbHatchery.ID, dbHatchery.Name)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &dbHatchery.Hatchery, nil
}

func getAllHatcheries(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.Hatchery, error) {
	var res []dbHatchery
	if err := gorpmapping.GetAll(ctx, db, query, &res); err != nil {
		return nil, err
	}

	hatcheries := make([]sdk.Hatchery, 0, len(res))
	for _, r := range res {
		isValid, err := gorpmapping.CheckSignature(r, r.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "hatchery %d / %s data corrupted", r.ID, r.Name)
			continue
		}
		hatcheries = append(hatcheries, r.Hatchery)
	}
	return hatcheries, nil
}

func Insert(ctx context.Context, db gorpmapper.SqlExecutorWithTx, h *sdk.Hatchery) error {
	h.ID = sdk.UUID()
	dbData := &dbHatchery{Hatchery: *h}
	if err := gorpmapping.InsertAndSign(ctx, db, dbData); err != nil {
		return err
	}
	*h = dbData.Hatchery
	return nil
}

func Update(ctx context.Context, db gorpmapper.SqlExecutorWithTx, h *sdk.Hatchery) error {
	dbData := &dbHatchery{Hatchery: *h}
	if err := gorpmapping.UpdateAndSign(ctx, db, dbData); err != nil {
		return err
	}
	*h = dbData.Hatchery
	return nil
}

func LoadHatcheries(ctx context.Context, db gorp.SqlExecutor) ([]sdk.Hatchery, error) {
	query := gorpmapping.NewQuery(`SELECT hatchery.* FROM hatchery`)
	return getAllHatcheries(ctx, db, query)
}

func LoadHatcheryByName(ctx context.Context, db gorp.SqlExecutor, name string) (*sdk.Hatchery, error) {
	query := gorpmapping.NewQuery(`SELECT hatchery.* FROM hatchery WHERE hatchery.name = $1`).Args(name)
	return getHatchery(ctx, db, query)
}

func LoadHatcheryByID(ctx context.Context, db gorp.SqlExecutor, ID string) (*sdk.Hatchery, error) {
	query := gorpmapping.NewQuery(`SELECT hatchery.* FROM hatchery WHERE hatchery.id = $1`).Args(ID)
	return getHatchery(ctx, db, query)
}

func Delete(db gorpmapper.SqlExecutorWithTx, hatcheryID string) error {
	_, err := db.Exec("DELETE FROM hatchery WHERE id = $1", hatcheryID)
	return sdk.WrapError(err, "cannot delete hatchery %s", hatcheryID)
}
