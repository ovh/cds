package featureflipping

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gorpmapping"
)

func LoadAll(ctx context.Context, db gorp.SqlExecutor) ([]sdk.Feature, error) {
	query := gorpmapping.NewQuery("select * from feature_flipping")
	var result []sdk.Feature
	if err := gorpmapping.GetAll(ctx, db, query, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func LoadByName(ctx context.Context, db gorp.SqlExecutor, name string) (sdk.Feature, error) {
	query := gorpmapping.NewQuery("select * from feature_flipping where name = $1").Args(name).Limit(1)
	var f sdk.Feature
	found, err := gorpmapping.Get(ctx, db, query, &f)
	if err != nil {
		return sdk.Feature{}, err
	}
	if !found {
		return sdk.Feature{}, sdk.WithStack(sdk.ErrNotFound)
	}
	return f, nil
}

func Insert(ctx context.Context, db gorp.SqlExecutor, f *sdk.Feature) error {
	return gorpmapping.Insert(db, f)
}

func Update(ctx context.Context, db gorp.SqlExecutor, f *sdk.Feature) error {
	return gorpmapping.Update(db, f)
}

func Delete(ctx context.Context, db gorp.SqlExecutor, id int64) error {
	_, err := db.Exec("delete from feature_flipping where id = $1", id)
	if err != nil {
		return sdk.WithStack(err)
	}
	return nil
}
