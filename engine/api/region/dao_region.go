package region

import (
	"context"
	"github.com/lib/pq"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func Insert(ctx context.Context, db gorpmapper.SqlExecutorWithTx, region *sdk.Region) error {
	region.ID = sdk.UUID()
	dbData := &dbRegion{Region: *region}
	if err := gorpmapping.InsertAndSign(ctx, db, dbData); err != nil {
		return err
	}
	*region = dbData.Region
	return nil
}

func Delete(db gorpmapper.SqlExecutorWithTx, regionID string) error {
	_, err := db.Exec("DELETE FROM region WHERE id = $1", regionID)
	return sdk.WrapError(err, "cannot delete region %s", regionID)
}

func getRegion(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) (*sdk.Region, error) {
	var dbRegion dbRegion
	found, err := gorpmapping.Get(ctx, db, query, &dbRegion)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.WrapError(sdk.ErrNotFound, "unable to find region")
	}

	isValid, err := gorpmapping.CheckSignature(dbRegion, dbRegion.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "region %s / %s data corrupted", dbRegion.ID, dbRegion.Name)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &dbRegion.Region, nil
}

func getAllRegions(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.Region, error) {
	var res []dbRegion
	if err := gorpmapping.GetAll(ctx, db, query, &res); err != nil {
		return nil, err
	}

	regions := make([]sdk.Region, 0, len(res))
	for _, r := range res {
		isValid, err := gorpmapping.CheckSignature(r, r.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "region %d / %s data corrupted", r.ID, r.Name)
			continue
		}
		regions = append(regions, r.Region)
	}
	return regions, nil
}

func LoadAllRegions(ctx context.Context, db gorp.SqlExecutor) ([]sdk.Region, error) {
	query := gorpmapping.NewQuery(`SELECT region.* FROM region`)
	return getAllRegions(ctx, db, query)
}

func LoadRegionByName(ctx context.Context, db gorp.SqlExecutor, name string) (*sdk.Region, error) {
	query := gorpmapping.NewQuery(`SELECT region.* FROM region WHERE region.name = $1`).Args(name)
	return getRegion(ctx, db, query)
}

func LoadRegionByID(ctx context.Context, db gorp.SqlExecutor, ID string) (*sdk.Region, error) {
	query := gorpmapping.NewQuery(`SELECT region.* FROM region WHERE region.id = $1`).Args(ID)
	return getRegion(ctx, db, query)
}

func LoadRegionByIDs(ctx context.Context, db gorp.SqlExecutor, IDs []string) ([]sdk.Region, error) {
	query := gorpmapping.NewQuery(`SELECT region.* FROM region WHERE region.id = ANY($1)`).Args(pq.StringArray(IDs))
	return getAllRegions(ctx, db, query)
}
